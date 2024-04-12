package api

import (
	"errors"
	"golang.org/x/mod/modfile"
	"os"
	"path/filepath"
	"strings"
)

type FileInfo struct {
	ModuleName           string
	ModuleAbsLocalPath   string
	FileFullPath         string
	FileFullAbsLocalPath string
}

func GetFileInfo(filename string) (*FileInfo, error) {
	inputFile, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	ret := &FileInfo{
		FileFullAbsLocalPath: inputFile,
	}

	// To find go.mod file
	dirPath := filepath.Dir(inputFile)
	for {
		if _, err = os.Stat(filepath.Join(dirPath, "go.mod")); err == nil {
			break
		}
		dirPath = filepath.Dir(dirPath)
		if dirPath == "/" || dirPath == "." {
			return nil, errors.New("go.mod not found")
		}
	}

	goModData, err := os.ReadFile(filepath.Join(dirPath, "go.mod"))
	if err != nil {
		return nil, err
	}

	ret.ModuleAbsLocalPath = dirPath
	ret.ModuleName = modfile.ModulePath(goModData)
	relativePath, err := filepath.Rel(dirPath, filepath.Dir(inputFile))
	if err != nil {
		return nil, err
	}
	ret.FileFullPath = strings.Replace(ret.ModuleName+"/"+relativePath, `\`, `/`, -1)

	return ret, nil
}
