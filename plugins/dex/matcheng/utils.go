package matcheng

func compareBuy(p1 int64, p2 int64) int {
	d := (p2 - p1)
	switch {
	case d >= PRECISION:
		return -1
	case d <= -PRECISION:
		return 1
	default:
		return 0
	}
}

func compareSell(p1 int64, p2 int64) int {
	return -compareBuy(p1, p2)
}

func mergeLevels(buyLevels []PriceLevel, sellLevels []PriceLevel, overlapped *[]OverLappedLevel) {
	*overlapped = (*overlapped)[:0]
	var i, j, bN int = 0, len(sellLevels) - 1, len(buyLevels)
	for i < bN && j >= 0 {
		b, s := buyLevels[i].Price, sellLevels[j].Price
		switch compareBuy(b, s) {
		case 0:
			*overlapped = append(*overlapped,
				OverLappedLevel{Price: b, BuyOrders: buyLevels[i].Orders,
					SellOrders: sellLevels[j].Orders})
			i++
			j--
		case -1:
			*overlapped = append(*overlapped, OverLappedLevel{Price: s,
				SellOrders: sellLevels[j].Orders})
			j--
		case 1:
			*overlapped = append(*overlapped, OverLappedLevel{Price: b,
				BuyOrders: buyLevels[i].Orders})
			i++
		}
	}
	for i < bN {
		b := buyLevels[i].Price
		*overlapped = append(*overlapped, OverLappedLevel{Price: b,
			BuyOrders: buyLevels[i].Orders})
		i++
	}
	for j >= 0 {
		s := sellLevels[j].Price
		*overlapped = append(*overlapped, OverLappedLevel{Price: s,
			SellOrders: sellLevels[j].Orders})
		j--
	}
}
