package main

import (
	"crypto/sha1"
	"embed"
	"fmt"
	"github.com/expgo/ag/api"
	"github.com/expgo/structure"
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
	filename    string
	fileSuffix  string
	packageMode bool
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
			pp.writeReplace()
		}

		println("do mod tidy")
		pp.runCommand(pp.baseDir, "go", "mod", "tidy")
		println("do mod update")
		pp.runCommand(pp.baseDir, "go", "get", "-u", "./...")
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

func (pp *PluginProgram) writeReplace() {
	fi, err := api.GetFileInfo(pp.filename)
	if err != nil {
		panic(err)
	}

	pp.runCommand(pp.baseDir, "bash", "-c", fmt.Sprintf(`echo "replace %s => %s" >> %s`, fi.ModuleName, fi.ModuleAbsLocalPath, AGFileGoMod.Val()))
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
	println("run ag plugin program, workDir: ", workDir)
	pp.runCommand(workDir, agExe, "-file="+pp.filename, "-suffix="+pp.fileSuffix, "-package-mode="+structure.MustConvertTo[string](pp.packageMode))
}

func (pp *PluginProgram) runCommand(workDir string, name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if len(workDir) > 0 {
		cmd.Dir = workDir
	}

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			println(string(exitErr.Stderr))
		} else {
			println(err.Error())
		}
	}
}
