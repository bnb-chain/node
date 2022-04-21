package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/koron/go-ssdp"
)

func main() {
	t := flag.String("t", ssdp.All, "search type")
	w := flag.Int("w", 1, "wait time")
	l := flag.String("l", "", "local address to listen")
	m := flag.String("m", "", "multicast address to send")
	v := flag.Bool("v", false, "verbose mode")
	h := flag.Bool("h", false, "show help")
	flag.Parse()
	if *h {
		flag.Usage()
		return
	}
	if *v {
		ssdp.Logger = log.New(os.Stderr, "[SSDP] ", log.LstdFlags)
	}
	if *m != "" {
		err := ssdp.SetMulticastSendAddrIPv4(*m)
		if err != nil {
			log.Fatal(err)
		}
	}
	list, err := ssdp.Search(*t, *w, *l)
	if err != nil {
		log.Fatal(err)
	}
	for i, srv := range list {
		//fmt.Printf("%d: %#v\n", i, srv)
		fmt.Printf("%d: %s %s\n", i, srv.Type, srv.Location)
	}
}
