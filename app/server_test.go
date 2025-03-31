package app

import "testing"

func TestServer(t *testing.T) {
	s := NewServer("0.0.0.0", 3388)
	err := s.Start()
	if err != nil {
		t.Fatal(err)
	}
}
