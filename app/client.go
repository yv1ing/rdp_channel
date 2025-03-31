package app

import (
	"fmt"
	"log"
	"net"
	"rdp_channel/protocol"
	"time"
)

type Client struct {
	Host string
	Port int
}

func NewClient(host string, port int) Client {
	return Client{host, port}
}

func (client Client) Start() error {
	addr := fmt.Sprintf("%s:%d", client.Host, client.Port)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Println("[Client] connected to " + addr)

	tpkt := protocol.NewTPKT(conn)
	//fast := protocol.NewFastPath(conn)
	//x224 := protocol.NewX224(conn)

	for {
		time.Sleep(1 * time.Second)
		err = tpkt.Write([]byte("This is a test message."))
	}
}
