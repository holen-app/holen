package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestManifestUtils struct {
	*MemLogger
	*MemConfig
	*MemSystem
	*MemSourcePather
}

func newTestManifestFinder(selfPath string) (*TestManifestUtils, *DefaultManifestFinder) {
	tu := &TestManifestUtils{
		MemLogger:       &MemLogger{},
		MemConfig:       NewMemConfig(),
		MemSystem:       NewMemSystem(),
		MemSourcePather: &MemSourcePather{},
	}
	return tu, &DefaultManifestFinder{
		Logger:       tu.MemLogger,
		ConfigGetter: tu.MemConfig,
		System:       tu.MemSystem,
		SourcePather: tu.MemSourcePather,
		SelfPath:     selfPath,
	}
}

// func TestRun(t *testing.T) {

// 	assert := assert.New(t)

// 	nameVer := ParseName("jq")

// 	wd, _ := os.Getwd()
// 	logger := &MemLogger{}
// 	config := NewMemConfig()
// 	system := NewMemSystem()
// 	config.Set(false, "strategy.priority", "binary,docker")
// 	manifestFinder, err := newTestManifestFinder(path.Join(wd, "testdata", "single", "holen"), config, logger, system)
// 	assert.Nil(err)
// 	assert.NotNil(manifestFinder)

// 	manifest, err := manifestFinder.Find(nameVer)
// 	assert.Nil(err)

// 	runner := &MemRunner{}
// 	downloader := &MemDownloader{}
// 	manifest.Runner = runner
// 	manifest.Downloader = downloader
// 	manifest.System = system

// 	err = manifest.Run(nameVer, []string{"."})
// 	assert.Nil(err)

// 	localPath := path.Join(system.Getenv("HOME"), ".local/share/holen/bin/jq--1.5")
// 	remoteUrl := "https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64"

// 	// check download
// 	assert.Contains(downloader.Files, remoteUrl)
// 	assert.Contains(downloader.Files[remoteUrl], path.Join(system.Getenv("HOME"), ".local/share/holen/tmp"))
// 	assert.Contains(downloader.Files[remoteUrl], "jq--1.5")

// 	// check run
// 	assert.Equal(runner.History[0], fmt.Sprintf("%s .", localPath))
// }

func TestLoadAllStrategies(t *testing.T) {

	assert := assert.New(t)

	logger := &MemLogger{}
	config := NewMemConfig()
	system := NewMemSystem()

	manifest, err := LoadManifest(ParseName("jq"), "testdata/single/manifests/jq.yaml", config, logger, system)
	assert.Nil(err)

	allStrategies, err := manifest.LoadAllStrategies(ParseName("jq"))
	assert.Nil(err)

	assert.Len(allStrategies, 4)
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
				config.Set(false, "strategy.priority", "binary,docker")
				config.Unset(false, "strategy.priority")
			},
			[]string{"docker", "binary", "cmdio"},
		},
		{
			"jq",
			func(config *MemConfig) {
				config.Set(false, "strategy.priority", "binary,docker")
			},
			[]string{"binary", "docker", "cmdio"},
		},
		{
			"jq",
			func(config *MemConfig) {
				config.Set(false, "strategy.priority", "cmdio")
			},
			[]string{"cmdio", "docker", "binary"},
		},
		{
			"jq",
			func(config *MemConfig) {
				config.Set(false, "strategy.xpriority", "binary")
			},
			[]string{"binary"},
		},
		// test utility level override and priority bump
		{
			"jq",
			func(config *MemConfig) {
				config.Set(false, "strategy.jq.xpriority", "binary")
			},
			[]string{"binary"},
		},
		{
			"hugo",
			func(config *MemConfig) {
				config.Set(false, "strategy.jq.xpriority", "binary")
			},
			[]string{"docker", "binary", "cmdio"},
		},
		{
			"jq",
			func(config *MemConfig) {
				config.Set(false, "strategy.jq.priority", "binary")
			},
			[]string{"binary", "docker", "cmdio"},
		},
		{
			"hugo",
			func(config *MemConfig) {
				config.Set(false, "strategy.jq.priority", "binary")
			},
			[]string{"docker", "binary", "cmdio"},
		},
		// test version level override and priority bump
		{
			"jq--1.6",
			func(config *MemConfig) {
				config.Set(false, "strategy.jq.1.6.xpriority", "binary")
			},
			[]string{"binary"},
		},
		{
			"jq",
			func(config *MemConfig) {
				config.Set(false, "strategy.jq.1.6.xpriority", "binary")
			},
			[]string{"docker", "binary", "cmdio"},
		},
		{
			"jq--1.6",
			func(config *MemConfig) {
				config.Set(false, "strategy.jq.1.6.priority", "binary")
			},
			[]string{"binary", "docker", "cmdio"},
		},
		{
			"jq",
			func(config *MemConfig) {
				config.Set(false, "strategy.jq.1.6.priority", "binary")
			},
			[]string{"docker", "binary", "cmdio"},
		},
	}

	for _, test := range strategyOrderTests {
		logger := &MemLogger{}
		config := NewMemConfig()
		system := NewMemSystem()

		manifest, err := LoadManifest(ParseName(test.utility), "testdata/single/manifests/jq.yaml", config, logger, system)
		assert.Nil(err)

		test.adjustment(config)
		assert.Equal(manifest.StrategyOrder(ParseName(test.utility)), test.result)
	}
}

