package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {

	assert := assert.New(t)

	nameVer := ParseName("jq")

	wd, _ := os.Getwd()
	logger := &MemLogger{}
	config := &MemConfig{}
	system := NewMemSystem()
	config.Set("strategy.priority", "binary,docker")
	manifestFinder, err := NewManifestFinder(path.Join(wd, "testdata", "single", "holen"), config, logger, system)
	assert.Nil(err)
	assert.NotNil(manifestFinder)

	manifest, err := manifestFinder.Find(nameVer)
	assert.Nil(err)

	runner := &MemRunner{}
	downloader := &MemDownloader{}
	manifest.Runner = runner
	manifest.Downloader = downloader
	manifest.System = system

	err = manifest.Run(nameVer, []string{"."})
	assert.Nil(err)

	localPath := path.Join(system.Getenv("HOME"), ".local/share/holen/bin/jq--1.5")
	remoteUrl := "https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64"

	// check download
	assert.Contains(downloader.Files, remoteUrl)
	assert.Contains(downloader.Files[remoteUrl], path.Join(system.Getenv("HOME"), ".local/share/holen/tmp"))
	assert.Contains(downloader.Files[remoteUrl], "jq--1.5")

	// check run
	assert.Equal(runner.History[0], fmt.Sprintf("%s .", localPath))
}

func TestLoadAllStrategies(t *testing.T) {

	assert := assert.New(t)

	logger := &MemLogger{}
	config := &MemConfig{}
	system := NewMemSystem()

	manifest, err := LoadManifest(ParseName("jq"), "testdata/single/manifests/jq.yaml", config, logger, system)
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
		system := NewMemSystem()

		manifest, err := LoadManifest(ParseName(test.utility), "testdata/single/manifests/jq.yaml", config, logger, system)
		assert.Nil(err)

		test.adjustment(config)
		assert.Equal(manifest.StrategyOrder(ParseName(test.utility)), test.result)
	}
}

func TestPaths(t *testing.T) {
	assert := assert.New(t)

	wd, _ := os.Getwd()
	localDir := path.Join(wd, "testdata", "single", "manifests")

	var pathsTests = []struct {
		adjustment func(*MemConfig, *MemSystem)
		result     []string
	}{
		{
			nil,
			[]string{localDir},
		},
		{
			func(config *MemConfig, sys *MemSystem) {
				sys.Setenv("HLN_PATH", "/path/one:/path/two")
			},
			[]string{"/path/one", "/path/two", localDir},
		},
		{
			func(config *MemConfig, sys *MemSystem) {
				sys.Setenv("HLN_PATH_POST", "/path/one:/path/two")
			},
			[]string{"/path/one", "/path/two", localDir},
		},
		{
			func(config *MemConfig, sys *MemSystem) {
				sys.Setenv("HLN_PATH", "/path/one")
				sys.Setenv("HLN_PATH_POST", "/path/two")
				config.Set("manifest.path", "/path/config")
			},
			[]string{"/path/one", "/path/config", "/path/two", localDir},
		},
	}

	for _, test := range pathsTests {

		logger := &MemLogger{}
		config := &MemConfig{}
		system := NewMemSystem()

		manifestFinder, err := NewManifestFinder(path.Join(wd, "testdata", "single", "holen"), config, logger, system)
		assert.Nil(err)

		if test.adjustment != nil {
			test.adjustment(config, system)
		}

		result := manifestFinder.Paths()
		assert.Equal(result, test.result)
	}
}

func TestList(t *testing.T) {
	assert := assert.New(t)
	var err error

	wd, _ := os.Getwd()

	logger := &MemLogger{}
	config := &MemConfig{}
	system := NewMemSystem()

	system.Setenv("HLN_PATH", "/path/one")
	manifestFinder, err := NewManifestFinder(path.Join(wd, "testdata", "single", "holen"), config, logger, system)
	assert.Nil(err)

	err = manifestFinder.List()
	assert.Nil(err)

	assert.Equal(system.StdoutMessages, []string{"jq\n"})
}

func TestLink(t *testing.T) {
	assert := assert.New(t)
	// var err error

	wd, _ := os.Getwd()
	base := path.Join(wd, "testdata", "link")

	tempdir, _ := ioutil.TempDir(base, "bin")
	defer os.RemoveAll(tempdir)

	logger := &MemLogger{}
	config := &MemConfig{}
	system := NewMemSystem()

	manifestFinder, err := NewManifestFinder(path.Join(base, "holen"), config, logger, system)
	assert.Nil(err)

	manifestFinder.LinkSingle(path.Join(base, "manifests"), "", tempdir)

	files, err := ioutil.ReadDir(tempdir)
	assert.Nil(err)

	fileNames := make([]string, len(files))
	for i, info := range files {
		fileNames[i] = info.Name()

		target, err := os.Readlink(path.Join(tempdir, info.Name()))
		assert.Nil(err)
		assert.Equal(target, "../holen")
	}

	assert.Equal(fileNames, []string{"util1", "util1--1.4", "util1--1.5", "util1--1.6", "util2", "util2--2.0"})
}
