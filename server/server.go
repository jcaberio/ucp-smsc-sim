// Package server provides a UCP server implementation.
package server

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/jcaberio/ucp-smsc-sim/ucp"
	"github.com/jcaberio/ucp-smsc-sim/util"
)

var cl = &connList{
	conns: make([]net.Conn, 0),
}

// Start starts the UCP server with the following configuration.
func Start(config util.Config) {
	port := fmt.Sprintf(":%d", config.Port)
	ln, err := net.Listen("tcp", port)

	if err != nil {
		log.Println(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
		}
		go handleConnection(conn, config)

	}
}

func handleConnection(conn net.Conn, config util.Config) {
	for {

		pdu, err := ucp.New(conn)
		if err != nil {
			return
		}
		cl.add(conn)
		select {
		case deliverSM := <-ucp.DeliverSMCh:
			cl.Lock()
			for _, c := range cl.conns {
				if c != nil {
					c.Write(deliverSM.Result())
				}
			}
			cl.Unlock()

		default:
			pdu.Decode(config)
			pdu.Stats()

		}
	}
}

type connList struct {
	sync.Mutex
	conns []net.Conn
}

func (c *connList) add(connToAdd net.Conn) {
	c.Lock()
	defer c.Unlock()
	for _, conn := range c.conns {
		if conn.RemoteAddr().String() == connToAdd.RemoteAddr().String() {
			return
		}
	}
	c.conns = append(c.conns, connToAdd)
}
