package main

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

type DockerTestUtils struct {
	*MemSystem
	*MemLogger
	*MemConfig
	*MemDownloader
	*MemRunner
}

func newDockerStrategy() (*DockerTestUtils, *DockerStrategy) {
	tu := &DockerTestUtils{
		MemSystem:     &MemSystem{runtime.GOOS, runtime.GOARCH, 1000, 1000, make(map[string]bool)},
		MemLogger:     &MemLogger{},
		MemConfig:     &MemConfig{},
		MemDownloader: &MemDownloader{},
		MemRunner:     &MemRunner{},
	}
	return tu, &DockerStrategy{
		StrategyCommon: &StrategyCommon{
			System:       tu.MemSystem,
			Logger:       tu.MemLogger,
			ConfigGetter: tu.MemConfig,
			Downloader:   tu.MemDownloader,
			Runner:       tu.MemRunner,
		},
		Data: DockerData{
			Name:      "testdocker",
			Desc:      "Test Docker Program",
			Version:   "1.9",
			Image:     "testdocker:{{.Version}}",
			OSArchMap: make(map[string]string),
		},
	}
}

func TestDockerSimple(t *testing.T) {

	assert := assert.New(t)

	tu, td := newDockerStrategy()
	td.Run([]string{"first", "second"})

	assert.Equal(tu.MemRunner.History[0], "docker run --rm testdocker:1.9 first second")
}

func TestDockerAllOptions(t *testing.T) {
	assert := assert.New(t)

	tu, td := newDockerStrategy()
	td.Data.Interactive = true
	td.Data.DockerConn = true
	td.Data.PidHost = true
	td.Data.MountPwdAs = "/test"
	td.Data.MountPwd = true
	td.Data.RunAsUser = true
	td.Data.Terminal = "always"
	assert.Nil(td.Run([]string{"first", "second"}))

	wd, _ := os.Getwd()
	assert.Equal(tu.MemRunner.History[0], fmt.Sprintf("docker run -i -v /var/run/docker.sock:/var/run/docker.sock --pid host --volume %s:/test --volume %s:%s -u 1000:1000 -t --rm testdocker:1.9 first second", wd, wd, wd))
}

func TestDockerNotInstalled(t *testing.T) {
	assert := assert.New(t)

	tu, td := newDockerStrategy()
	tu.MemRunner.FailCheck("docker version")
	err := td.Run([]string{"first", "second"})
	assert.NotNil(err)
	assert.Contains(err.Error(), "docker not available")
}

func TestDockerBadImageTemplate(t *testing.T) {
	assert := assert.New(t)

	_, td := newDockerStrategy()
	td.Data.Image = "{{.Foo"

	err := td.Run([]string{"first", "second"})
	assert.NotNil(err)
	assert.Contains(err.Error(), "unclosed action")
}

func TestDockerCommandFailed(t *testing.T) {
	assert := assert.New(t)

	tu, td := newDockerStrategy()
	tu.MemRunner.FailCommand("docker run --rm testdocker:1.9 first second", fmt.Errorf("bad output"))
	err := td.Run([]string{"first", "second"})
	assert.NotNil(err)
	// assert.Contains(err.Error(), "bad output")
}
