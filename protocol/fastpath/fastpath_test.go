package fastpath

import (
	"net"
	"rdp_channel/protocol/tpkt"
	"testing"
)

func TestFastPath(t *testing.T) {
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
			continue
		}

		go func(conn net.Conn) {
			defer conn.Close()

			tpkt := tpkt.New(conn)
			fp := New(tpkt)

			dataLen, data, err := fp.Read()
			if err != nil {
				return
			}

			t.Logf("fp server read data(%d bytes): %q\n", dataLen, data)

			_, err = fp.Write([]byte("fp server hello"))
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
	fp := New(tpkt)

	_, err = fp.Write([]byte("fp client hello"))
	if err != nil {
		t.Logf("fp client write error: %s\n", err)
	}

	dataLen, data, err := fp.Read()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("fp client read data(%d bytes): %q\n", dataLen, data)
}
