package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestGitSourceUtils struct {
	*MemSystem
	*MemLogger
	*MemRunner
}

func newTestGitSource(name, spec string) (*TestGitSourceUtils, *GitSource) {
	tu := &TestGitSourceUtils{
		MemSystem: NewMemSystem(),
		MemLogger: &MemLogger{},
		MemRunner: &MemRunner{},
	}
	return tu, &GitSource{
		tu.MemSystem,
		tu.MemLogger,
		tu.MemRunner,
		name,
		spec,
	}
}

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
		_, gs := newTestGitSource("test", test.spec)

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
		tu, gs := newTestGitSource("test", "test/repo")

		test.mod(tu.MemSystem)
		assert.Nil(gs.Update(baseDir))
		assert.Equal([]string{test.cmd}, tu.MemRunner.History)
	}
}

func TestGitSourceDelete(t *testing.T) {
	assert := assert.New(t)

	tempdir, _ := ioutil.TempDir("", "hash")
	defer os.RemoveAll(tempdir)

	_, gs := newTestGitSource("test", "test/repo")

	repoPath := filepath.Join(tempdir, "test")
	os.Mkdir(repoPath, 0755)
	assert.Nil(gs.Delete(tempdir))

	_, err := os.Stat(repoPath)
	assert.True(os.IsNotExist(err))
}

type TestSourceManagerUtils struct {
	*MemSystem
	*MemLogger
	*MemConfig
	*MemRunner
}

func newTestSourceManager() (*TestSourceManagerUtils, *RealSourceManager) {
	tu := &TestSourceManagerUtils{
		MemSystem: NewMemSystem(),
		MemLogger: &MemLogger{},
		MemConfig: NewMemConfig(),
		MemRunner: &MemRunner{},
	}
	return tu, &RealSourceManager{
		Logger:       tu.MemLogger,
		ConfigClient: tu.MemConfig,
		System:       tu.MemSystem,
		Runner:       tu.MemRunner,
	}
}

func TestSourceManagerAdd(t *testing.T) {
	assert := assert.New(t)

	tu, sm := newTestSourceManager()

	assert.Nil(sm.Add(false, "test", "test/repo"))
	assert.Equal(map[string]string{"source.test": "test/repo"}, tu.MemConfig.UserConfig)

	err := sm.Add(false, "test", "test/repo")
	assert.Contains(err.Error(), "already exists")
	assert.Equal(map[string]string{"source.test": "test/repo"}, tu.MemConfig.UserConfig)
}

func TestSourceManagerList(t *testing.T) {
	assert := assert.New(t)

	tu, sm := newTestSourceManager()

	assert.Nil(sm.Add(false, "test", "test/repo"))
	assert.Nil(sm.List())

	assert.Contains(tu.MemSystem.StdoutMessages, "test: test/repo (git source: https://github.com/test/repo.git)\n")
	assert.Contains(tu.MemSystem.StdoutMessages, "main: holen-app/manifests (git source: https://github.com/holen-app/manifests.git)\n")
}

func TestSourceManagerPaths(t *testing.T) {
	assert := assert.New(t)

	tempdir, _ := ioutil.TempDir("", "paths")
	defer os.RemoveAll(tempdir)

	tu, sm := newTestSourceManager()

	tu.MemSystem.Setenv("HOME", tempdir)

	dataPath, _ := tu.MemSystem.DataPath()
	testManifestsPath := filepath.Join(dataPath, "manifests", "test", "manifests")
	mainManifestsPath := filepath.Join(dataPath, "manifests", "main")
	tu.MemSystem.Files[testManifestsPath] = true

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
