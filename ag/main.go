package main

import (
	"flag"
	"fmt"
	"github.com/expgo/ag/generator"
	"os"
	"strings"
)

type Plugins []string

// String 是 flag.Value 接口的一部分，它返回值的默认文本表示形式
func (p *Plugins) String() string {
	return strings.Join(*p, ",")
}

func (p *Plugins) Set(value string) error {
	*p = append(*p, value)
	return nil
}

func main() {
	var filename string
	var fileSuffix string
	var rebuild bool
	var plugins Plugins
	var devPlugin string

	flag.StringVar(&filename, "file", "", "The file is used to generate the annotation file.")
	flag.StringVar(&fileSuffix, "file-suffix", "_ag", "Changes the default filename suffix of _ag to something else.")
	flag.BoolVar(&rebuild, "rebuild", false, "If plugin is used and rebuild is set to true, the plugin program will be rebuild.")
	flag.Var(&plugins, "plugin", "Add extended plugins to the Annotation Generator.")
	flag.StringVar(&devPlugin, "dev-plugin", "", "Used when develop ag plugin.")

	flag.Parse()

	if len(filename) == 0 {
		filename, _ = os.LookupEnv("GOFILE")

		if len(filename) == 0 {
			fmt.Fprintf(os.Stdout, "Usage of %s:\n", os.Args[0])
			flag.PrintDefaults()
			return
		}
	}

	if len(plugins) > 0 {
		runPlugins(filename, fileSuffix, plugins, devPlugin, rebuild)
	} else {
		generator.GenerateFile(filename, fileSuffix)
	}
}
