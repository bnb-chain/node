package main

import (
	"fmt"
	"os"

	"github.com/mr-tron/base58"
)

func main() {

	exampleBase58Encoded := []string{
		"1QCaxc8hutpdZ62iKZsn1TCG3nh7uPZojq",
		"1DhRmSGnhPjUaVPAj48zgPV9e2oRhAQFUb",
		"17LN2oPYRYsXS9TdYdXCCDvF2FegshLDU2",
		"14h2bDLZSuvRFhUL45VjPHJcW667mmRAAn",
	}

	// If a base58 string is on the command line, then use that instead of the 4 exampels above.
	if len(os.Args) > 1 {
		exampleBase58Encoded = os.Args[1:]
	}

	for _, vv := range exampleBase58Encoded {
		num, err := base58.Decode(vv)
		if err != nil {
			fmt.Printf("Demo %s, got error %s\n", vv, err)
			continue
		}
		chk := base58.Encode(num)
		if vv == string(chk) {
			fmt.Printf("Successfully decoded then re-encoded %s\n", vv)
		} else {
			fmt.Printf("Failed on %s\n", vv)
		}
	}
}
