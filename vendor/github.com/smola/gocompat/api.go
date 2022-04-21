package compat

import (
	"go/types"
)

//go:generate easyjson -all api.go

// API
type API struct {
	// Packages included directly in the API.
	Packages []*Package
	// Reachable is the set of all objects that are reachable from the API.
	Reachable []*Object
}

// NewAPI creates an empty API.
func NewAPI() *API {
	return &API{}
}

func (a *API) LookupSymbol(sym *Symbol) *Object {
	for _, obj := range a.Reachable {
		if &obj.Symbol == sym {
			return obj
		}
	}

	return nil
}

type Package struct {
	Path    string             `json:"path"`
	Objects map[string]*Object `json:"objects,omitempty"`
}

func NewPackage(path string) *Package {
	return &Package{
		Path:    path,
		Objects: make(map[string]*Object),
	}
}

type DeclarationType string

const (
	TypeDeclaration  DeclarationType = "type"
	AliasDeclaration                 = "alias"
	VarDeclaration                   = "var"
	ConstDeclaration                 = "const"
	FuncDeclaration                  = "func"
)

type Type string

const (
	StructType    Type = "struct"
	BasicType          = "basic"
	SliceType          = "slice"
	ArrayType          = "array"
	FuncType           = "func"
	ChanType           = "chan"
	MapType            = "map"
	PointerType        = "pointer"
	InterfaceType      = "interface"
)

type Symbol struct {
	Package string `json:"package"`
	Name    string `json:"name"`
}

type Object struct {
	Symbol     Symbol          `json:"symbol"`
	Type       DeclarationType `json:"type"`
	Definition *Definition     `json:"definition,omitempty"`
	Methods    []*Func         `json:"methods,omitempty"`
}

type Definition struct {
	Type      Type          `json:"type"`
	Symbol    *Symbol       `json:"symbol,omitempty"`
	Elem      *Definition   `json:"elem,omitempty"`
	Key       *Definition   `json:"key,omitempty"`
	Len       int64         `json:"len,omitempty"`
	ChanDir   types.ChanDir `json:"chandir,omitempty"`
	Fields    []*Field      `json:"fields,omitempty"`
	Functions []*Func       `json:"functions,omitempty"`
	Signature *Signature    `json:"signature,omitempty"`
}

type Field struct {
	Name string
	Type *Definition
}

type Func struct {
	Signature
	Name string
}

type Signature struct {
	Params   []*Definition
	Results  []*Definition
	Variadic bool
}
