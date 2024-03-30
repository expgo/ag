package ag

import (
	"github.com/expgo/ag/api"
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestParseAnnotation(t *testing.T) {
	singleComment := "// comment"
	multiComment := "/* comment */"

	tests := []struct {
		name     string
		fileName string
		text     string
		want     *api.Annotations
		wantErr  bool
	}{
		{
			name:     "Empty file name and text",
			fileName: "",
			text:     "",
			want:     &api.Annotations{},
			wantErr:  false,
		},
		{
			name:     "Valid file name but empty text",
			fileName: "file.go",
			text:     "",
			want:     &api.Annotations{},
			wantErr:  false,
		},
		{
			name:     "only one annotation name",
			fileName: "file.go",
			text:     "@tag",
			want: &api.Annotations{
				Annotations: []*api.Annotation{
					{
						Name: "tag",
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "two annotation name",
			fileName: "file.go",
			text: `
@tag
@sql
`,
			want: &api.Annotations{
				Annotations: []*api.Annotation{
					{
						Name: "tag",
					},
					{
						Name: "sql",
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "two annotation name with real comments",
			fileName: "file.go",
			text: `

some real comment 1
some real comment 2

// tag comment 1
/* tag comment 2
   tag comment 3
   tag comment 4
*/
@tag // tag inline comment
some real sql comment 0
some real sql comment 1
some real sql comment 2

/* sql comment 1
sql comment2
*/
// sql comment 3
@sql
some real sql comment
`,
			want: &api.Annotations{
				Annotations: []*api.Annotation{
					{
						Doc: []string{
							"// tag comment 1",
							`/* tag comment 2
   tag comment 3
   tag comment 4
*/`,
						},
						Name:    "tag",
						Comment: "// tag inline comment",
					},
					{
						Doc: []string{
							`/* sql comment 1
sql comment2
*/`,
							"// sql comment 3",
						},
						Name: "sql",
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "two annotation with params",
			fileName: "file.go",
			text: `
@tag(disable, string = "str\"ing" , int=123, double=456.7, bool = true, params = { "abc", 321, 123.4, false})
@sql(code int32, name string, message=string)
`,
			want: &api.Annotations{
				Annotations: []*api.Annotation{
					{
						Name: "tag",
						Params: []*api.AnnotationParam{
							{
								Key: "disable",
							},
							{
								Key:   "string",
								Value: api.String{V: "str\"ing"},
							},
							{
								Key:   "int",
								Value: api.Int{V: 123},
							},
							{
								Key:   "double",
								Value: api.Float{V: 456.7},
							},
							{
								Key:   "bool",
								Value: api.Bool{V: true},
							},
							{
								Key:   "params",
								Value: api.Slice{V: []api.Value{api.String{V: "abc"}, api.Int{V: 321}, api.Float{V: 123.4}, api.Bool{V: false}}},
							},
						},
					},
					{
						Name: "sql",
						Params: []*api.AnnotationParam{
							{
								Key:   "code",
								Value: api.String{V: "int32"},
							},
							{
								Key:   "name",
								Value: api.String{V: "string"},
							},
							{
								Key:   "message",
								Value: api.String{V: "string"},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "two annotation with params and extends",
			fileName: "file.go",
			text: `
@tag(disable, string = "str\"ing" , int=123, double=456.7, bool = true) {
	Good  ,
    GoodWithIntValue = 12 
    GoodWithStrValue = "str" 
    GoodWithParams ("string", 123, 456.7, true )  // comment
    GoodWithAll ("string", 123, 456.7, false ) = 89 /* comment */
}
@sql(code int32, name string, message=string)
`,
			want: &api.Annotations{
				Annotations: []*api.Annotation{
					{
						Name: "tag",
						Params: []*api.AnnotationParam{
							{
								Key: "disable",
							},
							{
								Key:   "string",
								Value: api.String{V: "str\"ing"},
							},
							{
								Key:   "int",
								Value: api.Int{V: 123},
							},
							{
								Key:   "double",
								Value: api.Float{V: 456.7},
							},
							{
								Key:   "bool",
								Value: api.Bool{V: true},
							},
						},
						Extends: []*api.AnnotationExtend{
							{
								Name: "Good",
							},
							{
								Name:  "GoodWithIntValue",
								Value: api.Int{V: 12},
							},
							{
								Name:  "GoodWithStrValue",
								Value: api.String{V: "str"},
							},
							{
								Name: "GoodWithParams",
								Values: []api.Value{
									api.String{V: "string"},
									api.Int{V: 123},
									api.Float{V: 456.7},
									api.Bool{V: true},
								},
								Comment: singleComment,
							},
							{
								Name: "GoodWithAll",
								Values: []api.Value{
									api.String{V: "string"},
									api.Int{V: 123},
									api.Float{V: 456.7},
									api.Bool{V: false},
								},
								Value:   api.Int{V: 89},
								Comment: multiComment,
							},
						},
					},
					{
						Name: "sql",
						Params: []*api.AnnotationParam{
							{
								Key:   "code",
								Value: api.String{V: "int32"},
							},
							{
								Key:   "name",
								Value: api.String{V: "string"},
							},
							{
								Key:   "message",
								Value: api.String{V: "string"},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "two annotation with params and extends on multiple lines and comments",
			fileName: "file.go",
			text: `

// tag comment 1
// tag comment 2
/* tag comment 3
tag comment 4
*/
@tag(
	// upper disable comment
	disable, // disable comment
    // string comment 1
    // string comment 2
	string = "str\"ing" ,
    /* int comment 1
       int comment 2
    */
	int=123
    double = 456.7  // double inline comment
    bool = true) {
	// comment 1
    // comment 2
	Good  
	/* comment 1
       comment 2
       comment 3 */
    GoodWithIntValue = 12 
    GoodWithStrValue = "str" 
    GoodWithParams ("string", 123, 456.7, true )  // comment
    GoodWithAll ("string", 123, 456.7, false ) = 89 /* comment */
}
// sql comment 0
/* sql comment 1
 sql comment 2
*/
@sql(code int32, name string, message=string) // sql inline comment
`,
			want: &api.Annotations{
				Annotations: []*api.Annotation{
					{
						Doc: []string{
							"// tag comment 1",
							"// tag comment 2",
							`/* tag comment 3
tag comment 4
*/`,
						},
						Name: "tag",
						Params: []*api.AnnotationParam{
							{
								Doc: []string{
									"// upper disable comment",
								},
								Key:     "disable",
								Comment: "// disable comment",
							},
							{
								Doc: []string{
									"// string comment 1",
									"// string comment 2",
								},
								Key:   "string",
								Value: api.String{V: "str\"ing"},
							},
							{
								Doc: []string{
									`/* int comment 1
       int comment 2
    */`,
								},
								Key:   "int",
								Value: api.Int{V: 123},
							},
							{
								Key:     "double",
								Value:   api.Float{V: 456.7},
								Comment: "// double inline comment",
							},
							{
								Key:   "bool",
								Value: api.Bool{V: true},
							},
						},
						Extends: []*api.AnnotationExtend{
							{
								Doc: []string{
									"// comment 1",
									"// comment 2",
								},
								Name: "Good",
							},
							{
								Doc: []string{
									`/* comment 1
       comment 2
       comment 3 */`,
								},
								Name:  "GoodWithIntValue",
								Value: api.Int{V: 12},
							},
							{
								Name:  "GoodWithStrValue",
								Value: api.String{V: "str"},
							},
							{
								Name: "GoodWithParams",
								Values: []api.Value{
									api.String{V: "string"},
									api.Int{V: 123},
									api.Float{V: 456.7},
									api.Bool{V: true},
								},
								Comment: singleComment,
							},
							{
								Name: "GoodWithAll",
								Values: []api.Value{
									api.String{V: "string"},
									api.Int{V: 123},
									api.Float{V: 456.7},
									api.Bool{V: false},
								},
								Value:   api.Int{V: 89},
								Comment: multiComment,
							},
						},
					},
					{
						Doc: []string{
							"// sql comment 0",
							`/* sql comment 1
 sql comment 2
*/`,
						},
						Name: "sql",
						Params: []*api.AnnotationParam{
							{
								Key:   "code",
								Value: api.String{V: "int32"},
							},
							{
								Key:   "name",
								Value: api.String{V: "string"},
							},
							{
								Key:   "message",
								Value: api.String{V: "string"},
							},
						},
						Comment: "// sql inline comment",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAnnotation(tt.fileName, tt.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAnnotation() error = %v, wantErr %v", err, tt.wantErr)
			}

			// TODO try to change cmp to github.com/stretchr/testify
			opt := cmp.FilterPath(ignorePosFields, cmp.Ignore())
			diff := cmp.Diff(tt.want, got, opt)
			if len(diff) > 0 {
				t.Errorf("ParseAnnotation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func ignorePosFields(path cmp.Path) bool {
	// 遍历路径中的每个步骤
	for _, step := range path {
		if t, ok := step.(cmp.StructField); ok {
			// 如果步骤是结构体字段并且名字为"Pos"，则返回true以忽略
			if t.Name() == "Pos" {
				return true
			}
		}
	}
	// 对于其他字段，不忽略
	return false
}
