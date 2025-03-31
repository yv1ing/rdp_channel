package main

import (
	"flag"
	"log"
	"rdp_channel/app"
)

func main() {
	mode := flag.String("mode", "server", "server or client")
	host := flag.String("host", "127.0.0.1", "server or client")
	port := flag.Int("port", 8080, "server or client")
	flag.Parse()

	var a app.App
	switch *mode {
	case "server":
		a = app.NewServer(*host, *port)
	case "client":
		a = app.NewClient(*host, *port)
	default:
		log.Fatal("[APP] invalid mode: " + *mode)
	}

	err := a.Start()
	if err != nil {
		log.Fatal(err)
	}
}
