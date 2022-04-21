package compat

import (
	"go/types"
)

func ConvertObject(obj types.Object) *Object {
	res := &Object{
		Symbol: Symbol{
			Package: obj.Pkg().Path(),
			Name:    obj.Name(),
		},
	}

	switch obj := obj.(type) {
	case *types.TypeName:
		res.Type = TypeDeclaration
		if obj.IsAlias() {
			res.Type = AliasDeclaration
		}
	case *types.Func:
		res.Type = FuncDeclaration
	case *types.Var:
		res.Type = VarDeclaration
	case *types.Const:
		res.Type = ConstDeclaration
	}

	if res.Type == AliasDeclaration {
		res.Definition = typeToShallowDefinition(obj.Type())
		return res
	}

	if _, ok := obj.Type().Underlying().(*types.Basic); ok {
		res.Definition = typeToShallowDefinition(obj.Type())
	} else {
		res.Definition = typeToDefinition(obj.Type())
	}

	switch typ := obj.Type().(type) {
	case *types.Named:
		res.Methods = methoderToFuncs(true, typ)
	}

	return res
}

func typeToShallowDefinition(typ types.Type) *Definition {
	switch typ := typ.(type) {
	case *types.Basic:
		return &Definition{
			Type:   BasicType,
			Symbol: &Symbol{Name: typ.Name()},
		}
	case *types.Named:
		tn := typ.Obj()
		var pkg string
		if tn.Pkg() != nil {
			pkg = tn.Pkg().Path()
		}
		return &Definition{
			Symbol: &Symbol{Package: pkg, Name: tn.Name()},
		}
	}

	return typeToDefinition(typ)
}

func typeToDefinition(typ types.Type) *Definition {
	underlying := typ.Underlying()
	switch underlying := underlying.(type) {
	case *types.Struct:
		return structToDefinition(underlying)
	case *types.Signature:
		return signatureToDefinition(underlying)
	case *types.Slice:
		return sliceToDefinition(underlying)
	case *types.Array:
		return arrayToDefinition(underlying)
	case *types.Chan:
		return chanToDefinition(underlying)
	case *types.Map:
		return mapToDefinition(underlying)
	case *types.Pointer:
		return pointerToDefinition(underlying)
	case *types.Interface:
		return interfaceToDefinition(underlying)
	}
	panic("unhandled type")
}

func methoderToFuncs(exportedOnly bool, typ methoder) []*Func {
	var funcs []*Func
	for i := 0; i < typ.NumMethods(); i++ {
		method := typ.Method(i)
		if exportedOnly && !method.Exported() {
			continue
		}
		funcs = append(funcs, &Func{
			Name:      method.Name(),
			Signature: signatureToSignature(method.Type().(*types.Signature)),
		})
	}
	return funcs
}

func structToDefinition(typ *types.Struct) *Definition {
	def := &Definition{
		Type: StructType,
	}

	for i := 0; i < typ.NumFields(); i++ {
		field := typ.Field(i)
		if !field.Exported() {
			continue
		}
		def.Fields = append(def.Fields, varToField(field))
	}

	return def
}

func signatureToDefinition(typ *types.Signature) *Definition {
	sig := signatureToSignature(typ)
	return &Definition{
		Type:      FuncType,
		Signature: &sig,
	}
}

func signatureToSignature(typ *types.Signature) Signature {
	return Signature{
		Params:   tupleToSymbols(typ.Params()),
		Results:  tupleToSymbols(typ.Results()),
		Variadic: typ.Variadic(),
	}
}

func sliceToDefinition(typ *types.Slice) *Definition {
	return &Definition{
		Type: SliceType,
		Elem: typeToShallowDefinition(typ.Elem()),
	}
}

func arrayToDefinition(typ *types.Array) *Definition {
	return &Definition{
		Type: ArrayType,
		Len:  typ.Len(),
		Elem: typeToShallowDefinition(typ.Elem()),
	}
}

func chanToDefinition(typ *types.Chan) *Definition {
	return &Definition{
		Type:    ChanType,
		ChanDir: typ.Dir(),
		Elem:    typeToShallowDefinition(typ.Elem()),
	}
}

func mapToDefinition(typ *types.Map) *Definition {
	return &Definition{
		Type: MapType,
		Key:  typeToShallowDefinition(typ.Key()),
		Elem: typeToShallowDefinition(typ.Elem()),
	}
}

func pointerToDefinition(typ *types.Pointer) *Definition {
	return &Definition{
		Type: PointerType,
		Elem: typeToShallowDefinition(typ.Elem()),
	}
}

func interfaceToDefinition(typ *types.Interface) *Definition {
	return &Definition{
		Type:      InterfaceType,
		Functions: methoderToFuncs(false, typ),
	}
}

func tupleToSymbols(tup *types.Tuple) []*Definition {
	var res []*Definition
	for i := 0; i < tup.Len(); i++ {
		field := tup.At(i)
		res = append(res, typeToShallowDefinition(field.Type()))
	}

	return res
}

func varToField(obj *types.Var) *Field {
	return &Field{
		Name: obj.Name(),
		Type: typeToShallowDefinition(obj.Type()),
	}
}
