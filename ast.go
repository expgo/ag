package ag

import (
	"fmt"
	"github.com/expgo/ag/api"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

func parseFile(inputFile string) (*ast.File, *token.FileSet, error) {
	fileSet := token.NewFileSet()
	fileNode, err := parser.ParseFile(fileSet, inputFile, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("generate: error parsing input file '%s': %s", inputFile, err)
	}

	return fileNode, fileSet, nil
}

func getAnnotations(parseName string, names []string, comments string) (*api.Annotations, error) {
	lowComments := strings.ToLower(comments)
	for _, name := range names {
		if strings.Contains(lowComments, "@"+strings.ToLower(name)) {
			ag, err := ParseAnnotation(parseName, comments)
			if err != nil {
				return nil, fmt.Errorf("parse annotation err: %v", err)
			}
			return ag, nil
		}
	}
	return nil, nil
}

func getRecvType(fd *ast.FuncDecl) *ast.TypeSpec {
	if fd.Recv != nil {
		if fd.Recv.NumFields() == 1 {
			var recvTypeIdent *ast.Ident
			switch tt := fd.Recv.List[0].Type.(type) {
			case *ast.Ident:
				recvTypeIdent = tt

			case *ast.StarExpr:
				if itt, ok := tt.X.(*ast.Ident); ok {
					recvTypeIdent = itt
				}
			}

			if recvTypeIdent != nil && recvTypeIdent.Obj != nil {
				if recvType, ok := recvTypeIdent.Obj.Decl.(*ast.TypeSpec); ok {
					return recvType
				}
			}
		}
	}

	return nil
}

func inspectFile(fileNode *ast.File, fileSet *token.FileSet, typeMaps map[api.AnnotationType][]string, fileInfo *api.FileInfo) (result []*api.TypedAnnotation, e error) {
	ast.Inspect(fileNode, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.TypeSpec:
			if names, ok := typeMaps[api.AnnotationTypeType]; ok {
				if decl.Doc == nil {
					decl.Doc = FindDocLocationCommentGroup(fileNode, fileSet, decl.Pos())
				}
				if decl.Comment == nil {
					decl.Comment = FindCommentLocationCommentGroup(fileNode, fileSet, decl.Pos())
				}

				var comment string
				if decl.Doc != nil {
					comment = decl.Doc.Text()
				} else if decl.Comment != nil {
					comment = decl.Comment.Text()
				}

				annotations, err := getAnnotations(decl.Name.Name, names, comment)
				if err != nil {
					e = err
					return false
				}

				if annotations != nil {
					result = append(result, &api.TypedAnnotation{api.AnnotationTypeType, decl, annotations, nil, fileInfo})
				}
			}
		case *ast.FuncDecl:
			if decl.Doc == nil {
				decl.Doc = FindDocLocationCommentGroup(fileNode, fileSet, decl.Pos())
			}

			var comment string
			if decl.Doc != nil {
				comment = decl.Doc.Text()
			}

			var annotations *api.Annotations
			if names, ok := typeMaps[api.AnnotationTypeFunc]; ok {
				annotations, e = getAnnotations(decl.Name.Name, names, comment)
				if e != nil {
					return false
				}
			}
			funcAnnotation := &api.TypedAnnotation{api.AnnotationTypeFunc, decl, annotations, nil, fileInfo}

			if names, ok := typeMaps[api.AnnotationTypeFuncRecv]; ok {
				if decl.Recv != nil {
					recvType := getRecvType(decl)
					if recvType != nil {
						if recvType.Doc == nil {
							recvType.Doc = FindDocLocationCommentGroup(fileNode, fileSet, recvType.Pos())
						}
						if recvType.Comment == nil {
							recvType.Comment = FindCommentLocationCommentGroup(fileNode, fileSet, recvType.Pos())
						}

						comment = ""
						if recvType.Doc != nil {
							comment = recvType.Doc.Text()
						} else if recvType.Comment != nil {
							comment = recvType.Comment.Text()
						}

						recvAnnotations, err := getAnnotations(recvType.Name.Name, names, comment)
						if err != nil {
							e = err
							return false
						}
						if recvAnnotations != nil {
							result = append(result, &api.TypedAnnotation{api.AnnotationTypeFuncRecv, recvType, recvAnnotations, funcAnnotation, fileInfo})
						}
					}
				}
			}

			if names, ok := typeMaps[api.AnnotationTypeFuncField]; ok {
				for _, field := range decl.Type.Params.List {
					if field.Doc == nil {
						field.Doc = FindDocLocationCommentGroup(fileNode, fileSet, field.Pos())
					}
					if field.Comment == nil {
						field.Comment = FindCommentLocationCommentGroup(fileNode, fileSet, field.Pos())
					}

					comment = ""
					if field.Doc != nil {
						comment = field.Doc.Text()
					} else if field.Comment != nil {
						comment = field.Comment.Text()
					}

					fieldAnnotations, err := getAnnotations(field.Names[0].Name, names, comment)
					if err != nil {
						e = err
						return false
					}
					if fieldAnnotations != nil {
						result = append(result, &api.TypedAnnotation{api.AnnotationTypeFuncField, field, fieldAnnotations, funcAnnotation, fileInfo})
					}
				}
			}

			if funcAnnotation.Annotations != nil {
				result = append(result, funcAnnotation)
			}
		}

		return true
	})

	return
}

func ParseFile(filename string, typeMaps map[api.AnnotationType][]string) (result []*api.TypedAnnotation, packageName string, e error) {
	fileInfo, err := api.GetFileInfo(filename)
	if err != nil {
		return nil, "", err
	}

	filename, _ = filepath.Abs(filename)

	fileNode, fileSet, err := parseFile(filename)
	if err != nil {
		return nil, "", err
	}

	packageName = fileNode.Name.Name

	// global TypedAnnotation
	if names, ok := typeMaps[api.AnnotationTypeGlobal]; ok {
		for _, cg := range fileNode.Comments {
			if strings.HasPrefix(cg.List[len(cg.List)-1].Text, "//go:generate") {
				annotations, err := getAnnotations(fileNode.Name.Name, names, cg.Text())
				if err != nil {
					return nil, "", err
				}
				if annotations != nil {
					result = append(result, &api.TypedAnnotation{api.AnnotationTypeGlobal, fileNode, annotations, nil, fileInfo})
				}
			}
		}
	}

	// other TypedAnnotation
	ta, err := inspectFile(fileNode, fileSet, typeMaps, fileInfo)
	if err != nil {
		return nil, "", err
	}
	result = append(result, ta...)

	return
}

func FindDocLocationCommentGroup(fileNode *ast.File, fileSet *token.FileSet, pos token.Pos) *ast.CommentGroup {
	indentPos := fileSet.Position(pos)

	for _, commentGroup := range fileNode.Comments {
		commentGroupPos := fileSet.Position(commentGroup.End())

		if commentGroupPos.Line+1 == indentPos.Line && commentGroupPos.Offset < indentPos.Offset {
			return commentGroup
		}
	}

	return nil
}

func FindCommentLocationCommentGroup(fileNode *ast.File, fileSet *token.FileSet, pos token.Pos) *ast.CommentGroup {
	indentPos := fileSet.Position(pos)

	for _, commentGroup := range fileNode.Comments {
		commentGroupPos := fileSet.Position(commentGroup.End())

		if commentGroupPos.Line == indentPos.Line && indentPos.Offset < commentGroupPos.Offset {
			return commentGroup
		}
	}

	return nil
}
