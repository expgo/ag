package ag

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/expgo/ag/api"
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

func (ans *Annotations) toApi() *api.Annotations {
	result := &api.Annotations{}

	if len(ans.Annotations) == 0 {
		return result
	}

	for _, a := range ans.Annotations {
		result.Annotations = append(result.Annotations, a.toApi())
	}

	return result
}

type ClosedParenthesis struct {
	Pos               lexer.Position
	ClosedParenthesis string `")"`
}

type Params struct {
	List              []*AnnotationParam `"(" @@*`
	ClosedParenthesis ClosedParenthesis  `@@`
}

func (p *Params) toApi() []*api.AnnotationParam {
	if len(p.List) == 0 {
		return nil
	}

	result := []*api.AnnotationParam{}

	for _, param := range p.List {
		result = append(result, param.toApi())
	}

	return result
}

type ClosedBracket struct {
	Pos           lexer.Position
	ClosedBracket string `"}"`
}

type Extends struct {
	List          []*AnnotationExtend `"{" @@*`
	ClosedBracket ClosedBracket       `@@`
}

func (e Extends) toApi() []*api.AnnotationExtend {
	if len(e.List) == 0 {
		return nil
	}

	result := []*api.AnnotationExtend{}

	for _, extend := range e.List {
		result = append(result, extend.toApi())
	}

	return result
}

type Annotation struct {
	BeforeUseless *string    `(~(Comment | "@"))*`
	Doc           []*Comment `@@*`
	Name          Name       `"@" @@`
	Params        *Params    `@@?`
	Extends       *Extends   `@@?`
	Comment       *Comment   `@@?`
	AfterUseless  *string    `(~(Comment | "@"))*`
}

func (a *Annotation) toApi() *api.Annotation {
	result := &api.Annotation{
		Doc:     toApiDoc(a.Doc),
		Name:    a.Name.Text,
		Comment: toApiComment(a.Comment),
	}

	if a.Params != nil {
		result.Params = a.Params.toApi()
	}

	if a.Extends != nil {
		result.Extends = a.Extends.toApi()
	}

	return result
}

type AnnotationParam struct {
	Pos     lexer.Position
	Doc     []*Comment `@@*`
	Key     Key        `@@`
	Value   api.Value  `@@? ","?`
	Comment *Comment   `@@?`
}

func (ap *AnnotationParam) toApi() *api.AnnotationParam {
	return &api.AnnotationParam{
		Doc:     toApiDoc(ap.Doc),
		Key:     ap.Key.Text,
		Value:   ap.Value,
		Comment: toApiComment(ap.Comment),
	}
}

type AnnotationExtend struct {
	Pos     lexer.Position
	Doc     []*Comment  `@@*`
	Name    Name        `@@`
	Values  []api.Value `("(" @@* ")")?`
	Value   api.Value   `("=" @@)? ","?`
	Comment *Comment    `@@?`
}

func (ae AnnotationExtend) toApi() *api.AnnotationExtend {
	return &api.AnnotationExtend{
		Doc:     toApiDoc(ae.Doc),
		Name:    ae.Name.Text,
		Values:  ae.Values,
		Value:   ae.Value,
		Comment: toApiComment(ae.Comment),
	}
}

func toApiDoc(comments []*Comment) []string {
	if len(comments) == 0 {
		return nil
	}

	result := []string{}

	for _, comment := range comments {
		result = append(result, comment.Text)
	}

	return result
}

func toApiComment(comment *Comment) string {
	if comment == nil {
		return ""
	}
	return comment.Text
}

var annotationParser = participle.MustBuild[Annotations](
	participle.Lexer(lexer.NewTextScannerLexer(func(s *scanner.Scanner) {
		s.Mode &^= scanner.SkipComments
	})),
	participle.Union[api.Value](api.Bool{}, api.Float{}, api.Int{}, api.Uint{}, api.String{}, api.Slice{}),
	participle.Unquote("String"),
)

func fixComments(annotations *Annotations, err error) (*api.Annotations, error) {
	if err != nil {
		return nil, err
	}

	for ai, annotation := range annotations.Annotations {
		if annotation.Params != nil {
			for pi, param := range annotation.Params.List {
				if param.Comment != nil &&
					param.Comment.Pos.Line != param.Key.Pos.Line &&
					pi+1 < len(annotation.Params.List) {
					annotation.Params.List[pi+1].Doc = append([]*Comment{param.Comment}, annotation.Params.List[pi+1].Doc...)
					param.Comment = nil
				}
			}
		}

		if annotation.Extends != nil {
			for ei, extend := range annotation.Extends.List {
				if extend.Comment != nil &&
					extend.Comment.Pos.Line != extend.Name.Pos.Line &&
					ei+1 < len(annotation.Extends.List) {
					annotation.Extends.List[ei+1].Doc = append([]*Comment{extend.Comment}, annotation.Extends.List[ei+1].Doc...)
					extend.Comment = nil
				}
			}
		}

		if annotation.Comment != nil &&
			!(annotation.Comment.Pos.Line == annotation.Name.Pos.Line ||
				(annotation.Params != nil && annotation.Params.ClosedParenthesis.Pos.Line == annotation.Comment.Pos.Line) ||
				(annotation.Extends != nil && annotation.Extends.ClosedBracket.Pos.Line == annotation.Comment.Pos.Line)) &&
			ai+1 < len(annotations.Annotations) {
			annotations.Annotations[ai+1].Doc = append([]*Comment{annotation.Comment}, annotations.Annotations[ai+1].Doc...)
			annotation.Comment = nil
		}
	}

	return annotations.toApi(), err
}

func ParseAnnotation(fileName string, text string) (*api.Annotations, error) {
	return fixComments(annotationParser.ParseString(fileName, text))
}
