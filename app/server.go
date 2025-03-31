package app

import (
	"fmt"
	"log"
	"net"
	"rdp_channel/protocol"
)

type Server struct {
	Host string
	Port int
}

func NewServer(host string, port int) Server {
	return Server{host, port}
}

func (server Server) Start() error {
	addr := fmt.Sprintf("%s:%d", server.Host, server.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	log.Println("[SERVER] listening on " + addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	log.Println("[SERVER] new connection from " + conn.RemoteAddr().String())

	tpkt := protocol.NewTPKT(conn)

	//fast := protocol.NewFastPath(conn)
	//x224 := protocol.NewX224(conn)
	for {
		payload, err := tpkt.Read()
		if err != nil {
			continue
		}

		log.Println("[SERVER] received payload: " + string(payload))
	}
}
