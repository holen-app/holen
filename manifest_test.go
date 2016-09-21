package main

import (
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

type NullRunner struct{}

func (nr NullRunner) CheckCommand(command string, args []string) bool {
	return true
}

func (nr NullRunner) RunCommand(command string, args []string) error {
	return nil
}

type NullDownloader struct{}

func (nd NullDownloader) DownloadFile(url, path string) error {
	return nil
}

func (nd NullDownloader) PullDockerImage(image string) error {
	return nil
}

type NullSystem struct{}

func (ns NullSystem) OS() string {
	return runtime.GOOS
}

func (ns NullSystem) Arch() string {
	return runtime.GOARCH
}

func (ns NullSystem) FileExists(localPath string) bool {
	return true
}

func (ns NullSystem) MakeExecutable(localPath string) error {
	return nil
}

func TestRun(t *testing.T) {

	assert := assert.New(t)

	nameVer := ParseName("jq")

	wd, _ := os.Getwd()
	manifestFinder, err := NewManifestFinder(path.Join(wd, "testdata"))
	assert.Nil(err)
	assert.NotNil(manifestFinder)

	manifest, err := manifestFinder.Find(nameVer)
	assert.Nil(err)

	manifest.Runner = NullRunner{}
	manifest.Downloader = NullDownloader{}
	manifest.System = NullSystem{}

	err = manifest.Run(nameVer, []string{})
	assert.Nil(err)
}
