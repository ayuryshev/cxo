package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/rpc/server"
	"github.com/skycoin/cxo/skyobject"
)

const (
	// default listening address
	ADDRESS = "127.0.0.1"
	PORT    = 9987
)

// Fill down known hosts
var knownHosts = map[cipher.SHA256][]string{}

func getConfigs() (nc node.Config, rc server.Config) {
	// get defaults
	nc = node.NewConfig()
	rc = server.NewConfig()
	//
	flag.StringVar(&nc.Listen,
		"address",
		ADDRESS,
		"Address to listen on. Set to empty string for arbitrary assignment")
	flag.IntVar(&nc.MaxConnections,
		"max-conn",
		nc.MaxConnections,
		"Connection limits")
	flag.IntVar(&nc.MaxMessageSize,
		"max-msg",
		nc.MaxMessageSize,
		"Messages greater than length are rejected and the sender disconnected")
	flag.DurationVar(&nc.DialTimeout,
		"dial-tm",
		nc.DialTimeout,
		"Timeout is the timeout for dialing new connections. Use a timeout"+
			" of 0 to ignore timeout")
	flag.DurationVar(&nc.ReadTimeout,
		"read-tm",
		nc.ReadTimeout,
		"Timeout for reading from a connection. Set to 0 to default to the"+
			" system's timeout")
	flag.DurationVar(&nc.WriteTimeout,
		"write-tm",
		nc.WriteTimeout,
		"Timeout for writing to a connection. Set to 0 to default to the"+
			" system's timeout")
	flag.IntVar(&nc.WriteQueueSize,
		"write-queue",
		nc.WriteQueueSize,
		"Individual connections' send queue size. This should be increased"+
			" if send volume per connection is high, so as not to block")
	flag.IntVar(&nc.ReadQueueSize,
		"read-queue",
		nc.ReadQueueSize,
		"Individual connections' send queue size. This should be increased"+
			" if send volume per connection is high, so as not to block")

	flag.BoolVar(&nc.Debug,
		"debug",
		nc.Debug,
		"show debug logs")
	flag.StringVar(&nc.Name,
		"name",
		nc.Name,
		"name of the node")
	flag.DurationVar(&nc.PingInterval,
		"ping",
		nc.PingInterval,
		"ping interval (0 = disabled)")

	flag.BoolVar(&nc.RemoteClose,
		"remote-close",
		nc.RemoteClose,
		"allow close the node using RPC client")

	flag.BoolVar(&rc.Enable,
		"rpc",
		rc.Enable,
		"use rpc")
	flag.IntVar(&nc.RPCEvents,
		"rpc-events",
		nc.RPCEvents,
		"rpc events chan size")
	flag.StringVar(&rc.Address,
		"rpc-address",
		rc.Address,
		"address for rpc")
	flag.IntVar(&rc.Max,
		"rpc-max",
		rc.Max,
		"maximum rpc-connections")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <flags> [known hosts]\n", os.Args[0])
		flag.PrintDefaults()
	}

	var help bool
	flag.BoolVar(&help, "h", false, "show help")

	flag.Parse()

	if help {
		fmt.Printf("Usage: %s <flags> [subscribe to]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}

	for _, sb := range flag.Args() { // subscribe
		pk, err := cipher.PubKeyFromHex(sb)
		if err != nil {
			log.Fatalf("malformed public key to subscribe to:\n"+
				"  %q\n"+
				"  %v\n", sb, err)
		}
		nc.Subscribe = append(nc.Subscribe, pk)
	}

	return
}

func waitInterrupt(quit <-chan struct{}) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	select {
	case <-sig:
	case <-quit:
	}
}

func main() {

	// exit code
	var code int
	defer func() { os.Exit(code) }()

	var err error

	var (
		db  *data.DB
		so  *skyobject.Container
		n   *node.Node
		rpc *server.Server

		nc node.Config
		rc server.Config
	)

	db = data.NewDB()

	so = skyobject.NewContainer(db)

	//
	// Get configurations from flags
	//

	nc, rc = getConfigs() // node config, rpc config

	//
	// Node
	//

	n = node.NewNode(nc, db, so)

	// can panic
	n.Start()
	defer n.Close()

	//
	// RPC
	//

	if rc.Enable {
		// TODO: add RPC control to skyobject
		rpc = server.NewServer(rc, n) // , so)
		if err = rpc.Start(); err != nil {
			fmt.Fprintln(os.Stderr, "error starting RPC:", err)
			code = 1
			return
		}
		defer rpc.Close()
	}

	// waiting for SIGING or termination using RPC

	waitInterrupt(n.Quiting())

}
