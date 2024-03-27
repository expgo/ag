package main

import "go/ast"

//go:generate enum

/*
	@Enum {
		global
		type
		func
	}
*/
type AnnotationType int

type TypedAnnotation struct {
	Type        AnnotationType
	Node        ast.Node
	Annotations *Annotations
}

type GeneratorFactory interface {
	Types() []AnnotationType
	Names() []string
	New([]TypedAnnotation) Generator
}
