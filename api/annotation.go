package api

import (
	"errors"
	"github.com/expgo/structure"
	"reflect"
	"strings"
)

type Annotations struct {
	Annotations []*Annotation
}

type Annotation struct {
	Doc     []string
	Name    string
	Params  []*AnnotationParam
	Extends []*AnnotationExtend
	Comment string
}

type AnnotationParam struct {
	Doc     []string
	Key     string
	Value   structure.ValueWrapper
	Comment string
}

type AnnotationExtend struct {
	Doc     []string
	Name    string
	Values  []structure.ValueWrapper
	Value   structure.ValueWrapper
	Comment string
}

type Float struct {
	V float64 `@Float ","? `
}

func (f Float) Value() any { return f.V }

type Int struct {
	V int `@(("-" | "+")? Int) ","? `
}

func (i Int) Value() any {
	return i.V
}

type Uint struct {
	V uint `@Int ","? `
}

func (u Uint) Value() any {
	return u.V
}

type String struct {
	V string `@(String | Ident) ","? `
}

func (s String) Value() any {
	return s.V
}

type Boolean bool

func (b *Boolean) Capture(values []string) error {
	*b = values[0] == "true"
	return nil
}

type Bool struct {
	V Boolean `@("true" | "false") ","? `
}

func (b Bool) Value() any {
	return bool(b.V)
}

type Slice struct {
	V []structure.ValueWrapper `"{" @@* "}"`
}

func (s Slice) Value() any {
	return s.V
}

var defaultBoolValue = Bool{V: true}

func (a *Annotation) To(t any) (err error) {
	if t == nil {
		return errors.New("the input parameter cannot be nil")
	}

	if a.Params != nil {
		err = structure.WalkField(t, func(fieldValue reflect.Value, structField reflect.StructField, rootValues []reflect.Value) error {
			switch fieldValue.Kind() {
			case reflect.Ptr, reflect.Struct:
				return nil
			default:
			}

			var ap *AnnotationParam = nil

			for _, p := range a.Params {
				if strings.EqualFold(structField.Name, p.Key) {
					ap = p
					break
				}
			}

			if ap == nil {
				return nil
			}

			if fieldValue.Kind() == reflect.Bool && ap.Value == nil {
				ap.Value = defaultBoolValue
			}

			if ap.Value != nil {
				value := structure.MustConvertToType(ap.Value, fieldValue.Type())
				if structure.SetFieldBySetMethod(fieldValue, value, structField, rootValues[len(rootValues)-1]) {
					return nil
				}
				return structure.SetField(fieldValue, value)
			}

			return nil
		})
	}

	return
}

func (anns *Annotations) FindAnnotationByName(name string) *Annotation {
	if len(anns.Annotations) > 0 {
		for _, a := range anns.Annotations {
			if strings.EqualFold(a.Name, name) {
				return a
			}
		}
	}

	return nil
}
