package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitSourceUrl(t *testing.T) {
	assert := assert.New(t)

	var testCases = []struct {
		spec, url string
	}{
		{
			"test/repo",
			"https://github.com/test/repo.git",
		},
		{
			"/absolute/path/repo.git",
			"/absolute/path/repo.git",
		},
		{
			"bitbucket.org/test/repo",
			"https://bitbucket.org/test/repo.git",
		},
	}

	for _, test := range testCases {
		system := NewMemSystem()
		logger := &MemLogger{}
		runner := &MemRunner{}
		gs := GitSource{
			system,
			logger,
			runner,
			"test",
			test.spec,
		}

		assert.Equal(gs.Name(), "test")
		assert.Equal(gs.Spec(), test.spec)
		assert.Equal(gs.Info(), fmt.Sprintf("git source: %s", test.url))
	}
}

func TestGitSourceUpdate(t *testing.T) {
	assert := assert.New(t)

	baseDir := "/tmp"

	var testCases = []struct {
		mod func(*MemSystem)
		cmd string
	}{
		{
			func(ms *MemSystem) {
				return
			},
			fmt.Sprintf("git clone https://github.com/test/repo.git %s/test", baseDir),
		},
		{
			func(ms *MemSystem) {
				ms.Files[fmt.Sprintf("%s/test", baseDir)] = true
			},
			"git pull",
		},
	}

	for _, test := range testCases {
		system := NewMemSystem()
		logger := &MemLogger{}
		runner := &MemRunner{}
		gs := GitSource{
			system,
			logger,
			runner,
			"test",
			"test/repo",
		}

		test.mod(system)
		assert.Nil(gs.Update(baseDir))
		assert.Equal([]string{test.cmd}, runner.History)
	}
}

func TestGitSourceDelete(t *testing.T) {
	assert := assert.New(t)

	tempdir, _ := ioutil.TempDir("", "hash")
	defer os.RemoveAll(tempdir)

	gs := GitSource{
		NewMemSystem(),
		&MemLogger{},
		&MemRunner{},
		"test",
		"test/repo",
	}

	repoPath := filepath.Join(tempdir, "test")
	os.Mkdir(repoPath, 0755)
	assert.Nil(gs.Delete(tempdir))

	_, err := os.Stat(repoPath)
	assert.True(os.IsNotExist(err))
}

func TestSourceManagerAdd(t *testing.T) {
	assert := assert.New(t)

	logger := &MemLogger{}
	config := NewMemConfig()
	system := NewMemSystem()
	sm := &RealSourceManager{
		Logger:       logger,
		ConfigClient: config,
		System:       system,
	}

	assert.Nil(sm.Add(false, "test", "test/repo"))
	assert.Equal(map[string]string{"source.test": "test/repo"}, config.UserConfig)

	err := sm.Add(false, "test", "test/repo")
	assert.Contains(err.Error(), "already exists")
	assert.Equal(map[string]string{"source.test": "test/repo"}, config.UserConfig)
}

func TestSourceManagerList(t *testing.T) {
	assert := assert.New(t)

	logger := &MemLogger{}
	config := NewMemConfig()
	system := NewMemSystem()
	sm := &RealSourceManager{
		Logger:       logger,
		ConfigClient: config,
		System:       system,
	}

	assert.Nil(sm.Add(false, "test", "test/repo"))
	assert.Nil(sm.List())

	assert.Contains(system.StdoutMessages, "test: test/repo (git source: https://github.com/test/repo.git)\n")
	assert.Contains(system.StdoutMessages, "main: justone/holen-manifests (git source: https://github.com/justone/holen-manifests.git)\n")
}

func TestSourceManagerPaths(t *testing.T) {
	assert := assert.New(t)

	tempdir, _ := ioutil.TempDir("", "paths")
	defer os.RemoveAll(tempdir)

	logger := &MemLogger{}
	config := NewMemConfig()
	system := NewMemSystem()
	system.Setenv("HOME", tempdir)

	sm := &RealSourceManager{
		Logger:       logger,
		ConfigClient: config,
		System:       system,
	}

	dataPath, _ := system.DataPath()
	testManifestsPath := filepath.Join(dataPath, "manifests", "test", "manifests")
	mainManifestsPath := filepath.Join(dataPath, "manifests", "main")
	system.Files[testManifestsPath] = true

	assert.Nil(sm.Add(false, "test", "test/repo"))

	paths, err := sm.Paths("")
	assert.Nil(err)
	assert.Equal(2, len(paths))
	assert.Contains(paths, testManifestsPath)
	assert.Contains(paths, mainManifestsPath)

	paths, err = sm.Paths("main")
	assert.Nil(err)
	assert.Equal(1, len(paths))
	assert.Contains(paths, mainManifestsPath)

	paths, err = sm.Paths("bogus")
	assert.NotNil(err)
	assert.Contains(err.Error(), "not found")
}
