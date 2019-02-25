// Copyright The go-okchain Authors 2018,  All rights reserved.

package node

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	backend "github.com/ok-chain/okchain/api/grpc_server"
	api "github.com/ok-chain/okchain/api/jsonrpc_server"
	"github.com/ok-chain/okchain/cmd/node/service"
	"github.com/ok-chain/okchain/config"
	"github.com/ok-chain/okchain/core/server"
	_ "github.com/ok-chain/okchain/core/server/consensus/blspbft"
	_ "github.com/ok-chain/okchain/core/server/role"
	_ "github.com/ok-chain/okchain/core/server/state"
	pb "github.com/ok-chain/okchain/protos"
	"github.com/ok-chain/okchain/rpc"
	"github.com/ok-chain/okchain/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

var chaincodeDevMode bool

func startCmd() *cobra.Command {
	// Set the flags on the node start command.
	flags := nodeStartCmd.Flags()
	flags.BoolVarP(&chaincodeDevMode, "peer-chaincodedev", "", false,
		"Whether peer in chaincode development mode")
	return nodeStartCmd
}

var nodeStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the node.",
	Long:  `Starts a node that interacts with the network.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return StartNode(nil)
	},
}

func StartNode(postrun func() error) error {
	fpath := util.CanonicalizePath(viper.GetString("peer.fileSystemPath"))
	if fpath != "" {
		util.MkdirIfNotExist(fpath)
		//err := log.LoggingFileInit(fpath)
		//if err != nil {
		//	logger.Warning("Could not open file for log")
		//}
	}

	var err error
	// Start the grpc server. Done in a goroutine so we can deploy the
	// genesis block if needed.
	serve := make(chan error)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		serve <- nil
	}()

	enablePprof := viper.GetBool("peer.enablePprof")

	debugUrl := viper.GetString("peer.debug.listenAddress")
	if len(debugUrl) > 0 && enablePprof {
		go func() {
			log.Println(http.ListenAndServe(debugUrl, nil))
		}()
	}

	grpcPort := viper.GetInt("peer.grpc.listenPort")
	peerServer := server.NewPeer()
	listenAddr := config.GetListenAddress()
	grpcServer := service.CreateGrpcServer()

	gossipPort := viper.GetInt("peer.gossip.listenPort")

	var gossipGrpcServer *grpc.Server

	if grpcPort == gossipPort {
		gossipGrpcServer = grpcServer
	}

	peerServer.Init(gossipPort, listenAddr, gossipGrpcServer)
	pb.RegisterPeerServer(grpcServer, peerServer)
	pb.RegisterBackendServer(grpcServer, backend.NewGrpcApiServer(peerServer))
	pb.RegisterInnerServer(grpcServer, peerServer)

	logger.Infof("peer.jsonrpcAddress: %s", viper.GetString("peer.jsonrpcAddress"))
	_, jsonrpc_http_server, err := rpc.StartHTTPEndpoint(viper.GetString("peer.jsonrpcAddress"), api.APIs(peerServer),
		[]string{"okchain", "account"}, []string{}, []string{"localhost"}, rpc.DefaultHTTPTimeouts)
	if err != nil {
		return err
	}
	_, jsonrpc_ipc_server, err := rpc.StartIPCEndpoint(viper.GetString("peer.ipcendpoint"), api.APIs(peerServer))
	if err != nil {
		return err
	}

	run := func() {
		acceptErr := service.StartGrpcServer(viper.GetString("peer.listenAddress"), grpcServer)
		if acceptErr != nil {
			acceptErr = fmt.Errorf("service exited with error: %s", acceptErr)
		} else {
			logger.Info("service exited")
		}
		serve <- acceptErr
	}

	// starting accept at peer.listenAddress
	go run()
	//peerServer.DsBlockChain().CurrentBlock().NumberU64()
	//if peerServer.DsBlockChain().CurrentBlock().NumberU64() > 0 {
	//	go func(peerServer *server.PeerServer) {
	//		time.Sleep(time.Second * 2)
	//		fmt.Println("start revert")
	//		revertPeerServerState(peerServer)
	//		time.Sleep(time.Second * 2)
	//	}(peerServer)
	//
	//}
	// Block until grpc server exits
	if err == nil {
		err = <-serve
	}
	service.StopServices()
	jsonrpc_http_server.Stop()
	jsonrpc_ipc_server.Stop()
	return err
}
