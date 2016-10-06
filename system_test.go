package main

import (
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystem(t *testing.T) {
	assert := assert.New(t)

	ds := DefaultSystem{}

	assert.Equal(ds.OS(), runtime.GOOS)
	assert.Equal(ds.Arch(), runtime.GOARCH)
	assert.Equal(ds.UID(), os.Getuid())
	assert.Equal(ds.GID(), os.Getgid())

	tempdir, _ := ioutil.TempDir("", "dir")
	defer os.RemoveAll(tempdir)

	filePath := path.Join(tempdir, "file")
	assert.False(ds.FileExists(filePath))
	os.Mkdir(filePath, 0755)
	assert.True(ds.FileExists(filePath))

	os.Chmod(filePath, 0644)
	info, _ := os.Stat(filePath)
	assert.False(strings.Contains(info.Mode().Perm().String(), "x"))
	assert.Nil(ds.MakeExecutable(filePath))
	info, _ = os.Stat(filePath)
	assert.True(strings.Contains(info.Mode().Perm().String(), "x"))
}
