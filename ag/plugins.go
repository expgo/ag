package main

import (
	"crypto/sha1"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed main.tmpl
var mainTmpl embed.FS

type PluginProgram struct {
	Plugins   []string
	baseDir   string
	exeFile   string
	exeMainGo string
	// -------------
	filename   string
	fileSuffix string
}

func getPluginSuffix(plugins []string) string {
	hasher := sha1.New()
	hasher.Write([]byte(strings.Join(plugins, ",")))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func getExePath() string {
	exeFile, err := os.Executable()
	if err != nil {
		panic(err)
	}

	return filepath.Join(filepath.Dir(exeFile), ".ag")
}

func runPlugins(filename string, fileSuffix string, plugins []string) {
	suffix := getPluginSuffix(plugins)

	pp := &PluginProgram{
		Plugins:    plugins,
		baseDir:    getExePath(),
		filename:   filename,
		fileSuffix: fileSuffix,
	}
	pp.exeFile = filepath.Join(pp.baseDir, "ag-"+suffix)
	pp.exeMainGo = filepath.Join(pp.baseDir, "ag-"+suffix+".go")

	pp.run()
}

func (pp *PluginProgram) writeMain() {
	if err := os.MkdirAll(pp.baseDir, os.ModePerm); err != nil {
		panic(err)
	}

	tmpl := template.New("ag")
	tmpl = template.Must(tmpl.ParseFS(mainTmpl, "*.tmpl"))
	tmpl = tmpl.Lookup("main")

	var mainFile *os.File
	mainFile, err := os.Create(pp.exeMainGo)
	if err != nil {
		panic(err)
	}
	defer mainFile.Close()

	err = tmpl.Execute(mainFile, pp)
	if err != nil {
		panic(err)
	}
}

func (pp *PluginProgram) build() {
	pp.writeMain()
	pp.runCommand(pp.baseDir, "go", "mod", "init", "main")
	pp.runCommand(pp.baseDir, "go", "mod", "tidy")
	pp.runCommand(pp.baseDir, "go", "get", "-u", ".")
	pp.runCommand(pp.baseDir, "go", "mod", "tidy")
	pp.runCommand(pp.baseDir, "go", "build", "-o", filepath.Base(pp.exeFile), filepath.Base(pp.exeMainGo))
	pp.runCommand(pp.baseDir, "rm", "go.mod", "go.sum", pp.exeMainGo)
}

func (pp *PluginProgram) run() {
	// 判断pp.exeFile是否存在
	_, err := os.Stat(pp.exeFile)
	if os.IsNotExist(err) {
		pp.build()
	}

	workDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	pp.runCommand(workDir, pp.exeFile, "-file", pp.filename, "-suffix", pp.fileSuffix)
}

func (pp *PluginProgram) runCommand(workDir string, name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	if len(workDir) > 0 {
		cmd.Dir = workDir
	}

	// 执行命令并获取输出
	output, err := cmd.Output()
	println(string(output))
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			println(string(exitErr.Stderr))
		} else {
			println(err.Error())
		}
	}
}
