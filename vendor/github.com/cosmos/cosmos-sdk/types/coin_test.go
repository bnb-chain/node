package types

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsPositiveCoin(t *testing.T) {
	cases := []struct {
		inputOne Coin
		expected bool
	}{
		{NewCoin("A", 1), true},
		{NewCoin("A", 0), false},
		{NewCoin("a", -1), false},
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.IsPositive()
		require.Equal(t, tc.expected, res, "%s positivity is incorrect, tc #%d", tc.inputOne.String(), tcIndex)
	}
}

func TestIsNotNegativeCoin(t *testing.T) {
	cases := []struct {
		inputOne Coin
		expected bool
	}{
		{NewCoin("A", 1), true},
		{NewCoin("A", 0), true},
		{NewCoin("a", -1), false},
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.IsNotNegative()
		require.Equal(t, tc.expected, res, "%s not-negativity is incorrect, tc #%d", tc.inputOne.String(), tcIndex)
	}
}

func TestSameDenomAsCoin(t *testing.T) {
	cases := []struct {
		inputOne Coin
		inputTwo Coin
		expected bool
	}{
		{NewCoin("A", 1), NewCoin("A", 1), true},
		{NewCoin("A", 1), NewCoin("a", 1), false},
		{NewCoin("a", 1), NewCoin("b", 1), false},
		{NewCoin("steak", 1), NewCoin("steak", 10), true},
		{NewCoin("steak", -11), NewCoin("steak", 10), true},
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.SameDenomAs(tc.inputTwo)
		require.Equal(t, tc.expected, res, "coin denominations didn't match, tc #%d", tcIndex)
	}
}

func TestIsGTECoin(t *testing.T) {
	cases := []struct {
		inputOne Coin
		inputTwo Coin
		expected bool
	}{
		{NewCoin("A", 1), NewCoin("A", 1), true},
		{NewCoin("A", 2), NewCoin("A", 1), true},
		{NewCoin("A", -1), NewCoin("A", 5), false},
		{NewCoin("a", 1), NewCoin("b", 1), false},
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.IsGTE(tc.inputTwo)
		require.Equal(t, tc.expected, res, "coin GTE relation is incorrect, tc #%d", tcIndex)
	}
}

func TestIsLTCoin(t *testing.T) {
	cases := []struct {
		inputOne Coin
		inputTwo Coin
		expected bool
	}{
		{NewCoin("A", 1), NewCoin("A", 1), false},
		{NewCoin("A", 2), NewCoin("A", 1), false},
		{NewCoin("A", -1), NewCoin("A", 5), true},
		{NewCoin("a", 0), NewCoin("b", 1), true},
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.IsLT(tc.inputTwo)
		require.Equal(t, tc.expected, res, "coin LT relation is incorrect, tc #%d", tcIndex)
	}
}

func TestIsEqualCoin(t *testing.T) {
	cases := []struct {
		inputOne Coin
		inputTwo Coin
		expected bool
	}{
		{NewCoin("A", 1), NewCoin("A", 1), true},
		{NewCoin("A", 1), NewCoin("a", 1), false},
		{NewCoin("a", 1), NewCoin("b", 1), false},
		{NewCoin("steak", 1), NewCoin("steak", 10), false},
		{NewCoin("steak", -11), NewCoin("steak", 10), false},
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.IsEqual(tc.inputTwo)
		require.Equal(t, tc.expected, res, "coin equality relation is incorrect, tc #%d", tcIndex)
	}
}

func TestPlusCoin(t *testing.T) {
	cases := []struct {
		inputOne Coin
		inputTwo Coin
		expected Coin
	}{
		{NewCoin("A", 1), NewCoin("A", 1), NewCoin("A", 2)},
		{NewCoin("A", 1), NewCoin("B", 1), NewCoin("A", 1)},
		{NewCoin("asdf", -4), NewCoin("asdf", 5), NewCoin("asdf", 1)},
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.Plus(tc.inputTwo)
		require.Equal(t, tc.expected, res, "sum of coins is incorrect, tc #%d", tcIndex)
	}

	tc := struct {
		inputOne Coin
		inputTwo Coin
		expected int64
	}{NewCoin("asdf", -1), NewCoin("asdf", 1), 0}
	res := tc.inputOne.Plus(tc.inputTwo)
	require.Equal(t, tc.expected, res.Amount)
}

