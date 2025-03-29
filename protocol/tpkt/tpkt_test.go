package tpkt

import (
	"net"
	"testing"
)

func TestTpkt(t *testing.T) {
	go runServer(t)
	runClient(t)
}

func runServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:3388")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	t.Logf("tpkt server listening at %s\n", listener.Addr())

	for {
		conn, err := listener.Accept()
		if err != nil {
			t.Logf("tpkt server accept error: %s\n", err)
			continue
		}

		t.Logf("tpkt server accepted connection from %s\n", conn.RemoteAddr())
		go func(conn net.Conn) {
			defer conn.Close()

			tpkt := New(conn)
			dataLen, data, err := tpkt.Read()
			if err != nil {
				t.Logf("tpkt server read error: %s\n", err)
				return
			}

			t.Logf("tpkt server read data(%d): %q\n", dataLen, data)

			_, err = tpkt.Write([]byte("tpkt server hello"))
			if err != nil {
				t.Logf("tpkt server write error: %s\n", err)
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

	tpkt := New(conn)

	_, err = tpkt.Write([]byte("tpkt client hello"))
	if err != nil {
		t.Logf("tpkt client write error: %s\n", err)
	}

	dataLen, data, err := tpkt.Read()
	if err != nil {
		t.Logf("tpkt client read error: %s\n", err)
	}
	t.Logf("tpkt client read data(%d): %q\n", dataLen, data)
}
