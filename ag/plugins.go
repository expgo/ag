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

//go:generate ag
/*
	@Enum {
		Exe = "ag"
		MainGo = "main.go"
		GoMod = "go.mod"
		GoSum = "go.sum"
	}
*/
type AGFile string

func (x AGFile) GetFilePath(baseDir string) string {
	return filepath.Join(baseDir, x.Val())
}

type PluginProgram struct {
	Plugins []string
	baseDir string
	// -------------
	devPlugin string
	rebuild   bool
	devMode   bool
	// -------------
	filename   string
	fileSuffix string
}

func getPathHash(plugins []string) string {
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

func runPlugins(filename string, fileSuffix string, plugins []string, devPlugin string, rebuild bool) {
	pp := &PluginProgram{
		Plugins:    plugins,
		devPlugin:  devPlugin,
		rebuild:    rebuild,
		filename:   filename,
		fileSuffix: fileSuffix,
	}

	if len(devPlugin) > 0 {
		pp.Plugins = append(pp.Plugins, devPlugin)
	}

	hash := getPathHash(plugins)
	pp.baseDir = filepath.Join(getExePath(), hash)
	pp.devMode = len(pp.devPlugin) > 0

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
	mainFile, err := os.Create(AGFileMainGo.GetFilePath(pp.baseDir))
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
	newCreate := false

	_, err := os.Stat(AGFileMainGo.GetFilePath(pp.baseDir))
	if os.IsNotExist(err) {
		println("write plugin main.go")
		pp.writeMain()
		newCreate = true
	}

	if newCreate {
		println("do mod init")
		pp.runCommand(pp.baseDir, "go", "mod", "init", "main")

		if pp.devMode {
			// write replace to go.mod
			workDir, err := os.Getwd()
			if err != nil {
				panic(err)
			}
			pp.runCommand(pp.baseDir, "echo", fmt.Sprintf("replace %s => %s", pp.devPlugin, workDir), ">>", AGFileGoMod.Val())
		}

		println("do mod tidy")
		pp.runCommand(pp.baseDir, "go", "mod", "tidy")
		println("do mod update")
		pp.runCommand(pp.baseDir, "go", "get", "-u", ".")
	}

	if newCreate || pp.devMode {
		println("do mod tidy")
		pp.runCommand(pp.baseDir, "go", "mod", "tidy")
	}

	println("build plugin program")
	pp.runCommand(pp.baseDir, "go", "build", "-o", AGFileExe.Val(), AGFileMainGo.Val())

	if !pp.devMode {
		println("remove go files")
		pp.runCommand(pp.baseDir, "rm", AGFileGoMod.Val(), AGFileGoSum.Val(), AGFileMainGo.Val())
	}
}

func (pp *PluginProgram) run() {
	// 判断pp.exeFile是否存在
	agExe := AGFileExe.GetFilePath(pp.baseDir)
	_, err := os.Stat(agExe)
	build := false
	if os.IsNotExist(err) {
		pp.build()
		build = true
	}

	if !build && (pp.rebuild || pp.devMode) {
		pp.runCommand(pp.baseDir, "rm", agExe)
		pp.build()
		build = true
	}

	workDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	println("run ag plugin program")
	pp.runCommand(workDir, agExe, "-file", pp.filename, "-suffix", pp.fileSuffix)
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
