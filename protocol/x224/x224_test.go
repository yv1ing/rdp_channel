package x224

import (
	"fmt"
	"net"
	"rdp_channel/protocol/tpkt"
	"testing"
	"time"
)

func TestX224(t *testing.T) {
	go runServer(t)
	runClient(t)
}

func runServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:3388")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			t.Fatal(err)
		}

		go func(conn net.Conn) {
			defer conn.Close()

			tpkt := tpkt.New(conn)
			x224 := New(tpkt)

			x224.OnData(func(bytes []byte) {
				fmt.Printf("server received: %s\n", string(bytes))
				x224.Write([]byte("yes! server hear!"))
			})

			x224.serverHandleClientMessage()
		}(conn)
	}
}

func runClient(t *testing.T) {
	conn, err := net.Dial("tcp", "127.0.0.1:3388")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	tpkt := tpkt.New(conn)
	x224 := New(tpkt)

	x224.ConnectToServer()
	x224.OnData(func(bytes []byte) {
		fmt.Printf("client received: %s\n", string(bytes))
	})

	for {
		time.Sleep(1 * time.Second)
		x224.Write([]byte("this is client!"))
	}
}
