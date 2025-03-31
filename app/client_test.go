package app

import "testing"

func TestClient(t *testing.T) {
	c := NewClient("127.0.0.1", 3388)
	err := c.Start()
	if err != nil {
		t.Fatal(err)
	}
}
