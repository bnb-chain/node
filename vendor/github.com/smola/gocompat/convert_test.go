package compat

import (
	"fmt"
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"
)

const thisPkg = "github.com/smola/gocompat"

type ConvertObjectFixture struct {
	Name     string
	Input    string
	Expected *Object
}

var ConvertObjectFixtures = []ConvertObjectFixture{{
	Name:  "empty struct",
	Input: "StructA1",
	Expected: &Object{
		Symbol: Symbol{Package: thisPkg, Name: "StructA1"},
		Type:   TypeDeclaration,
		Definition: &Definition{
			Type: StructType,
		},
	}}, {
	Name:  "struct with fields",
	Input: "StructA4",
	Expected: &Object{
		Symbol: Symbol{Package: thisPkg, Name: "StructA4"},
		Type:   TypeDeclaration,
		Definition: &Definition{
			Type: StructType,
			Fields: []*Field{{
				Name: "Field1",
				Type: &Definition{
					Type:   BasicType,
					Symbol: &Symbol{Name: "string"},
				}}, {
				Name: "Field2",
				Type: &Definition{
					Type:   BasicType,
					Symbol: &Symbol{Name: "int"},
				}},
			}}}}, {
	Name:  "type alias with basic type",
	Input: "TypeAlias1",
	Expected: &Object{
		Symbol: Symbol{Package: thisPkg, Name: "TypeAlias1"},
		Type:   AliasDeclaration,
		Definition: &Definition{
			Type:   BasicType,
			Symbol: &Symbol{Name: "int"},
		},
	}}, {
	Name:  "type alias with struct",
	Input: "TypeAlias3",
	Expected: &Object{
		Symbol: Symbol{Package: thisPkg, Name: "TypeAlias3"},
		Type:   AliasDeclaration,
		Definition: &Definition{
			Symbol: &Symbol{Package: thisPkg, Name: "StructA3"},
		},
	}}, {
	Name:  "func",
	Input: "Func5",
	Expected: &Object{
		Symbol: Symbol{Package: thisPkg, Name: "Func5"},
		Type:   FuncDeclaration,
		Definition: &Definition{
			Type: FuncType,
			Signature: &Signature{
				Params: []*Definition{
					{Type: BasicType, Symbol: &Symbol{Name: "string"}},
					{Type: BasicType, Symbol: &Symbol{Name: "int"}},
				},
				Results: []*Definition{
					{Type: BasicType, Symbol: &Symbol{Name: "string"}},
				},
			},
		},
	}}, {
	Name:  "func variadic",
	Input: "Func8",
	Expected: &Object{
		Symbol: Symbol{Package: thisPkg, Name: "Func8"},
		Type:   FuncDeclaration,
		Definition: &Definition{
			Type: FuncType,
			Signature: &Signature{
				Params: []*Definition{{
					Type: SliceType,
					Elem: &Definition{
						Type:   BasicType,
						Symbol: &Symbol{Name: "string"}},
				}},
				Variadic: true,
			},
		},
	}}, {
	Name:  "struct with method",
	Input: "StructB1",
	Expected: &Object{
		Symbol: Symbol{Package: thisPkg, Name: "StructB1"},
		Type:   TypeDeclaration,
		Definition: &Definition{
			Type: StructType,
		},
		Methods: []*Func{
			{Name: "Func1", Signature: Signature{}},
		},
	}}, {
	Name:  "array of ints",
	Input: "Array1",
	Expected: &Object{
		Symbol: Symbol{Package: thisPkg, Name: "Array1"},
		Type:   VarDeclaration,
		Definition: &Definition{
			Type: ArrayType,
			Len:  20,
			Elem: &Definition{
				Type:   BasicType,
				Symbol: &Symbol{Name: "int"},
			},
		},
	}}, {
	Name:  "read chan",
	Input: "ChanRead1",
	Expected: &Object{
		Symbol: Symbol{Package: thisPkg, Name: "ChanRead1"},
		Type:   VarDeclaration,
		Definition: &Definition{
			Type:    ChanType,
			ChanDir: types.RecvOnly,
			Elem: &Definition{
				Type:   BasicType,
				Symbol: &Symbol{Name: "int"},
			},
		},
	}}, {
	Name:  "map",
	Input: "Map1",
	Expected: &Object{
		Symbol: Symbol{Package: thisPkg, Name: "Map1"},
		Type:   VarDeclaration,
		Definition: &Definition{
			Type: MapType,
			Key: &Definition{
				Type:   BasicType,
				Symbol: &Symbol{Name: "int"},
			},
			Elem: &Definition{
				Type:   BasicType,
				Symbol: &Symbol{Name: "string"},
			},
		},
	}}, {
	Name:  "pointer",
	Input: "Pointer1",
	Expected: &Object{
		Symbol: Symbol{Package: thisPkg, Name: "Pointer1"},
		Type:   VarDeclaration,
		Definition: &Definition{
			Type: PointerType,
			Elem: &Definition{
				Type:   BasicType,
				Symbol: &Symbol{Name: "int"},
			},
		},
	}}, {
	Name:  "interface",
	Input: "Interface3",
	Expected: &Object{
		Symbol: Symbol{Package: thisPkg, Name: "Interface3"},
		Type:   TypeDeclaration,
		Definition: &Definition{
			Type: InterfaceType,
			Functions: []*Func{
				{Name: "F", Signature: Signature{
					Params: []*Definition{
						{Type: BasicType, Symbol: &Symbol{Name: "string"}},
					},
				}},
			},
		},
	}}, {
	Name:  "interface with private method",
	Input: "Interface4",
	Expected: &Object{
		Symbol: Symbol{Package: thisPkg, Name: "Interface4"},
		Type:   TypeDeclaration,
		Definition: &Definition{
			Type: InterfaceType,
			Functions: []*Func{
				{Name: "f"},
			},
		},
	}}}

func TestConvertObjects(t *testing.T) {
	for _, fixture := range ConvertObjectFixtures {
		name := fmt.Sprintf("%s_%s", fixture.Name, fixture.Input)
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			a, ok := FixtureObjects[fixture.Input]
			require.True(ok)
			actual := ConvertObject(a)
			require.Equal(fixture.Expected, actual)
		})
	}
}
