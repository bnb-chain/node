package mint_test

import (
	"fmt"
	"testing"

	"github.com/otiai10/curr"

	"github.com/otiai10/mint"
)

func TestMint_ToBeX(t *testing.T) {
	mint.Expect(t, 1).ToBe(1)
	type Foo struct{}
	var foo *Foo
	mint.Expect(t, foo).ToBe(nil)
	foo = &Foo{}
	mint.Expect(t, foo).Not().ToBe(nil)
	foo = nil
	mint.Expect(t, foo).Not().ToBe(100)
}
func TestMint_ToBe_Fail(t *testing.T) {
	r := mint.Expect(t, 2).Dry().ToBe(1)
	// assert mint by using mint
	mint.Expect(t, r.OK()).ToBe(false)
}

func TestMint_Match(t *testing.T) {
	mint.Expect(t, 10).Match("10")
	mint.Expect(t, "3.05.00dev").Match("[0-9].[0-9]{2}(.[0-9a-z]+)?")
}

func TestTestee_In(t *testing.T) {
	r := mint.Expect(t, 100).Dry().In(10, 100, 1000)
	mint.Expect(t, r.OK()).ToBe(true)

	r = mint.Expect(t, 999).Dry().In(10, 100, 1000)
	mint.Expect(t, r.OK()).ToBe(false)
}

type MyStruct struct{}

func TestMint_TypeOf(t *testing.T) {
	mint.Expect(t, "foo").TypeOf("string")

	bar := MyStruct{}
	mint.Expect(t, bar).TypeOf("mint_test.MyStruct")
}
func TestMint_TypeOf_Fail(t *testing.T) {
	r := mint.Expect(t, "foo").Dry().TypeOf("int")
	// assert mint by using mint
	mint.Expect(t, r.OK()).ToBe(false)
	mint.Expect(t, r.NG()).ToBe(true)

	bar := MyStruct{}
	line := curr.Line() + 1
	r = mint.Expect(t, bar).Dry().TypeOf("foo.Bar")
	// assert mint by using mint
	mint.Expect(t, r.OK()).ToBe(false)
	mint.Expect(t, r.NG()).ToBe(true)
	mint.Expect(t, r.Message()).ToBe(fmt.Sprintf("all_test.go at line %d\n\tExpected type\t`foo.Bar`\n\tBut actual\t`mint_test.MyStruct`", line))
}

func TestMint_Not(t *testing.T) {
	mint.Expect(t, 100).Not().ToBe(200)
	mint.Expect(t, "foo").Not().TypeOf("int")
	mint.Expect(t, true).Not().ToBe(nil)
}
func TestMint_Not_Fail(t *testing.T) {
	r := mint.Expect(t, "foo").Dry().Not().TypeOf("string")
	// assert mint by using mint
	mint.Expect(t, r.OK()).Not().ToBe(true)
	mint.Expect(t, r.NG()).ToBe(true)
}

func TestMint_Deeply(t *testing.T) {
	map0 := &map[int]string{
		3:  "three",
		5:  "five",
		10: "ten",
	}
	map1 := &map[int]string{
		3:  "three",
		5:  "five",
		10: "ten",
	}
	// It SHALLOWLY different.
	mint.Expect(t, map0).Not().ToBe(map1)
	// But it DEEPLY equal.
	mint.Expect(t, map0).Deeply().ToBe(map1)
}

func TestMint_Deeply_slice(t *testing.T) {
	s1 := []int{1, 2, 3}
	s2 := []int{4, 5, 6}
	s3 := []int{1, 2, 3}
	mint.Expect(t, s1).Not().ToBe(s2)
	mint.Expect(t, s1).ToBe(s3)
}
func TestMint_Deeply_map(t *testing.T) {
	m1 := map[int]int{1: 1, 2: 4, 3: 9}
	m2 := map[int]int{1: 1, 2: 4, 3: 6}
	m3 := map[int]int{1: 1, 2: 4, 3: 9}
	mint.Expect(t, m1).Not().ToBe(m2)
	mint.Expect(t, m1).ToBe(m3)
}

// Blend is a shorhand to get testee
func TestMint_Blend(t *testing.T) {
	m := mint.Blend(t)
	// assert mint by using mint
	mint.Expect(t, m).TypeOf("*mint.Mint")
	mint.Expect(t, m.Expect("foo")).TypeOf("*mint.Testee")
}

// Because
func TestBecause(t *testing.T) {
	mint.Because(t, "`Because` should print context.", func(t *testing.T) {
		mint.Expect(t, true).ToBe(true)
		res := mint.Expect(t, false).Dry().ToBe(true)
		mint.Expect(t, res.OK()).ToBe(false)
	})
}

// When
func TestWhen(t *testing.T) {
	mint.When(t, "`When` should print context.", func(t *testing.T) {
		mint.Expect(t, true).ToBe(true)
		res := mint.Expect(t, false).Dry().ToBe(true)
		mint.Expect(t, res.OK()).ToBe(false)
	})
}
