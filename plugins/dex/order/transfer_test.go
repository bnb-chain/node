package order

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTradeTransfers_Sort(t *testing.T) {
	e := TradeTransfers{
		{inAsset: "ABC", outAsset: "BNB", Oid: "1"},
		{inAsset: "ABC", outAsset: "BTC", Oid: "2"},
		{inAsset: "XYZ", outAsset: "BTC", Oid: "3"},
		{inAsset: "XYZ", outAsset: "BNB", Oid: "4"},
		{inAsset: "ABC", outAsset: "XYZ", Oid: "5"},
		{inAsset: "BTC", outAsset: "BNB", Oid: "6"},
		{inAsset: "BNB", outAsset: "BTC", Oid: "7"},
		{inAsset: "BNB", outAsset: "ABC", Oid: "8"},
		{inAsset: "ABC", outAsset: "BNB", Oid: "9"},
		{inAsset: "ABC", outAsset: "BTC", Oid: "10"},
	}
	e.Sort()
	require.Equal(t, TradeTransfers{
		{inAsset: "BNB", outAsset: "ABC", Oid: "8"},
		{inAsset: "BNB", outAsset: "BTC", Oid: "7"},
		{inAsset: "ABC", outAsset: "BNB", Oid: "1"},
		{inAsset: "ABC", outAsset: "BNB", Oid: "9"},
		{inAsset: "BTC", outAsset: "BNB", Oid: "6"},
		{inAsset: "XYZ", outAsset: "BNB", Oid: "4"},
		{inAsset: "ABC", outAsset: "BTC", Oid: "2"},
		{inAsset: "ABC", outAsset: "BTC", Oid: "10"},
		{inAsset: "ABC", outAsset: "XYZ", Oid: "5"},
		{inAsset: "XYZ", outAsset: "BTC", Oid: "3"},
	}, e)
}

func TestExpireTransfers_Sort(t *testing.T) {
	e := ExpireTransfers{
		{inAsset: "ABC", Symbol: "ABC_BNB", Oid: "1"},
		{inAsset: "ABC", Symbol: "ABC_BTC", Oid: "2"},
		{inAsset: "XYZ", Symbol: "XYZ_BTC", Oid: "3"},
		{inAsset: "XYZ", Symbol: "XYZ_BNB", Oid: "4"},
		{inAsset: "ABC", Symbol: "ABC_XYZ", Oid: "5"},
		{inAsset: "BTC", Symbol: "BNB_BTC", Oid: "6"},
		{inAsset: "BNB", Symbol: "BNB_BTC", Oid: "7"},
		{inAsset: "BNB", Symbol: "ABC_BNB", Oid: "8"},
		{inAsset: "ABC", Symbol: "ABC_BNB", Oid: "9"},
		{inAsset: "ABC", Symbol: "ABC_BTC", Oid: "10"},
	}
	e.Sort()
	require.Equal(t, ExpireTransfers{
		{inAsset: "BNB", Symbol: "ABC_BNB", Oid: "8"},
		{inAsset: "BNB", Symbol: "BNB_BTC", Oid: "7"},
		{inAsset: "ABC", Symbol: "ABC_BNB", Oid: "1"},
		{inAsset: "ABC", Symbol: "ABC_BNB", Oid: "9"},
		{inAsset: "ABC", Symbol: "ABC_BTC", Oid: "2"},
		{inAsset: "ABC", Symbol: "ABC_BTC", Oid: "10"},
		{inAsset: "ABC", Symbol: "ABC_XYZ", Oid: "5"},
		{inAsset: "BTC", Symbol: "BNB_BTC", Oid: "6"},
		{inAsset: "XYZ", Symbol: "XYZ_BNB", Oid: "4"},
		{inAsset: "XYZ", Symbol: "XYZ_BTC", Oid: "3"},
	}, e)
}
