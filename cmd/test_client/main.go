package main

import (
	"log"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)
	log.Printf("----- start delist -----")
	err := Staking()
	if err != nil {
		log.Printf("%+v\n", err)
	}
	log.Printf("----- end delist -----")
}
