package compat

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type CompareFixture struct {
	Name     string
	A, B     string
	Expected []Change
}

var CompareFixtures = []CompareFixture{{
	Name: "equal empty struct",
	A:    "StructA1",
	B:    "StructA1",
}, {
	Name: "equal struct",
	A:    "StructA2",
	B:    "StructA2",
}, {
	Name: "added field to empty struct",
	A:    "StructA1",
	B:    "StructA2",
	Expected: []Change{{
		Type:   FieldAdded,
		Symbol: `"github.com/smola/gocompat".StructA1.Field1`}},
}, {
	Name: "added field to struct",
	A:    "StructA2",
	B:    "StructA3",
	Expected: []Change{{
		Type:   FieldAdded,
		Symbol: `"github.com/smola/gocompat".StructA2.Field2`}},
}, {
	Name: "added two fields to struct",
	A:    "StructA1",
	B:    "StructA3",
	Expected: []Change{{
		Type:   FieldAdded,
		Symbol: `"github.com/smola/gocompat".StructA1.Field1`,
	}, {
		Type:   FieldAdded,
		Symbol: `"github.com/smola/gocompat".StructA1.Field2`,
	}},
}, {
	Name: "deleted field to struct",
	A:    "StructA3",
	B:    "StructA2",
	Expected: []Change{{
		Type:   FieldDeleted,
		Symbol: `"github.com/smola/gocompat".StructA3.Field2`,
	}},
}, {
	Name: "deleted field to struct (empty)",
	A:    "StructA2",
	B:    "StructA1",
	Expected: []Change{{
		Type:   FieldDeleted,
		Symbol: `"github.com/smola/gocompat".StructA2.Field1`,
	}},
}, {
	Name: "changed type of struct field",
	A:    "StructA3",
	B:    "StructA4",
	Expected: []Change{{
		Type:   FieldChangedType,
		Symbol: `"github.com/smola/gocompat".StructA3.Field2`,
	}},
}, {
	Name: "changed type of struct field and added field",
	A:    "StructA3",
	B:    "StructA5",
	Expected: []Change{{
		Type:   FieldChangedType,
		Symbol: `"github.com/smola/gocompat".StructA3.Field2`,
	}, {
		Type:   FieldAdded,
		Symbol: `"github.com/smola/gocompat".StructA3.Field3`,
	}},
}, {
	Name: "type alias equal",
	A:    "TypeAlias1",
	B:    "TypeAlias1",
}, {
	Name: "type alias changed",
	A:    "TypeAlias1",
	B:    "TypeAlias2",
	Expected: []Change{{
		Type:   TypeChanged,
		Symbol: `"github.com/smola/gocompat".TypeAlias1`,
	}},
}, {
	Name: "type alias changed to struct",
	A:    "TypeAlias2",
	B:    "TypeAlias3",
	Expected: []Change{{
		Type:   TypeChanged,
		Symbol: `"github.com/smola/gocompat".TypeAlias2`,
	}},
}, {
	Name: "type alias changed between structs",
	A:    "TypeAlias3",
	B:    "TypeAlias4",
	Expected: []Change{{
		Type:   TypeChanged,
		Symbol: `"github.com/smola/gocompat".TypeAlias3`,
	}},
}, {
	Name: "func equal",
	A:    "Func5",
	B:    "Func5",
}, {
	Name: "func add arg",
	A:    "Func1",
	B:    "Func2",
	Expected: []Change{{
		Type:   SignatureChanged,
		Symbol: `"github.com/smola/gocompat".Func1`,
	}},
}, {
	Name: "func change arg type",
	A:    "Func3",
	B:    "Func4",
	Expected: []Change{{
		Type:   SignatureChanged,
		Symbol: `"github.com/smola/gocompat".Func3`,
	}},
}, {
	Name: "func change arg variadic",
	A:    "Func1",
	B:    "Func8",
	Expected: []Change{{
		Type:   SignatureChanged,
		Symbol: `"github.com/smola/gocompat".Func1`,
	}},
}, {
	Name: "func change return type",
	A:    "Func5",
	B:    "Func6",
	Expected: []Change{{
		Type:   SignatureChanged,
		Symbol: `"github.com/smola/gocompat".Func5`,
	}},
}, {
	Name: "func change return type multiple",
	A:    "Func5",
	B:    "Func7",
	Expected: []Change{{
		Type:   SignatureChanged,
		Symbol: `"github.com/smola/gocompat".Func5`,
	}},
}, {
	Name: "type with method equal",
	A:    "StructB1",
	B:    "StructB1",
}, {
	Name: "type with added method",
	A:    "StructA1",
	B:    "StructB1",
	Expected: []Change{{
		Type:   MethodAdded,
		Symbol: `"github.com/smola/gocompat".StructA1.Func1`,
	}},
}, {
	Name: "type with deleted method",
	A:    "StructB1",
	B:    "StructA1",
	Expected: []Change{{
		Type:   MethodDeleted,
		Symbol: `"github.com/smola/gocompat".StructB1.Func1`,
	}},
}, {
	Name: "type with changed method signature",
	A:    "StructB1",
	B:    "StructB2",
	Expected: []Change{{
		Type:   MethodSignatureChanged,
		Symbol: `"github.com/smola/gocompat".StructB1.Func1`,
	}},
}, {
	Name: "interface empty equal",
	A:    "Interface1",
	B:    "Interface1",
}, {
	Name: "interface equal",
	A:    "Interface2",
	B:    "Interface2",
}, {
	Name: "interface method added",
	A:    "Interface1",
	B:    "Interface2",
	Expected: []Change{Change{
		Type:   InterfaceChanged,
		Symbol: `"github.com/smola/gocompat".Interface1`,
	}},
}, {
	Name: "interface method deleted",
	A:    "Interface2",
	B:    "Interface1",
	Expected: []Change{Change{
		Type:   InterfaceChanged,
		Symbol: `"github.com/smola/gocompat".Interface2`,
	}},
}, {
	Name: "interface method changed",
	A:    "Interface2",
	B:    "Interface3",
	Expected: []Change{Change{
		Type:   InterfaceChanged,
		Symbol: `"github.com/smola/gocompat".Interface2`,
	}},
}, {
	Name: "interface private method added",
	A:    "Interface1",
	B:    "Interface4",
	Expected: []Change{Change{
		Type:   InterfaceChanged,
		Symbol: `"github.com/smola/gocompat".Interface1`,
	}},
}}

func TestCompareObjects(t *testing.T) {
	for _, fixture := range CompareFixtures {
		name := fmt.Sprintf("%s_%s_%s", fixture.Name, fixture.A, fixture.B)
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			a, ok := FixtureObjects[fixture.A]
			require.True(ok)
			aobj := ConvertObject(a)
			b, ok := FixtureObjects[fixture.B]
			require.True(ok)
			bobj := ConvertObject(b)
			bobj.Symbol = aobj.Symbol
			actual := CompareObjects(aobj, bobj)
			require.Equal(fixture.Expected, actual)
		})
	}
}
