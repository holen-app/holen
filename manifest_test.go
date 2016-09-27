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
	system := &MemSystem{runtime.GOOS, runtime.GOARCH, 1000, 1000, make(map[string]bool)}
	manifest.Runner = runner
	manifest.Downloader = downloader
	manifest.System = system

	err = manifest.Run(nameVer, []string{"."})
	assert.Nil(err)

	for _, foo := range logger.Debugs {
		fmt.Println(foo)
	}
	for url, foo := range downloader.Files {
		fmt.Println(url, foo)
	}
	for _, foo := range runner.History {
		fmt.Println(foo)
	}
}