func TestDefaultLinkBinPath(t *testing.T) {
	assert := assert.New(t)

	wd, _ := os.Getwd()

	var tests = []struct {
		update   func(*TestManifestUtils)
		expected string
	}{
		{
			func(tu *TestManifestUtils) {
				tu.MemSystem.Setenv("HOME", "")
			},
			"",
		},
		{
			func(tu *TestManifestUtils) {
				tu.MemSystem.Setenv("HOME", "/home/user")
			},
			"/home/user/bin",
		},
		{
			func(tu *TestManifestUtils) {
				tu.MemSystem.Setenv("HLN_LINK_BIN_PATH", "/path/to/bin")
			},
			"/path/to/bin",
		},
		{
			func(tu *TestManifestUtils) {

				tu.MemConfig.Set(false, "link.bin_path", "/other/bin/path")
			},
			"/other/bin/path",
		},
	}

	for _, test := range tests {
		tu, manifestFinder := newTestManifestFinder("")
		test.update(tu)
		tu.MemSourcePather.TestPaths = []string{path.Join(wd, "testdata", "single", "manifests")}

		result := manifestFinder.DefaultLinkBinPath()
		assert.Equal(test.expected, result)
	}
}

func TestList(t *testing.T) {
	assert := assert.New(t)

	wd, _ := os.Getwd()

	var tests = []struct {
		name   string
		desc   bool
		result []string
	}{
		{
			"",
			false,
			[]string{"jq\n"},
		},
		{
			"",
			true,
			[]string{"jq: Lightweight and flexible command-line JSON processor\n"},
		},
	}

	for _, test := range tests {
		tu, manifestFinder := newTestManifestFinder("")
		tu.MemSourcePather.TestPaths = []string{path.Join(wd, "testdata", "single", "manifests")}

		err := manifestFinder.List(test.name, test.desc)
		assert.Nil(err)

		assert.Equal(tu.MemSystem.StdoutMessages, test.result)
	}
}

func TestLink(t *testing.T) {
	assert := assert.New(t)

	wd, _ := os.Getwd()
	manifestsPath := path.Join(wd, "testdata", "link", "manifests")

	var tests = []struct {
		link  func(*TestManifestUtils, ManifestFinder, string) error
		err   error
		links []string
	}{
		{
			func(tu *TestManifestUtils, mf ManifestFinder, binPath string) error {
				tu.MemSystem.Files[fmt.Sprintf(path.Join(manifestsPath, "util1.yaml"))] = true
				return mf.LinkSingleUtility("holen", "util1", "", binPath, false)
			},
			nil,
			[]string{"util1", "util1--1.4", "util1--1.5", "util1--1.6"},
		},
	}

	for _, test := range tests {
		tu, manifestFinder := newTestManifestFinder(path.Join(wd, "testdata", "link", "holen"))

		var err error
		tempdir, _ := ioutil.TempDir("", "link")
		defer os.RemoveAll(tempdir)
		tu.MemSystem.Setenv("HLN_LINK_BIN_PATH", tempdir)
		tu.MemSourcePather.TestPaths = []string{manifestsPath}

		err = test.link(tu, manifestFinder, tempdir)

		if test.err != nil {
			assert.NotNil(err)
			assert.Contains(err.Error(), test.err.Error())
		} else {
			assert.Nil(err)

			files, err := ioutil.ReadDir(tempdir)
			assert.Nil(err)

			fileNames := make([]string, len(files))
			for i, info := range files {
				fileNames[i] = info.Name()

				target, err := os.Readlink(path.Join(tempdir, info.Name()))
				assert.Nil(err)
				assert.Equal(target, path.Join(wd, "testdata", "link", "holen"))
			}

			// fmt.Println(fileNames)
			assert.Equal(fileNames, test.links)
		}

		// fmt.Println(tu.MemLogger.Debugs)
	}
}

// func TestLink(t *testing.T) {
// 	assert := assert.New(t)
// 	// var err error

// 	wd, _ := os.Getwd()
// 	base := path.Join(wd, "testdata", "link")

