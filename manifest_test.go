package main

import (
	"os"
	"path"
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

	manifestFinder.Logger = NullLogger{}
	manifestFinder.ConfigGetter = NullConfigGetter{}

	manifest, err := manifestFinder.Find(nameVer)
	assert.Nil(err)

	manifest.Runner = NullRunner{}
	manifest.Downloader = NullDownloader{}
	manifest.System = NullSystem{}

	err = manifest.Run(nameVer, []string{})
	assert.Nil(err)
}
