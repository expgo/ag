package main

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/expgo/factory"
	"github.com/expgo/generic/stream"
	"github.com/expgo/structure"
	"reflect"
	"strings"
	"text/scanner"
)

type Key struct {
	Pos  lexer.Position
	Text string `@Ident "="?`
}

type Name struct {
	Pos  lexer.Position
	Text string `@Ident`
}

type Comment struct {
	Pos  lexer.Position
	Text string `@Comment`
}

type Annotations struct {
	Annotations []*Annotation `@@*`
}

type ClosedParenthesis struct {
	Pos               lexer.Position
	ClosedParenthesis string `")"`
}

type Params struct {
	List              []*AnnotationParam `"(" @@*`
	ClosedParenthesis ClosedParenthesis  `@@`
}

type ClosedBracket struct {
	Pos           lexer.Position
	ClosedBracket string `"}"`
}

type Extends struct {
	List          []*AnnotationExtend `"{" @@*`
	ClosedBracket ClosedBracket       `@@`
}

type Annotation struct {
	BeforeUseless *string    `(~(Comment | "@"))*`
	Comments      []*Comment `@@*`
	Name          Name       `"@" @@`
	Params        *Params    `@@?`
	Extends       *Extends   `@@?`
	Comment       *Comment   `@@?`
	AfterUseless  *string    `(~(Comment | "@"))*`
}

type AnnotationParam struct {
	Pos      lexer.Position
	Comments []*Comment `@@*`
	Key      Key        `@@`
	Value    Value      `@@? ","?`
	Comment  *Comment   `@@?`
}

type AnnotationExtend struct {
	Pos      lexer.Position
	Comments []*Comment `@@*`
	Name     Name       `@@`
	Values   []Value    `("(" @@* ")")?`
	Value    Value      `("=" @@)? ","?`
	Comment  *Comment   `@@?`
}

type Value interface{ Value() any }

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

//type Unknown struct {
//	V string `@Ident ","? `
//}
//
//func (u Unknown) Value() {}

var annotationParser = participle.MustBuild[Annotations](
	participle.Lexer(lexer.NewTextScannerLexer(func(s *scanner.Scanner) {
		s.Mode &^= scanner.SkipComments
	})),
	participle.Union[Value](Bool{}, Float{}, Int{}, Uint{}, String{}),
	participle.Unquote("String"),
)

func fixComments(annotationGroup *Annotations, err error) (*Annotations, error) {
	if err != nil {
		return annotationGroup, err
	}

	for ai, annotation := range annotationGroup.Annotations {
		if annotation.Params != nil {
			for pi, param := range annotation.Params.List {
				if param.Comment != nil &&
					param.Comment.Pos.Line != param.Key.Pos.Line &&
					pi+1 < len(annotation.Params.List) {
					annotation.Params.List[pi+1].Comments = append([]*Comment{param.Comment}, annotation.Params.List[pi+1].Comments...)
					param.Comment = nil
				}
			}
		}

		if annotation.Extends != nil {
			for ei, extend := range annotation.Extends.List {
				if extend.Comment != nil &&
					extend.Comment.Pos.Line != extend.Name.Pos.Line &&
					ei+1 < len(annotation.Extends.List) {
					annotation.Extends.List[ei+1].Comments = append([]*Comment{extend.Comment}, annotation.Extends.List[ei+1].Comments...)
					extend.Comment = nil
				}
			}
		}

		if annotation.Comment != nil &&
			!(annotation.Comment.Pos.Line == annotation.Name.Pos.Line ||
				(annotation.Params != nil && annotation.Params.ClosedParenthesis.Pos.Line == annotation.Comment.Pos.Line) ||
				(annotation.Extends != nil && annotation.Extends.ClosedBracket.Pos.Line == annotation.Comment.Pos.Line)) &&
			ai+1 < len(annotationGroup.Annotations) {
			annotationGroup.Annotations[ai+1].Comments = append([]*Comment{annotation.Comment}, annotationGroup.Annotations[ai+1].Comments...)
			annotation.Comment = nil
		}
	}

	return annotationGroup, err
}

var defaultBoolValue = any(Bool{V: true}).(Value)

func AnnotationParamsTo[T any](val *T, a *Annotation) (t *T, err error) {
	t = val
	if t == nil {
		t = factory.New[T]()
	}

	if a.Params != nil {
		err = structure.WalkField(t, func(fieldValue reflect.Value, structField reflect.StructField, rootValues []reflect.Value) error {
			switch fieldValue.Kind() {
			case reflect.Ptr, reflect.Struct:
				return nil
			default:
			}

			var ap *AnnotationParam = nil

			for _, p := range a.Params.List {
				if strings.EqualFold(structField.Name, p.Key.Text) {
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
				value := structure.MustConvertToType(ap.Value.Value(), fieldValue.Type())
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

func ParseAnnotation(fileName string, text string) (*Annotations, error) {
	return fixComments(annotationParser.ParseString(fileName, text))
}

func GetCommentsText(comments []*Comment) string {
	if len(comments) == 0 {
		return ""
	}

	return strings.Join(stream.Must(stream.Map[*Comment, string](stream.Of(comments), func(comment *Comment) (string, error) {
		return comment.Text, nil
	}).ToSlice()), "\n")
}

func GetCommentText(comment *Comment) string {
	if comment == nil {
		return ""
	}
	return comment.Text
}

func (ag *Annotations) FindAnnotationByName(name string) *Annotation {
	if len(ag.Annotations) > 0 {
		for _, a := range ag.Annotations {
			if strings.EqualFold(a.Name.Text, name) {
				return a
			}
		}
	}

	return nil
}