// 	var pathsTests = []struct {
// 		tempbin func() string
// 		link    func(ManifestFinder, string) error
// 		links   []string
// 		target  string
// 		err     error
// 		cleanup func()
// 	}{
// 		// link from one dir
// 		{
// 			nil,
// 			func(finder ManifestFinder, dir string) error {
// 				return finder.LinkSingle(path.Join(base, "manifests"), "", dir)
// 			},
// 			[]string{"util1", "util1--1.4", "util1--1.5", "util1--1.6", "util2", "util2--2.0"},
// 			"../holen",
// 			nil,
// 			nil,
// 		},
// 		// pass a different holen path
// 		{
// 			nil,
// 			func(finder ManifestFinder, dir string) error {
// 				return finder.LinkSingle(path.Join(base, "manifests"), path.Join(wd, "holen"), dir)
// 			},
// 			[]string{"util1", "util1--1.4", "util1--1.5", "util1--1.6", "util2", "util2--2.0"},
// 			"../../../holen",
// 			nil,
// 			nil,
// 		},
// 		// link from two dirs, earlier manifests should mask later ones
// 		{
// 			nil,
// 			func(finder ManifestFinder, dir string) error {
// 				holenPath, _ := filepath.Rel(wd, path.Join(base, "holen"))
// 				return finder.LinkMultiple([]string{path.Join(base, "manifests2"), path.Join(base, "manifests")}, holenPath, dir)
// 			},
// 			[]string{"util1", "util1--3.4", "util1--3.5", "util1--3.6", "util2", "util2--2.0"},
// 			"../holen",
// 			nil,
// 			nil,
// 		},
// 		// if a symlink to something non-holen exists
// 		{
// 			nil,
// 			func(finder ManifestFinder, dir string) error {
// 				os.Symlink("/somewhere/else", path.Join(dir, "util1"))
// 				return finder.LinkSingle(path.Join(base, "manifests"), "", dir)
// 			},
// 			[]string{"util1", "util1--1.4", "util1--1.5", "util1--1.6", "util2", "util2--2.0"},
// 			"../holen",
// 			fmt.Errorf("already exists"),
// 			nil,
// 		},
// 		// link to something with no common parent dir results in absolute paths
// 		{
// 			func() string {
// 				tempdir, _ := ioutil.TempDir("/tmp", "bin")
// 				return tempdir
// 			},
// 			func(finder ManifestFinder, dir string) error {
// 				return finder.LinkSingle(path.Join(base, "manifests"), "", dir)
// 			},
// 			[]string{"util1", "util1--1.4", "util1--1.5", "util1--1.6", "util2", "util2--2.0"},
// 			path.Join(base, "holen"),
// 			nil,
// 			nil,
// 		},
// 		// link to dir that's got a symlink in its path
// 		{
// 			func() string {
// 				p1 := path.Join(base, "folder", "other")
// 				bin := path.Join(base, "bin")
// 				os.MkdirAll(p1, 0755)

// 				os.Symlink("./folder/other", bin)

// 				return bin
// 			},
// 			func(finder ManifestFinder, dir string) error {
// 				// TODO: figure out if this and the EvalSymlinks in manifest.go
// 				// is necessary or not:
// 				// p1 := path.Join(base, "folder", "holen")
// 				// os.Symlink("../holen", p1)
// 				// return finder.LinkSingle(path.Join(base, "manifests"), p1, dir)
// 				return finder.LinkSingle(path.Join(base, "manifests"), "", dir)
// 			},
// 			[]string{"util1", "util1--1.4", "util1--1.5", "util1--1.6", "util2", "util2--2.0"},
// 			"../../holen",
// 			nil,
// 			func() {
// 				defer os.RemoveAll(path.Join(base, "folder"))
// 			},
// 		},
// 	}

// 	for _, test := range pathsTests {
// 		var tempdir string
// 		if test.tempbin != nil {
// 			tempdir = test.tempbin()
// 		} else {
// 			tempdir, _ = ioutil.TempDir(base, "bin")
// 		}
// 		defer os.RemoveAll(tempdir)

// 		logger := &MemLogger{}
// 		config := NewMemConfig()
// 		system := NewMemSystem()

// 		manifestFinder, err := newTestManifestFinder(path.Join(base, "holen"), config, logger, system)
// 		assert.Nil(err)

// 		err = test.link(manifestFinder, tempdir)
// 		if test.err != nil {
// 			assert.NotNil(err)
// 			assert.Contains(err.Error(), test.err.Error())
// 		} else {
// 			assert.Nil(err)

// 			files, err := ioutil.ReadDir(tempdir)
// 			assert.Nil(err)

// 			fileNames := make([]string, len(files))
// 			for i, info := range files {
// 				fileNames[i] = info.Name()

// 				target, err := os.Readlink(path.Join(tempdir, info.Name()))
// 				assert.Nil(err)
// 				assert.Equal(target, test.target)
// 			}

// 			assert.Equal(fileNames, test.links)
// 		}

// 		if test.cleanup != nil {
// 			test.cleanup()
// 		}
// 	}
// }
