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
	list     = flag.Bool("list", false, "List")
	start    = flag.Int64("start", -1, "Start")
	startNow = flag.Int64("startnow", -1, "Start Now")
	stop     = flag.Int64("stop", -1, "Stop")
	remove   = flag.Int64("remove", -1, "Remove")
)

func main() {
	flag.Parse()
	t, err := transmission_go_api.New(*address, *username, *password)
	if err != nil {
		log.Fatalf("Failed to create Transmission client: %v", err)
	}
	if *list {
		torrents, err := t.ListAll()
		if err != nil {
			log.Fatalf("ListAll error: %v", err)
		}
		for _, torrent := range torrents {
			fmt.Printf("%d: (Status %d) (Done: %.2f) %s\n", torrent.Id, torrent.Status, torrent.PercentDone*100, torrent.Name)
		}
	} else if *start != -1 {
		err := t.Start([]int64{*start})
		if err != nil {
			log.Fatalf("ListAll error: %v", err)
		}
	} else if *startNow != -1 {
		err := t.StartNow([]int64{*startNow})
		if err != nil {
			log.Fatalf("StartNow error: %v", err)
		}
	} else if *stop != -1 {
		err := t.Stop([]int64{*stop})
		if err != nil {
			log.Fatalf("ListAll error: %v", err)
		}
	} else if *remove != -1 {
		err := t.Remove([]int64{*remove})
		if err != nil {
			log.Fatalf("Remove error: %v", err)
		}
	}
}
