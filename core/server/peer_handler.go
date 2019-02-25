// Copyright The go-okchain Authors 2018,  All rights reserved.

package server

import (
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/looplab/fsm"
	pb "github.com/ok-chain/okchain/protos"
	"github.com/ok-chain/okchain/util"
)

// Handler peer handler implementation.
type P2PHandler struct {
	chatMutex      sync.Mutex
	ToPeerEndpoint *pb.PeerEndpoint
	shardingId     string
	Coordinator    *PeerServer
	ChatStream     ChatStream
	doneChan       chan struct{}
	FSM            *fsm.FSM
	msgMutex       sync.Mutex

	registered bool
	Active     bool
}

func newP2PHandler(coord *PeerServer, stream ChatStream, active bool) (IP2PHandler, error) {
	d := &P2PHandler{
		ChatStream:  stream,
		Coordinator: coord,
		Active:      active,
	}
	d.doneChan = make(chan struct{})

	d.FSM = fsm.NewFSM(
		"created",
		fsm.Events{
			{Name: pb.Message_Peer_Hello.String(), Src: []string{"created"}, Dst: "established"},
		},
		fsm.Callbacks{
			"enter_created":     func(e *fsm.Event) { d.enter_created(e) },
			"leave_created":     func(e *fsm.Event) { d.leave_created(e) },
			"enter_established": func(e *fsm.Event) { d.enter_established(e) },
			"leave_established": func(e *fsm.Event) { d.leave_established(e) },

			"before_" + pb.Message_Peer_Hello.String(): func(e *fsm.Event) { d.onPeer_Hello(e) },
			"after_" + pb.Message_Peer_Hello.String(): func(e *fsm.Event) {
				d.msgMutex.Lock()
				d.onConsensusCheck(e)
				d.msgMutex.Unlock()
			},
		},
	)

	if d.Active {
		helloMessage, err := d.Coordinator.NewOpenchainDiscoveryHello()
		if err != nil {
			return nil, fmt.Errorf("error getting new HelloMessage: %s", err)
		}

		if err := d.SendMessage(helloMessage); err != nil {
			return nil, fmt.Errorf("error creating new peer Handler, error returned sending %s: %s",
				pb.Message_Peer_Hello, err)
		}
	}
	return d, nil
}

func (d *P2PHandler) HandleMessage(msg *pb.Message) error {
	typeStr := msg.Type.String()
	var from string
	if d.ToPeerEndpoint != nil {
		from = d.ToPeerEndpoint.Id.Name
		peerLogger.Debugf("%s: >>>--- recv msg<%s> from peer <%s>", util.GId, typeStr, from)
	} else {
		peerLogger.Debugf("%s: >>>--- recv msg<%s>", util.GId, typeStr)
	}

	if msg.Type == pb.Message_Peer_Hello {
		err := d.FSM.Event(typeStr, msg)
		if err != nil {
			if _, ok := err.(*fsm.NoTransitionError); !ok {
				return fmt.Errorf("peer FSM failed while handling message (%s): current state: %s, error: %s",
					msg.Type.String(), d.FSM.Current(), err)
			}
		}
	} else {
		r := d.Coordinator.GetCurrentRole()
		r.ProcessMsg(msg, d.ToPeerEndpoint)
	}

	return nil
}

func (d *P2PHandler) deregister() error {
	var err error
	if d.registered {
		err = d.Coordinator.DeregisterPHHandler(d)
		//doneChan is created and waiting for registered handlers only
		//d.doneChan <- struct{}{}
		d.registered = false
	}
	return err
}

// To return the PeerEndpoint this Handler is connected to.
func (d *P2PHandler) To() (pb.PeerEndpoint, error) {
	if d.ToPeerEndpoint == nil {
		return pb.PeerEndpoint{}, fmt.Errorf("no peer endpoint for handler")
	}
	return *(d.ToPeerEndpoint), nil
}

// Stop stops this handler, which will trigger the Deregister from the MessageHandlerCoordinator.
func (d *P2PHandler) Stop(active bool) error {
	// Deregister the handler
	peerLogger.Debugf("%s: Stopping handler to <%s>, active<%t>, registered<%t>",
		util.GId, d.ToPeerEndpoint, active, d.registered)

	err := d.deregister()
	if err != nil {
		return fmt.Errorf("error stopping MessageHandler: %s", err)
	}
	return nil
}

