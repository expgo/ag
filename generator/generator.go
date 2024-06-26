package generator

import (
	"bytes"
	"fmt"
	"github.com/expgo/ag"
	"github.com/expgo/ag/api"
	"github.com/expgo/factory"
	"golang.org/x/tools/imports"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

func filterTypedAnnotation(typedAnnotations []*api.TypedAnnotation, annotationMap map[string][]api.AnnotationType) []*api.TypedAnnotation {
	filteredAnnotations := []*api.TypedAnnotation{}
	for _, ta := range typedAnnotations {
		if ta.Annotations != nil {
			for name, annotationTypes := range annotationMap {
				if an := ta.Annotations.FindAnnotationByName(name); an != nil {
					for _, annotationType := range annotationTypes {
						if annotationType == ta.Type {
							filteredAnnotations = append(filteredAnnotations, ta)
							break
						}
					}
				}
			}
		}
	}

	distinctAnnotations := []*api.TypedAnnotation{}
	for _, annotationType := range filteredAnnotations {
		contains := false
		for _, da := range distinctAnnotations {
			if da == annotationType {
				contains = true
				break
			}
		}

		if !contains {
			distinctAnnotations = append(distinctAnnotations, annotationType)
		}
	}

	return distinctAnnotations
}

func getAllTypedAnnotations(filename string, typeMaps map[api.AnnotationType][]string, packageMode bool) (result []*api.TypedAnnotation, packageName string, e error) {
	filename, e = filepath.Abs(filename)
	if e != nil {
		return
	}

	if packageMode {
		result, packageName, e = ag.ParseFile(filename, typeMaps)

		workDir, err := os.Getwd()
		if err != nil {
			e = err
			return
		}
		// 获取当前目录下除filename和_test.go后缀的所有go文件
		files, err := filepath.Glob(filepath.Join(workDir, "*.go"))
		if err != nil {
			return nil, "", err
		}
		for _, file := range files {
			if file != filename && !strings.HasSuffix(file, "_test.go") {
				ta, _, err := ag.ParseFile(file, typeMaps)
				if err != nil {
					return nil, "", err
				}
				result = append(result, ta...)
			}
		}

		return
	} else {
		return ag.ParseFile(filename, typeMaps)
	}
}

func GenerateFile(filename string, outputSuffix string, packageMode bool) {
	factories := factory.FindInterfaces[api.GeneratorFactory]()
	if len(factories) == 0 {
		println("No GeneratorFactory was found for the annotation generator.")
		return
	}

	// sort factories by order
	sort.Slice(factories, func(i, j int) bool {
		x := factories[i]
		y := factories[j]

		xOrder := api.OrderNormal
		if xO, ok := y.(api.IOrder); ok {
			xOrder = xO.Order()
		}
		yOrder := api.OrderNormal
		if yO, ok := y.(api.IOrder); ok {
			yOrder = yO.Order()
		}

		if xOrder == yOrder {
			xTypePath := reflect.TypeOf(x).Elem().PkgPath()
			yTypePath := reflect.TypeOf(y).Elem().PkgPath()
			return strings.Compare(xTypePath, yTypePath) < 0
		}
		return xOrder.Val()-yOrder.Val() < 0
	})

	typeMaps := map[api.AnnotationType][]string{}
	for _, f := range factories {
		annotations := f.Annotations()
		for name, types := range annotations {
			for _, t := range types {
				typeMaps[t] = append(typeMaps[t], name)
			}
		}
	}

	typedAnnotations, packageName, err := getAllTypedAnnotations(filename, typeMaps, packageMode)
	if err != nil {
		println(err.Error())
		return
	}

	if len(typedAnnotations) == 0 {
		println("No annotation found.")
		return
	}

	gens := []api.Generator{}

	for _, f := range factories {
		if ftas := filterTypedAnnotation(typedAnnotations, f.Annotations()); len(ftas) > 0 {
			gen, e := f.New(ftas)
			if e != nil {
				panic(e)
			}
			if gen != nil {
				gens = append(gens, gen)
			}
		}
	}

	if len(gens) == 0 {
		println("No generator found.")
		return
	}

	buf := bytes.NewBuffer([]byte{})

	plugins := []string{}
	for _, gen := range gens {
		t := reflect.TypeOf(gen)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		plugins = append(plugins, t.PkgPath())
	}

	println("run with plugins: \n", strings.Join(plugins, "\n"))

	buf.WriteString("// Code generated by https://github.com/expgo/ag DO NOT EDIT.\n")
	buf.WriteString("// Plugins: \n")
	for _, plugin := range plugins {
		buf.WriteString(fmt.Sprintf("//   - %s \n", plugin))
	}
	buf.WriteString("\n\n")

	// write package
	buf.WriteString("package " + packageName)
	buf.WriteString("\n\n")

	buf.WriteString("import (\n")
	for i, gen := range gens {
		println("Creating " + plugins[i] + " imports")
		importList := gen.GetImports()
		for _, imp := range importList {
			buf.WriteString("\t\"" + imp + "\"\n")
		}
	}
	buf.WriteString(")\n\n")

	for i, gen := range gens {
		println("Creating " + plugins[i] + " const")
		err = gen.WriteConst(buf)
		if err != nil {
			panic(err)
		}
	}
	buf.WriteString("\n\n")

	for i, gen := range gens {
		println("Creating " + plugins[i] + " func")
		err = gen.WriteInitFunc(buf)
		if err != nil {
			panic(err)
		}
	}
	buf.WriteString("\n\n")

	for i, gen := range gens {
		println("Creating " + plugins[i] + " body")
		err = gen.WriteBody(buf)
		if err != nil {
			panic(err)
		}
	}

	formatted, err := imports.Process(packageName, buf.Bytes(), nil)
	if err != nil {
		panic(fmt.Errorf("generate: error formatting code %s\n\n%s", err, buf.String()))
	}

	outFilePath := fmt.Sprintf("%s%s.go", strings.TrimSuffix(filename, filepath.Ext(filename)), outputSuffix)
	if strings.HasSuffix(filename, "_test.go") {
		outFilePath = strings.Replace(outFilePath, "_test"+outputSuffix+".go", outputSuffix+"_test.go", 1)
	}

	mode := int(0o644)
	err = os.WriteFile(outFilePath, formatted, os.FileMode(mode))
	if err != nil {
		panic(fmt.Errorf("failed writing to file %s: %s", outFilePath, err))
	}
	println("Finish write : " + outFilePath)
}