func TestMinusCoin(t *testing.T) {
	cases := []struct {
		inputOne Coin
		inputTwo Coin
		expected Coin
	}{

		{NewCoin("A", 1), NewCoin("B", 1), NewCoin("A", 1)},
		{NewCoin("asdf", -4), NewCoin("asdf", 5), NewCoin("asdf", -9)},
		{NewCoin("asdf", 10), NewCoin("asdf", 1), NewCoin("asdf", 9)},
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.Minus(tc.inputTwo)
		require.Equal(t, tc.expected, res, "difference of coins is incorrect, tc #%d", tcIndex)
	}

	tc := struct {
		inputOne Coin
		inputTwo Coin
		expected int64
	}{NewCoin("A", 1), NewCoin("A", 1), 0}
	res := tc.inputOne.Minus(tc.inputTwo)
	require.Equal(t, tc.expected, res.Amount)

}

func TestIsZeroCoins(t *testing.T) {
	cases := []struct {
		inputOne Coins
		expected bool
	}{
		{Coins{}, true},
		{Coins{NewCoin("A", 0)}, true},
		{Coins{NewCoin("A", 0), NewCoin("B", 0)}, true},
		{Coins{NewCoin("A", 1)}, false},
		{Coins{NewCoin("A", 0), NewCoin("B", 1)}, false},
	}

	for _, tc := range cases {
		res := tc.inputOne.IsZero()
		require.Equal(t, tc.expected, res)
	}
}

func TestEqualCoins(t *testing.T) {
	cases := []struct {
		inputOne Coins
		inputTwo Coins
		expected bool
	}{
		{Coins{}, Coins{}, true},
		{Coins{NewCoin("A", 0)}, Coins{NewCoin("A", 0)}, true},
		{Coins{NewCoin("A", 0), NewCoin("B", 1)}, Coins{NewCoin("A", 0), NewCoin("B", 1)}, true},
		{Coins{NewCoin("A", 0)}, Coins{NewCoin("B", 0)}, false},
		{Coins{NewCoin("A", 0)}, Coins{NewCoin("A", 1)}, false},
		{Coins{NewCoin("A", 0)}, Coins{NewCoin("A", 0), NewCoin("B", 1)}, false},
		// TODO: is it expected behaviour? shouldn't we sort the coins before comparing them?
		{Coins{NewCoin("A", 0), NewCoin("B", 1)}, Coins{NewCoin("B", 1), NewCoin("A", 0)}, false},
	}

	for tcnum, tc := range cases {
		res := tc.inputOne.IsEqual(tc.inputTwo)
		require.Equal(t, tc.expected, res, "Equality is differed from expected. tc #%d, expected %b, actual %b.", tcnum, tc.expected, res)
	}
}

func TestCoins(t *testing.T) {

	//Define the coins to be used in tests
	good := Coins{
		{"GAS", 1},
		{"MINERAL", 1},
		{"TREE", 1},
	}
	neg := good.Negative()
	sum := good.Plus(neg)
	empty := Coins{
		{"GOLD", 0},
	}
	null := Coins{}
	badSort1 := Coins{
		{"TREE", 0},
		{"GAS", 0},
		{"MINERAL", 0},
	}
	// both are after the first one, but the second and third are in the wrong order
	badSort2 := Coins{
		{"GAS", 1},
		{"TREE", 1},
		{"MINERAL", 1},
	}
	badAmt := Coins{
		{"GAS", 1},
		{"TREE", 1},
		{"MINERAL", 1},
	}
	dup := Coins{
		{"GAS", 1},
		{"GAS", 1},
		{"MINERAL", 1},
	}

	assert.True(t, good.IsValid(), "Coins are valid")
	assert.True(t, good.IsPositive(), "Expected coins to be positive: %v", good)
	assert.False(t, null.IsPositive(), "Expected coins to not be positive: %v", null)
	assert.True(t, good.IsGTE(empty), "Expected %v to be >= %v", good, empty)
	assert.False(t, good.IsLT(empty), "Expected %v to be < %v", good, empty)
	assert.True(t, empty.IsLT(good), "Expected %v to be < %v", empty, good)
	assert.False(t, neg.IsPositive(), "Expected neg coins to not be positive: %v", neg)
	assert.Zero(t, len(sum), "Expected 0 coins")
	assert.False(t, badSort1.IsValid(), "Coins are not sorted")
	assert.False(t, badSort2.IsValid(), "Coins are not sorted")
	assert.False(t, badAmt.IsValid(), "Coins cannot include 0 amounts")
	assert.False(t, dup.IsValid(), "Duplicate coin")

}

