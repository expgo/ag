package ag

import "go/ast"

//go:generate enum

/*
	@Enum {
		global
		type
		func
		funcRecv
		funcField
	}
*/
type AnnotationType int

type TypedAnnotation struct {
	Type        AnnotationType
	Node        ast.Node
	Annotations *Annotations
	Parent      *TypedAnnotation
}

type GeneratorFactory interface {
	Annotations() map[string][]AnnotationType // a map of name -> []AnnotationType
	New([]*TypedAnnotation) Generator
}
