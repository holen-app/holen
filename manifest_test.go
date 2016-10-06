package main

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {

	assert := assert.New(t)

	nameVer := ParseName("jq")

	wd, _ := os.Getwd()
	manifestFinder, err := NewManifestFinder(path.Join(wd, "testdata"))
	assert.Nil(err)
	assert.NotNil(manifestFinder)

	logger := &MemLogger{}
	config := &MemConfig{}
	config.Set("strategy.priority", "binary,docker")
	manifestFinder.Logger = logger
	manifestFinder.ConfigGetter = config

	manifest, err := manifestFinder.Find(nameVer)
	assert.Nil(err)

	runner := &MemRunner{}
	downloader := &MemDownloader{}
	system := &MemSystem{runtime.GOOS, runtime.GOARCH, 1000, 1000, make(map[string]bool), []string{}}
	manifest.Runner = runner
	manifest.Downloader = downloader
	manifest.System = system

	err = manifest.Run(nameVer, []string{"."})
	assert.Nil(err)

	localPath := path.Join(os.Getenv("HOME"), ".local/share/holen/bin/jq--1.5")
	remoteUrl := "https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64"

	// check download
	assert.Contains(downloader.Files, remoteUrl)
	assert.Equal(downloader.Files[remoteUrl], localPath)

	// check run
	assert.Equal(runner.History[0], fmt.Sprintf("%s .", localPath))
}
