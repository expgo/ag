package api

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetMod(t *testing.T) {
	fi, err := GetFileInfo("a/b/c/d.go")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, "github.com/expgo/ag", fi.ModuleName)
	assert.Equal(t, "github.com/expgo/ag/api/a/b/c", fi.FileFullPath)
}
