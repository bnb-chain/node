package main

import (
	"log"
)

func main() {
	var err error
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)
	log.Printf("----- start test -----")
	err = ChangeParameterViaGov()
	if err != nil {
		log.Printf("%+v\n", err)
		panic(err)
	}
	err = Staking()
	if err != nil {
		log.Printf("%+v\n", err)
		panic(err)
	}
	//err = TestFee()
	//if err != nil {
	//	log.Printf("TestFee failed: %v\n", err)
	//}
	log.Printf("----- end test -----")
}
