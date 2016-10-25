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
	logger := &MemLogger{}
	config := &MemConfig{}
	config.Set("strategy.priority", "binary,docker")
	manifestFinder, err := NewManifestFinder(path.Join(wd, "testdata", "manifests"), config, logger)
	assert.Nil(err)
	assert.NotNil(manifestFinder)

	manifest, err := manifestFinder.Find(nameVer)
	assert.Nil(err)

	runner := &MemRunner{}
	downloader := &MemDownloader{}
	system := &MemSystem{runtime.GOOS, runtime.GOARCH, 1000, 1000, make(map[string]bool), []string{}, []string{}, make(map[string][]string)}
	manifest.Runner = runner
	manifest.Downloader = downloader
	manifest.System = system

	err = manifest.Run(nameVer, []string{"."})
	assert.Nil(err)

	localPath := path.Join(os.Getenv("HOME"), ".local/share/holen/bin/jq--1.5")
	remoteUrl := "https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64"

	// check download
	assert.Contains(downloader.Files, remoteUrl)
	assert.Contains(downloader.Files[remoteUrl], path.Join(os.Getenv("HOME"), ".local/share/holen/tmp"))
	assert.Contains(downloader.Files[remoteUrl], "jq--1.5")

	// check run
	assert.Equal(runner.History[0], fmt.Sprintf("%s .", localPath))
}

func TestLoadAllStrategies(t *testing.T) {

	assert := assert.New(t)

	logger := &MemLogger{}
	config := &MemConfig{}

	manifest, err := LoadManifest(ParseName("jq"), "testdata/manifests/jq.yaml", config, logger)
	assert.Nil(err)

	allStrategies, err := manifest.LoadAllStrategies()
	assert.Nil(err)

	assert.Contains(allStrategies, "docker")
	assert.Contains(allStrategies, "binary")
	assert.Len(allStrategies["docker"], 1)
	assert.Len(allStrategies["binary"], 2)
	assert.NotEqual(allStrategies["binary"][0].(BinaryStrategy).Data.OSArchData, allStrategies["binary"][1].(BinaryStrategy).Data.OSArchData)
}
