package main

import (
	"io"
	"text/template"
)

type Generator interface {
	GetImports() []string
	WriteConst(wr io.Writer) error
	WriteInitFunc(wr io.Writer) error
	WriteBody(wr io.Writer) error
}

type BaseGenerator[T any] struct {
	Tmpl     *template.Template
	DataList []*T
}

func (bg *BaseGenerator[T]) GetImports() []string {
	return nil
}

func (bg *BaseGenerator[T]) ExecuteTemplate(wr io.Writer, name string) error {
	tmpl := bg.Tmpl.Lookup(name)

	if tmpl != nil {
		for _, data := range bg.DataList {
			if err := tmpl.Execute(wr, data); err != nil {
				return err
			}
		}
	}

	return nil
}

func (bg *BaseGenerator[T]) WriteConst(wr io.Writer) error {
	return bg.ExecuteTemplate(wr, "const")
}

func (bg *BaseGenerator[T]) WriteInitFunc(wr io.Writer) error {
	return bg.ExecuteTemplate(wr, "init")
}

func (bg *BaseGenerator[T]) WriteBody(wr io.Writer) error {
	return bg.ExecuteTemplate(wr, "body")
}
