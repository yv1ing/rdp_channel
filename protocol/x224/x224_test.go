package x224

import (
	"net"
	"rdp_channel/protocol/tpkt"
	"testing"
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

			transport := tpkt.New(conn)
			x224 := New(transport)

			_, packet, err := x224.Read()
			if err != nil {
				t.Fatal(err)
			}

			err = x224.handleConnectionRequest(packet)
			if err != nil {
				t.Fatal(err)
			}
		}(conn)
	}
}

func runClient(t *testing.T) {
	conn, err := net.Dial("tcp", "127.0.0.1:3388")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	transport := tpkt.New(conn)
	x224 := New(transport)

	err = x224.ConnectToServer()
	if err != nil {
		t.Fatal(err)
	}
}
