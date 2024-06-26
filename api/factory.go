package api

import (
	"go/ast"
)

//go:generate ag

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
	FileInfo    *FileInfo
}

type GeneratorFactory interface {
	Annotations() map[string][]AnnotationType // a map of name -> []AnnotationType
	New([]*TypedAnnotation) (Generator, error)
}