func (d *P2PHandler) onPeer_Hello(e *fsm.Event) {
	if _, ok := e.Args[0].(*pb.Message); !ok {
		e.Cancel(fmt.Errorf("received unexpected message type"))
		return
	}

	msg := e.Args[0].(*pb.Message)
	helloMessageIn := &pb.PeerEndpoint{}
	err := proto.Unmarshal(msg.Payload, helloMessageIn)
	if err != nil {
		e.Cancel(fmt.Errorf("error unmarshalling HelloMessage: %s", err))
		return
	}

	// Store the PeerEndpoint
	d.ToPeerEndpoint = helloMessageIn
	d.Coordinator.commNodeInfo.Store(helloMessageIn.Id.Name, helloMessageIn)

	peerLogger.Debugf("%s: received %s from endpoint=%s", util.GId, e.Event, helloMessageIn)

	if !d.Active {
		// send hello message to peer
		helloMessageOut, err := d.Coordinator.NewOpenchainDiscoveryHello()
		if err != nil {
			e.Cancel(fmt.Errorf("error getting new HelloMessage: %s", err))
			return
		}
		if err := d.SendMessage(helloMessageOut); err != nil {
			e.Cancel(fmt.Errorf("error sending response to %s:  %s", e.Event, err))
			return
		}
	}
	// Register
	err = d.Coordinator.RegisterPHHandler(d)
	if err != nil {
		e.Cancel(err)
	} else {
		d.registered = true
	}
}

func (d *P2PHandler) Registered() bool {
	return d.registered
}

// SendMessage sends a message to the remote PEER through the stream
func (d *P2PHandler) SendMessage(msg *pb.Message) error {
	//make sure Sends are serialized. Also make sure everyone uses SendMessage
	//instead of calling Send directly on the grpc stream
	d.chatMutex.Lock()
	defer d.chatMutex.Unlock()

	if d.ToPeerEndpoint != nil && d.ToPeerEndpoint.Id != nil {
		peerLogger.Debugf("%s: <<<--- sending msg<%s> to peer <%s>", util.GId,
			msg.Type.String(), d.ToPeerEndpoint.Id)
	}

	err := d.ChatStream.Send(msg)
	if err != nil {
		return fmt.Errorf("error sending message through ChatStream: %s", err)
	}
	return nil
}

func (d *P2PHandler) sendMessageToNewConnectedPeer() {
	if _, ok := d.Coordinator.peerHandlerMap.m[d.ToPeerEndpoint.Id.Name]; ok {
		if d.Coordinator.msg.retStr != "consensus" && d.Coordinator.msg.retStr != "pow" {
			return
		}
		if d.Coordinator.msg.data == nil {
			return
		}
		peerLogger.Debugf("new connection to %s created", d.ToPeerEndpoint.Id.Name)
		err := d.Coordinator.peerHandlerMap.m[d.ToPeerEndpoint.Id.Name].SendMessage(d.Coordinator.msg.data)
		if err != nil {
			peerLogger.Errorf("send message to peer %s failed, error: %s", d.ToPeerEndpoint.Id.Name, err.Error())
		}
		peerLogger.Debugf("waitgroup delete one!!!!")
		d.Coordinator.msg.wg.Done()

		if err != nil {
			d.Coordinator.msg.retStr = err.Error()
		} else {
			d.Coordinator.msg.retStr = ""
		}

	} else {
		peerLogger.Errorf("chat with peer %s failed", d.ToPeerEndpoint.Id.Name)
	}
}

func (d *P2PHandler) leave_established(e *fsm.Event) {
	//peerLogger.Debugf("the Peer's bi-directional stream to %s is %s, from event %s\n", d.ToPeerEndpoint, e.Dst, e.Event)
}

func (d *P2PHandler) enter_established(e *fsm.Event) {
	//peerLogger.Debugf("the Peer's bi-directional stream to %s is %s, from event %s\n", d.ToPeerEndpoint, e.Dst, e.Event)
}

func (d *P2PHandler) leave_created(e *fsm.Event) {
	//peerLogger.Debugf("the Peer's bi-directional stream to %s is %s, from event %s\n", d.ToPeerEndpoint, e.Dst, e.Event)
}

func (d *P2PHandler) enter_created(e *fsm.Event) {
	//peerLogger.Debugf("the Peer's bi-directional stream to %s is %s, from event %s\n", d.ToPeerEndpoint, e.Dst, e.Event)
}

func (d *P2PHandler) onConsensusCheck(e *fsm.Event) {
	if d.Coordinator.msg.data == nil {
		return
	}
	if d.Coordinator.msg.retStr != "consensus" && d.Coordinator.msg.retStr != "pow" {
		return
	}

	d.sendMessageToNewConnectedPeer()
}
