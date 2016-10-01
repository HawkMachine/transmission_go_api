package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/HawkMachine/transmission_go_api"
)

var (
	address  = flag.String("address", "", "Transmission address")
	username = flag.String("username", "", "Transmission username")
	password = flag.String("password", "", "Transmission password")
)

func main() {
	flag.Parse()
	t, err := transmission_go_api.New(*address, *username, *password)
	if err != nil {
		log.Fatalf("Failed to create Transmission client: %v", err)
	}
	torrents, err := t.ListAll()
	if err != nil {
		log.Fatalf("ListAll error: %v", err)
	}
	for _, torrent := range torrents {
		fmt.Printf("%s (%d) %s\n", torrent.Name, torrent.Id, torrent.Status)
	}
}
