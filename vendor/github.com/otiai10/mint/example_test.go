package mint_test

import (
	"os"
	"testing"

	"github.com/otiai10/mint"
)

func ExampleExpect(t *testing.T) {
	mint.Expect(t, 100).ToBe(100)
	mint.Expect(t, 100).TypeOf("int")
}

func ExampleTestee_ToBe(t *testing.T) {
	mint.Expect(t, 100).ToBe(100)
}

func ExampleTestee_Match(t *testing.T) {
	mint.Expect(t, "3.05.00dev").Match("[0-9].[0-9]{2}(.[0-9a-z]+)?")
}

func ExampleTestee_TypeOf(t *testing.T) {
	mint.Expect(t, 100).TypeOf("int")
}

func ExampleTestee_In(t *testing.T) {
	mint.Expect(t, 100).In(10, 100, 1000)
}

func ExampleTestee_Not(t *testing.T) {
	mint.Expect(t, 100).Not().ToBe(200)
	mint.Expect(t, 100).Not().TypeOf("string")
}

func ExampleTestee_Deeply(t *testing.T) {
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
	mint.Expect(t, map0).Not().ToBe(map1)
	mint.Expect(t, map0).Deeply().ToBe(map1)
}

func ExampleTestee_Dry(t *testing.T) {
	result := mint.Expect(t, 100).Dry().ToBe(100)
	if !result.OK() {
		t.Fail()
	}
}

func ExampleBlend(t *testing.T) {
	// get blended mint
	m := mint.Blend(t)
	m.Expect(100).ToBe(100)
	m.Expect(100).Not().ToBe(200)
}

func ExampleExit(t *testing.T) {
	unsuccessful := func() {
		os.Exit(1)
	}
	mint.Expect(t, unsuccessful).Not().Exit(0)
}
