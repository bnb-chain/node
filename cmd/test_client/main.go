package main

import (
	"log"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)
	log.Printf("----- start test -----")
	err := ChangeParameterViaGov()
	if err != nil {
		log.Printf("%+v\n", err)
		panic(err)
	}
	err = Staking()
	if err != nil {
		log.Printf("%+v\n", err)
		panic(err)
	}
	log.Printf("----- end test -----")
}
