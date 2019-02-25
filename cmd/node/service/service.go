// Copyright The go-okchain Authors 2018,  All rights reserved.

package service

import (
	"errors"
	"fmt"
	"net"
)

import (
	logging "github.com/ok-chain/okchain/log"
	"github.com/ok-chain/okchain/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

var serviceLogger = logging.MustGetLogger("service")
var servers []*grpc.Server

func StopServices() {

	serviceLogger.Infof("Stop all services")

	for _, s := range servers {
		s.Stop()
	}

	servers = nil
}

var rpcMessageSize int = 32 * 1024 * 1024 //32MB

func StartService(listenAddr string, regserv ...func(*grpc.Server)) error {
	if "" == listenAddr {
		return errors.New("Listen address for service not specified")
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		grpclog.Fatalf("Failed to listen: %v", err)
		return err
	}

	var opts []grpc.ServerOption

	msgsize := rpcMessageSize
	opts = append(opts, grpc.MaxRecvMsgSize(msgsize), grpc.MaxSendMsgSize(msgsize))

	grpcServer := grpc.NewServer(opts...)

	servers = append(servers, grpcServer)

	for _, f := range regserv {
		f(grpcServer)
	}

	serviceLogger.Infof("Starting service with address=%s", listenAddr)

	// add gossip server
	//bootstrapPeers := []string{"localhost:15001"}
	//gossipInstance := discovery.NewGossipInstance(listenAddr, listenAddr, grpcServer, lis, bootstrapPeers,false)
	//gossipInstance := discovery.NewGossipInstance(listenAddr, listenAddr, grpcServer, lis, bootstrapPeers,true)
	//pb.RegisterGossipServer(grpcServer, gossipInstance)
	//gossip.RegisterGossipServer(grpcServer, gossipInstance)
	if grpcErr := grpcServer.Serve(lis); grpcErr != nil {
		return fmt.Errorf("grpc server exited with error: %s", grpcErr)
	} else {
		serviceLogger.Info("grpc server exited")
	}

	return nil
}

func CreateGrpcServer() *grpc.Server {
	var opts []grpc.ServerOption
	msgsize := rpcMessageSize
	opts = append(opts, grpc.MaxRecvMsgSize(msgsize), grpc.MaxSendMsgSize(msgsize))
	grpcServer := grpc.NewServer(opts...)
	servers = append(servers, grpcServer)
	return grpcServer
}

// accept
func StartGrpcServer(listenAddr string, grpcServer *grpc.Server) error {

	if "" == listenAddr {
		util.OkChainPanic("Listen address for service not specified")
	}

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		grpclog.Fatalf("Failed to listen: %v", err)
		return err
	}
	serviceLogger.Infof("Starting grpc service with address: %s", listenAddr)

	if grpcErr := grpcServer.Serve(lis); grpcErr != nil {
		return fmt.Errorf("grpc server exited with error: %s", grpcErr)
	} else {
		serviceLogger.Info("grpc server exited")
	}

	return nil
}
