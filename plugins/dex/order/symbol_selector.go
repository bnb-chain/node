package order

import (
	"hash/crc32"
)

type SymbolSelector interface {
	SelectSymbolsToMatch(roundOrders map[string][]string, height, timestamp int64, matchAllSymbols bool) []string
	AddSymbolHash(symbol string)
	GetRoundMatchSymbol() *[]string
	SetRoundMatchSymbol([]string)
}

type BEP2SymbolSelector struct {
}

var _ SymbolSelector = &BEP2SymbolSelector{}

func (bss *BEP2SymbolSelector) AddSymbolHash(symbol string) {
	panic("unsupported method")
}

func (bss *BEP2SymbolSelector) SetRoundMatchSymbol([]string) {
	panic("unsupported method")
}

func (bss *BEP2SymbolSelector) GetRoundMatchSymbol() *[]string {
	panic("unsupported method")
}

func (bss *BEP2SymbolSelector) SelectSymbolsToMatch(roundOrders map[string][]string, height, timestamp int64, matchAllSymbols bool) []string {
	size := len(roundOrders)
	if size == 0 {
		return make([]string, 0)
	}
	symbolsToMatch := make([]string, 0, 256)
	for symbol := range roundOrders {
		symbolsToMatch = append(symbolsToMatch, symbol)
	}
	return symbolsToMatch
}

type MiniSymbolSelector struct {
	miniSymbolsHash  map[string]uint32 //mini token pairs -> hash value for Round-Robin
	roundMiniSymbols []string          //mini token pairs to match in this round
}

var _ SymbolSelector = &MiniSymbolSelector{
	make(map[string]uint32),
	make([]string, 0),
}


func (mss *MiniSymbolSelector) GetRoundMatchSymbol() *[]string {
	return &mss.roundMiniSymbols
}

func (mss *MiniSymbolSelector) AddSymbolHash(symbol string) {
	mss.miniSymbolsHash[symbol] = crc32.ChecksumIEEE([]byte(symbol))
}

func (mss *MiniSymbolSelector) SetRoundMatchSymbol(symbols []string) {
	mss.roundMiniSymbols = symbols
}

func (mss *MiniSymbolSelector) SelectSymbolsToMatch(roundOrders map[string][]string, height, timestamp int64, matchAllSymbols bool) []string {
	size := len(roundOrders)
	if size == 0 {
		return make([]string, 0)
	}
	symbolsToMatch := make([]string, 0, 256)
	if matchAllSymbols {
		for symbol := range roundOrders {
			symbolsToMatch = append(symbolsToMatch, symbol)
		}
	} else {
		selectMiniSymbolsToMatch(roundOrders, mss.miniSymbolsHash, height, func(miniSymbols map[string]struct{}) {
			for symbol := range miniSymbols {
				symbolsToMatch = append(symbolsToMatch, symbol)
			}
		})
	}
	mss.roundMiniSymbols = symbolsToMatch
	return symbolsToMatch
}

func selectMiniSymbolsToMatch(roundOrders map[string][]string, miniSymbolsHash map[string]uint32, height int64, postSelect func(map[string]struct{})) {
	symbolsToMatch := make(map[string]struct{}, 256)
	selectActiveMiniSymbols(&symbolsToMatch, &roundOrders, defaultActiveMiniSymbolCount)
	selectMiniSymbolsRoundRobin(&symbolsToMatch, &miniSymbolsHash, height)
	postSelect(symbolsToMatch)
}

func selectActiveMiniSymbols(symbolsToMatch *map[string]struct{}, roundOrdersMini *map[string][]string, k int) {
	//use quick select to select top k symbols
	symbolOrderNumsSlice := make([]*SymbolWithOrderNumber, 0, len(*roundOrdersMini))
	for symbol, orders := range *roundOrdersMini {
		symbolOrderNumsSlice = append(symbolOrderNumsSlice, &SymbolWithOrderNumber{symbol, len(orders)})
	}
	topKSymbolOrderNums := findTopKLargest(symbolOrderNumsSlice, k)

	for _, selected := range topKSymbolOrderNums {
		(*symbolsToMatch)[selected.symbol] = struct{}{}
	}
}

func selectMiniSymbolsRoundRobin(symbolsToMatch *map[string]struct{}, miniSymbolsHash *map[string]uint32, height int64) {
	m := height % defaultMiniBlockMatchInterval
	for symbol, symbolHash := range *miniSymbolsHash {
		if int64(symbolHash%defaultMiniBlockMatchInterval) == m {
			(*symbolsToMatch)[symbol] = struct{}{}
		}
	}
}
