package compat

import (
	"fmt"
	"go/types"
	"reflect"
)

func Compare(a, b *API) []Change {
	var changes []Change
	checked := make(map[*Object]bool)

	//TODO: check added symbols to package
	for _, apkg := range a.Packages {
		found := false
		for _, bpkg := range b.Packages {
			if apkg.Path == bpkg.Path {
				changes = append(changes, comparePackages(checked, apkg, bpkg)...)
				found = true
				break
			}
		}

		if !found {
			changes = append(changes, Change{
				Type:   PackageDeleted,
				Symbol: fmt.Sprintf(`"%s"`, apkg.Path),
			})
		}
	}

	for _, aobj := range a.Reachable {
		bobj := b.LookupSymbol(&aobj.Symbol)
		if bobj == nil {
			//TODO: not reachable anymore!
			continue
		}

		changes = append(changes, CompareObjects(aobj, bobj)...)
	}

	return changes
}

func comparePackages(checked map[*Object]bool, a, b *Package) []Change {
	var result []Change
	for name, aobj := range a.Objects {
		checked[aobj] = true
		bobj, ok := b.Objects[name]
		if ok {
			result = append(result, CompareObjects(aobj, bobj)...)
			continue
		}

		result = append(result, Change{
			Type:   SymbolDeleted,
			Symbol: symbolName(aobj),
		})
	}

	for name, bobj := range b.Objects {
		_, ok := a.Objects[name]
		if !ok {
			result = append(result, Change{
				Type:   SymbolAdded,
				Symbol: symbolName(bobj),
			})
		}
	}

	return result
}

// CompareObjects compares two objects and reports backwards incompatible changes.
func CompareObjects(a, b *Object) []Change {
	if a.Type == AliasDeclaration {
		return compareAliases(a, b)
	}

	return compareTypes(a, b)
}

func symbolName(parent *Object, children ...string) string {
	str := fmt.Sprintf(`"%s".%s`, parent.Symbol.Package, parent.Symbol.Name)
	for _, child := range children {
		str += fmt.Sprintf(".%s", child)
	}
	return str
}

func compareTypes(a, b *Object) []Change {
	var changes []Change

	if a.Definition.Type != b.Definition.Type || a.Symbol != b.Symbol {
		changes = append(changes, Change{
			Type:   TypeChanged,
			Symbol: symbolName(a),
		})
		return changes
	}

	switch a.Definition.Type {
	case StructType:
		changes = append(changes, compareStruct(a, b)...)
	case FuncType:
		if !reflect.DeepEqual(a.Definition.Signature, b.Definition.Signature) {
			changes = append(changes, Change{
				Type:   SignatureChanged,
				Symbol: symbolName(a),
			})
		}
	case InterfaceType:
		if !reflect.DeepEqual(a.Definition.Functions, b.Definition.Functions) {
			changes = append(changes, Change{
				Type:   InterfaceChanged,
				Symbol: symbolName(a),
			})
		}
	case BasicType, MapType, SliceType, ArrayType, PointerType, ChanType:
		if !reflect.DeepEqual(a.Definition, b.Definition) {
			changes = append(changes, Change{
				Type:   TypeChanged,
				Symbol: symbolName(a),
			})
		}
	case "":
		return nil
	default:
		panic(fmt.Sprintf("unhandled type: %s", a.Definition.Type))
	}

	changes = append(changes, compareMethods(a, b)...)

	return changes
}

func compareAliases(a, b *Object) []Change {
	if !reflect.DeepEqual(a.Definition, b.Definition) {
		return []Change{{
			Type:   TypeChanged,
			Symbol: symbolName(a),
		}}
	}

	return nil
}

func compareStruct(a, b *Object) []Change {
	//TODO: report field order changes
	//TODO: report struct tag changes

	var changes []Change

	for _, aField := range a.Definition.Fields {
		found := false
		for _, bField := range b.Definition.Fields {
			if aField.Name == bField.Name {
				found = true
				if !reflect.DeepEqual(aField.Type, bField.Type) {
					changes = append(changes, Change{
						Type:   FieldChangedType,
						Symbol: symbolName(a, aField.Name),
					})
				}
			}
		}

		if !found {
			changes = append(changes, Change{
				Type:   FieldDeleted,
				Symbol: symbolName(a, aField.Name),
			})
		}
	}

	for _, bField := range b.Definition.Fields {
		found := false
		for _, aField := range a.Definition.Fields {
			if aField.Name == bField.Name {
				found = true
				break
			}
		}

		if !found {
			changes = append(changes, Change{
				Type:   FieldAdded,
				Symbol: symbolName(a, bField.Name),
			})
		}
	}

	return changes
}

type methoder interface {
	NumMethods() int
	Method(int) *types.Func
}

func compareMethods(a, b *Object) []Change {
	var changes []Change
	for _, aMethod := range a.Methods {
		found := false
		for _, bMethod := range b.Methods {
			if aMethod.Name == bMethod.Name {
				found = true
				if !reflect.DeepEqual(aMethod.Signature, bMethod.Signature) {
					changes = append(changes, Change{
						Type:   MethodSignatureChanged,
						Symbol: symbolName(a, aMethod.Name),
					})
				}
				break
			}
		}

		if !found {
			changes = append(changes, Change{
				Type:   MethodDeleted,
				Symbol: symbolName(a, aMethod.Name),
			})
		}
	}

	for _, bMethod := range b.Methods {
		found := false
		for _, aMethod := range a.Methods {
			if aMethod.Name == bMethod.Name {
				found = true
				break
			}
		}

		if !found {
			changes = append(changes, Change{
				Type:   MethodAdded,
				Symbol: symbolName(b, bMethod.Name),
			})
		}
	}

	return changes
}