func TestPlusCoins(t *testing.T) {
	one := int64(1)
	zero := int64(0)
	negone := int64(-1)
	two := int64(2)

	cases := []struct {
		inputOne Coins
		inputTwo Coins
		expected Coins
	}{
		{Coins{{"A", one}, {"B", one}}, Coins{{"A", one}, {"B", one}}, Coins{{"A", two}, {"B", two}}},
		{Coins{{"A", zero}, {"B", one}}, Coins{{"A", zero}, {"B", zero}}, Coins{{"B", one}}},
		{Coins{{"A", zero}, {"B", zero}}, Coins{{"A", zero}, {"B", zero}}, Coins(nil)},
		{Coins{{"A", one}, {"B", zero}}, Coins{{"A", negone}, {"B", zero}}, Coins(nil)},
		{Coins{{"A", negone}, {"B", zero}}, Coins{{"A", zero}, {"B", zero}}, Coins{{"A", negone}}},
	}

	for tcIndex, tc := range cases {
		res := tc.inputOne.Plus(tc.inputTwo)
		assert.True(t, res.IsValid())
		require.Equal(t, tc.expected, res, "sum of coins is incorrect, tc #%d", tcIndex)
	}
}

//Test the parsing of Coin and Coins
func TestParse(t *testing.T) {
	one := int64(1)

	cases := []struct {
		input    string
		valid    bool  // if false, we expect an error on parse
		expected Coins // if valid is true, make sure this is returned
	}{
		{"", true, nil},
		{"1:foo", true, Coins{{"foo", one}}},
		{"10:bar", true, Coins{{"bar", 10}}},
		{"10:bar.B", true, Coins{{"bar.B", 10}}},
		{"10:bar-1B", true, Coins{{"bar-1B", 10}}},
		{"10:bar-1BCDEF", true, Coins{{"bar-1BCDEF", 10}}},
		{"10:bar.B-1BCDEF", true, Coins{{"bar.B-1BCDEF", 10}}},
		{"99:bar,1:foo", true, Coins{{"bar", 99}, {"foo", one}}},
		{"98:bar , 1:foo  ", true, Coins{{"bar", 98}, {"foo", one}}},
		{"  55:bling\n", true, Coins{{"bling", 55}}},
		{"2:foo, 97:bar", true, Coins{{"bar", 97}, {"foo", 2}}},
		{"10:ba-1BC", true, Coins{{"ba-1BC", 10}}},
		{"5:mycoin,", false, nil},                                 // no empty coins in a list
		{"2:3foo, 97:bar", true, Coins{{"3foo", 2}, {"bar", 97}}}, // 3foo is invalid coin name
		{"11me:coin, 12you:coin", false, nil},                     // no spaces in coin names
		{"1.2:btc", false, nil},                                   // amount must be integer
		{"5:foo-bar", false, nil},                                 // once more, only letters in coin name
		{"5:foo-12BCDEF", false, nil},                             // incorrect tx suffix lens
	}

	for tcIndex, tc := range cases {
		res, err := ParseCoins(tc.input)
		if !tc.valid {
			require.NotNil(t, err, "%s: %#v. tc #%d", tc.input, res, tcIndex)
		} else if assert.Nil(t, err, "%s: %+v", tc.input, err) {
			require.Equal(t, tc.expected, res, "coin parsing was incorrect, tc #%d", tcIndex)
		}
	}

}

func TestSortCoins(t *testing.T) {

	good := Coins{
		NewCoin("GAS", 1),
		NewCoin("MINERAL", 1),
		NewCoin("TREE", 1),
	}
	empty := Coins{
		NewCoin("GOLD", 0),
	}
	badSort1 := Coins{
		NewCoin("TREE", 1),
		NewCoin("GAS", 1),
		NewCoin("MINERAL", 1),
	}
	badSort2 := Coins{ // both are after the first one, but the second and third are in the wrong order
		NewCoin("GAS", 1),
		NewCoin("TREE", 1),
		NewCoin("MINERAL", 1),
	}
	badAmt := Coins{
		NewCoin("GAS", 1),
		NewCoin("TREE", 0),
		NewCoin("MINERAL", 1),
	}
	dup := Coins{
		NewCoin("GAS", 1),
		NewCoin("GAS", 1),
		NewCoin("MINERAL", 1),
	}

	cases := []struct {
		coins         Coins
		before, after bool // valid before/after sort
	}{
		{good, true, true},
		{empty, false, false},
		{badSort1, false, true},
		{badSort2, false, true},
		{badAmt, false, false},
		{dup, false, false},
	}

	for tcIndex, tc := range cases {
		require.Equal(t, tc.before, tc.coins.IsValid(), "coin validity is incorrect before sorting, tc #%d", tcIndex)
		tc.coins.Sort()
		require.Equal(t, tc.after, tc.coins.IsValid(), "coin validity is incorrect after sorting, tc #%d", tcIndex)
	}
}

