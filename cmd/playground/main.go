package main

import (
	"fmt"
)

func main() {
	//groupedTrades := cmap.New()
	//
	//if tmp, exists := groupedTrades.Get("42"); exists {
	//	groupedByBid := tmp.(map[string]int)
	//	if _, exists := groupedByBid["84"]; !exists {
	//		groupedByBid["84"] = 84
	//	}
	//} else {
	//	groupedByBid := make(map[string]int)
	//	groupedTrades.Set("42", groupedByBid)
	//	groupedByBid["84"] = 84
	//}
	//
	//tmp, _ := groupedTrades.Get("42")
	//realMap := tmp.(map[string]int)
	//fmt.Printf("%d\n", realMap["84"])


	groupedTrades := make(map[string]map[string]int)

	if tmp, exists := groupedTrades["42"]; exists {
		if _, exists := tmp["84"]; !exists {
			tmp["84"] = 84
		}
	} else {
		tmp := make(map[string]int)
		tmp["84"] = 84
	}

	tmp, _ := groupedTrades["42"]["84"]
	fmt.Printf("%d\n", tmp)
}

