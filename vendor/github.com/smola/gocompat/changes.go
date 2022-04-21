package compat

import (
	"fmt"
	"strings"
)

//go:generate stringer -type=ChangeType

type ChangeType int

const (
	_ ChangeType = iota
	PackageDeleted
	SymbolAdded
	SymbolDeleted
	TypeChanged
	FieldAdded
	FieldDeleted
	FieldChangedType
	SignatureChanged
	MethodAdded
	MethodDeleted
	MethodSignatureChanged
	InterfaceChanged
)

type Change struct {
	Type   ChangeType
	Symbol string
}

func (c Change) String() string {
	return fmt.Sprintf("%s %s", c.Symbol, c.Type.String())
}

func init() {
	for i := 0; i < len(_ChangeType_index)-1; i++ {
		s, e := _ChangeType_index[i], _ChangeType_index[i+1]
		name := _ChangeType_name[s:e]
		name = strings.ToLower(name)
		lookupChangeType[name] = ChangeType(i + 1)
	}
}

var lookupChangeType = make(map[string]ChangeType)

// ChangeTypeFromString converts a string representation of the ChangeType to
// its numeric value.
func ChangeTypeFromString(s string) (ChangeType, error) {
	c, ok := lookupChangeType[strings.ToLower(s)]
	if !ok {
		return 0, fmt.Errorf("invalid change type: %s", s)
	}

	return c, nil
}