func TestAmountOf(t *testing.T) {

	case0 := Coins{}
	case1 := Coins{
		NewCoin("", 0),
	}
	case2 := Coins{
		NewCoin(" ", 0),
	}
	case3 := Coins{
		NewCoin("GOLD", 0),
	}
	case4 := Coins{
		NewCoin("GAS", 1),
		NewCoin("MINERAL", 1),
		NewCoin("TREE", 1),
	}
	case5 := Coins{
		NewCoin("MINERAL", 1),
		NewCoin("TREE", 1),
	}
	case6 := Coins{
		NewCoin("", 6),
	}
	case7 := Coins{
		NewCoin(" ", 7),
	}
	case8 := Coins{
		NewCoin("GAS", 8),
	}

	cases := []struct {
		coins           Coins
		amountOf        int64
		amountOfSpace   int64
		amountOfGAS     int64
		amountOfMINERAL int64
		amountOfTREE    int64
	}{
		{case0, 0, 0, 0, 0, 0},
		{case1, 0, 0, 0, 0, 0},
		{case2, 0, 0, 0, 0, 0},
		{case3, 0, 0, 0, 0, 0},
		{case4, 0, 0, 1, 1, 1},
		{case5, 0, 0, 0, 1, 1},
		{case6, 6, 0, 0, 0, 0},
		{case7, 0, 7, 0, 0, 0},
		{case8, 0, 0, 8, 0, 0},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.amountOf, tc.coins.AmountOf(""))
		assert.Equal(t, tc.amountOfSpace, tc.coins.AmountOf(" "))
		assert.Equal(t, tc.amountOfGAS, tc.coins.AmountOf("GAS"))
		assert.Equal(t, tc.amountOfMINERAL, tc.coins.AmountOf("MINERAL"))
		assert.Equal(t, tc.amountOfTREE, tc.coins.AmountOf("TREE"))
	}
}

func BenchmarkCoinsAdditionIntersect(b *testing.B) {
	benchmarkingFunc := func(numCoinsA int, numCoinsB int) func(b *testing.B) {
		return func(b *testing.B) {
			coinsA := Coins(make([]Coin, numCoinsA))
			coinsB := Coins(make([]Coin, numCoinsB))
			for i := 0; i < numCoinsA; i++ {
				coinsA[i] = NewCoin("COINZ_"+strconv.Itoa(i), int64(i))
			}
			for i := 0; i < numCoinsB; i++ {
				coinsB[i] = NewCoin("COINZ_"+strconv.Itoa(i), int64(i))
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				coinsA.Plus(coinsB)
			}
		}
	}

	benchmarkSizes := [][]int{{1, 1}, {5, 5}, {5, 20}, {1, 1000}, {2, 1000}}
	for i := 0; i < len(benchmarkSizes); i++ {
		sizeA := benchmarkSizes[i][0]
		sizeB := benchmarkSizes[i][1]
		b.Run(fmt.Sprintf("sizes: A_%d, B_%d", sizeA, sizeB), benchmarkingFunc(sizeA, sizeB))
	}
}

func BenchmarkCoinsAdditionNoIntersect(b *testing.B) {
	benchmarkingFunc := func(numCoinsA int, numCoinsB int) func(b *testing.B) {
		return func(b *testing.B) {
			coinsA := Coins(make([]Coin, numCoinsA))
			coinsB := Coins(make([]Coin, numCoinsB))
			for i := 0; i < numCoinsA; i++ {
				coinsA[i] = NewCoin("COINZ_"+strconv.Itoa(numCoinsB+i), int64(i))
			}
			for i := 0; i < numCoinsB; i++ {
				coinsB[i] = NewCoin("COINZ_"+strconv.Itoa(i), int64(i))
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				coinsA.Plus(coinsB)
			}
		}
	}

	benchmarkSizes := [][]int{{1, 1}, {5, 5}, {5, 20}, {1, 1000}, {2, 1000}, {1000, 2}}
	for i := 0; i < len(benchmarkSizes); i++ {
		sizeA := benchmarkSizes[i][0]
		sizeB := benchmarkSizes[i][1]
		b.Run(fmt.Sprintf("sizes: A_%d, B_%d", sizeA, sizeB), benchmarkingFunc(sizeA, sizeB))
	}
}
