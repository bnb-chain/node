package order

import (
	"hash/crc32"
)

type SymbolSelector interface {
	SelectSymbolsToMatch(roundOrders map[string][]string, height int64, matchAllSymbols bool) []string
}

var _ SymbolSelector = &BEP2SymbolSelector{}

type BEP2SymbolSelector struct{}

func (bss *BEP2SymbolSelector) SelectSymbolsToMatch(roundOrders map[string][]string, height int64, matchAllSymbols bool) []string {
	size := len(roundOrders)
	if size == 0 {
		return make([]string, 0)
	}
	symbolsToMatch := make([]string, 0, len(roundOrders))
	for symbol := range roundOrders {
		symbolsToMatch = append(symbolsToMatch, symbol)
	}
	return symbolsToMatch
}

type MiniSymbolSelector struct {
	symbolsHash          map[string]uint32 //mini token pairs -> hash value for Round-Robin
	roundSelectedSymbols []string          //mini token pairs to match in this round
}

var _ SymbolSelector = &MiniSymbolSelector{}

func (mss *MiniSymbolSelector) addSymbolHash(symbol string) {
	mss.symbolsHash[symbol] = crc32.ChecksumIEEE([]byte(symbol))
}

func (mss *MiniSymbolSelector) clearRoundMatchSymbol() {
	mss.roundSelectedSymbols = make([]string, 0)
}

func (mss *MiniSymbolSelector) SelectSymbolsToMatch(roundOrders map[string][]string, height int64, matchAllSymbols bool) []string {
	size := len(roundOrders)
	if size == 0 {
		return make([]string, 0)
	}
	symbolsToMatch := make([]string, 0, len(roundOrders))
	if matchAllSymbols {
		for symbol := range roundOrders {
			symbolsToMatch = append(symbolsToMatch, symbol)
		}
	} else {
		mss.selectMiniSymbolsToMatch(roundOrders, height, func(miniSymbols map[string]struct{}) {
			for symbol := range miniSymbols {
				symbolsToMatch = append(symbolsToMatch, symbol)
			}
		})
	}
	mss.roundSelectedSymbols = symbolsToMatch
	return symbolsToMatch
}

func (mss *MiniSymbolSelector) selectMiniSymbolsToMatch(roundOrders map[string][]string, height int64, postSelect func(map[string]struct{})) {
	symbolsToMatch := make(map[string]struct{}, 256)
	mss.selectActiveMiniSymbols(symbolsToMatch, roundOrders, defaultActiveMiniSymbolCount)
	mss.selectMiniSymbolsRoundRobin(symbolsToMatch, height, defaultMiniBlockMatchInterval)
	postSelect(symbolsToMatch)
}

func (mss *MiniSymbolSelector) selectActiveMiniSymbols(symbolsToMatch map[string]struct{}, roundOrdersMini map[string][]string, k int) {
	//use quick select to select top k symbols
	symbolOrderNumsSlice := make([]*SymbolWithOrderNumber, 0, len(roundOrdersMini))
	for symbol, orders := range roundOrdersMini {
		symbolOrderNumsSlice = append(symbolOrderNumsSlice, &SymbolWithOrderNumber{symbol, len(orders)})
	}
	topKSymbolOrderNums := findTopKLargest(symbolOrderNumsSlice, k)

	for _, selected := range topKSymbolOrderNums {
		symbolsToMatch[selected.symbol] = struct{}{}
	}
}

func (mss *MiniSymbolSelector) selectMiniSymbolsRoundRobin(symbolsToMatch map[string]struct{}, height int64, matchInterval int) {
	m := height % int64(matchInterval)
	for symbol, symbolHash := range mss.symbolsHash {
		if int64(symbolHash%uint32(matchInterval)) == m {
			symbolsToMatch[symbol] = struct{}{}
		}
	}
}
