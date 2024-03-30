package ag

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// @Enum eabc
type Eabc int

// @Enum abc
type Abc struct {
}

func (abc *Abc) Init(
	// @Enum a
	a int,
	b int, // @Enum b
) {
}

func TestAAA(t *testing.T) {
	const src = `package main

//ExampleFunction is an example
func ExampleFunction(
    //Param1 is the first param
    Param1 int,
	// Param2 is for aaa
    // @Enum Param2 is the second param
    Param2 string, // @123
) {
    //function body
}`
	// 创建一个新的token.FileSet
	fset := token.NewFileSet()

	// 解析源码以获得AST
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	//file, err := ast.ParseFile(fset, "./annotation.go", nil, ast.ParseComments)
	if err != nil {
		panic(err)
	}

	// 遍历AST中的所有声明
	for _, decl := range file.Decls {
		// 确保声明是函数
		if fn, ok := decl.(*ast.FuncDecl); ok {
			// 遍历函数的参数
			for _, field := range fn.Type.Params.List {
				for _, name := range field.Names {
					// 找到Param2参数
					if name.Name == "Param2" {
						// 获取Param2参数的位置
					}
				}
			}
		}
	}
}

func TestInspectAnnotation(t *testing.T) {
	const src = `package main

// @Enum{cat, dog}
type MyEnum int

const (
   MyEnumA MyEnum = 0
   MyEnumB MyEnum = 1
)

// @Singleton
// struct doc comment
type MyInitTest struct {
	Name string
} // struct inline comment

func (m *MyInitTest) Init(
    abc int, // @V("123")
    ddd string, // @V(abc)
) {
}

func (m MyInitTest) Init1(
    abc int, // @V("123")
) {
}

//ExampleFunction is an example
func ExampleFunction(
    //Param1 is the first param
    Param1 int,
	// Param2 is for aaa
    // @Enum Param2 is the second param
    Param2 string, // @123
) {
    //function body
}


func abc(){}
`
	// 创建一个新的token.FileSet
	fset := token.NewFileSet()

	// 解析源码以获得AST
	_, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	//file, err := ast.ParseFile(fset, "./annotation.go", nil, ast.ParseComments)
	if err != nil {
		panic(err)
	}

	//InspectMapper[ast.TypeSpec, any](file, fset, func(x *ast.TypeSpec) *any {
	//	fmt.Printf("InspectMapper.TypeSpec: %#v\n", x)
	//	return nil
	//})
	//
	//InspectMapper[ast.FuncDecl, any](file, fset, func(x *ast.FuncDecl) *any {
	//	fmt.Printf("InspectMapper.FuncDecl: %#v\n", x)
	//	return nil
	//})
}
