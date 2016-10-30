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

	allStrategies, err := manifest.LoadAllStrategies(ParseName("jq"))
	assert.Nil(err)

	assert.Len(allStrategies, 3)
	assert.NotEqual(allStrategies[1].(BinaryStrategy).Data.OSArchData, allStrategies[2].(BinaryStrategy).Data.OSArchData)

	assert.Equal(allStrategies[1].(BinaryStrategy).Data.OSArchData,
		map[string]map[string]string{
			"windows_amd64": map[string]string{"ext": "win64.exe", "md5sum": "abababab"},
			"linux_amd64":   map[string]string{"ext": "linux64"},
			"darwin_amd64":  map[string]string{"ext": "osx-amd64"},
		})

	assert.Equal(allStrategies[2].(BinaryStrategy).Data.OSArchData,
		map[string]map[string]string{
			"windows_amd64": map[string]string{"ext": "win64.exe"},
			"linux_amd64":   map[string]string{"ext": "linux-x86_64", "md5sum": "cdcdcdcd"},
			"darwin_amd64":  map[string]string{"ext": "osx-x86_64"},
		})
}

func TestStrategyOrder(t *testing.T) {
	assert := assert.New(t)

	var strategyOrderTests = []struct {
		utility    string
		adjustment func(*MemConfig)
		result     []string
	}{
		{
			"jq",
			func(config *MemConfig) {},
			[]string{"docker", "binary", "cmdio"},
		},
		{
			"jq",
			func(config *MemConfig) {
				config.Set("strategy.priority", "binary,docker")
				config.Unset("strategy.priority")
			},
			[]string{"docker", "binary", "cmdio"},
		},
		{
			"jq",
			func(config *MemConfig) {
				config.Set("strategy.priority", "binary,docker")
			},
			[]string{"binary", "docker", "cmdio"},
		},
		{
			"jq",
			func(config *MemConfig) {
				config.Set("strategy.priority", "cmdio")
			},
			[]string{"cmdio", "docker", "binary"},
		},
		{
			"jq",
			func(config *MemConfig) {
				config.Set("strategy.xpriority", "binary")
			},
			[]string{"binary"},
		},
		// test utility level override and priority bump
		{
			"jq",
			func(config *MemConfig) {
				config.Set("strategy.jq.xpriority", "binary")
			},
			[]string{"binary"},
		},
		{
			"hugo",
			func(config *MemConfig) {
				config.Set("strategy.jq.xpriority", "binary")
			},
			[]string{"docker", "binary", "cmdio"},
		},
		{
			"jq",
			func(config *MemConfig) {
				config.Set("strategy.jq.priority", "binary")
			},
			[]string{"binary", "docker", "cmdio"},
		},
		{
			"hugo",
			func(config *MemConfig) {
				config.Set("strategy.jq.priority", "binary")
			},
			[]string{"docker", "binary", "cmdio"},
		},
		// test version level override and priority bump
		{
			"jq--1.6",
			func(config *MemConfig) {
				config.Set("strategy.jq.1.6.xpriority", "binary")
			},
			[]string{"binary"},
		},
		{
			"jq",
			func(config *MemConfig) {
				config.Set("strategy.jq.1.6.xpriority", "binary")
			},
			[]string{"docker", "binary", "cmdio"},
		},
		{
			"jq--1.6",
			func(config *MemConfig) {
				config.Set("strategy.jq.1.6.priority", "binary")
			},
			[]string{"binary", "docker", "cmdio"},
		},
		{
			"jq",
			func(config *MemConfig) {
				config.Set("strategy.jq.1.6.priority", "binary")
			},
			[]string{"docker", "binary", "cmdio"},
		},
	}

	for _, test := range strategyOrderTests {
		logger := &MemLogger{}
		config := &MemConfig{}

		manifest, err := LoadManifest(ParseName(test.utility), "testdata/manifests/jq.yaml", config, logger)
		assert.Nil(err)

		test.adjustment(config)
		assert.Equal(manifest.StrategyOrder(ParseName(test.utility)), test.result)
	}
}
