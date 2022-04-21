package main

import (
	"fmt"
	"os"

	multibase "github.com/multiformats/go-multibase"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("usage: %s <new-base> <multibase-str>...\n", os.Args[0])
		os.Exit(1)
	}

	var newBase multibase.Encoding
	if baseParam := os.Args[1]; len(baseParam) != 0 {
		newBase = multibase.Encoding(baseParam[0])
	} else {
		fmt.Fprintln(os.Stderr, "<new-base> is empty")
		os.Exit(1)
	}

	input := os.Args[2:]

	for _, strmbase := range input {
		_, data, err := multibase.Decode(strmbase)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while decoding: %s\n", err)
			os.Exit(1)
		}

		newCid, err := multibase.Encode(newBase, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while encoding: %s\n", err)
			os.Exit(1)
		}
		fmt.Println(newCid)
	}

}
