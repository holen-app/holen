package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestUtils struct {
	*MemSystem
	*MemLogger
	*MemConfig
	*MemDownloader
	*MemRunner
}

func newDockerStrategy() (*TestUtils, *DockerStrategy) {
	tu := &TestUtils{
		MemSystem:     &MemSystem{runtime.GOOS, runtime.GOARCH, 1000, 1000, make(map[string]bool), []string{}, []string{}, make(map[string][]string)},
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
			Name:       "testdocker",
			Desc:       "Test Docker Program",
			Version:    "1.9",
			Image:      "testdocker:{{.Version}}",
			OSArchData: make(map[string]map[string]string),
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

func newBinaryStrategy() (*TestUtils, *BinaryStrategy) {
	tu := &TestUtils{
		MemSystem:     &MemSystem{runtime.GOOS, runtime.GOARCH, 1000, 1000, make(map[string]bool), []string{}, []string{}, make(map[string][]string)},
		MemLogger:     &MemLogger{},
		MemConfig:     &MemConfig{},
		MemDownloader: &MemDownloader{},
		MemRunner:     &MemRunner{},
	}
	return tu, &BinaryStrategy{
		StrategyCommon: &StrategyCommon{
			System:       tu.MemSystem,
			Logger:       tu.MemLogger,
			ConfigGetter: tu.MemConfig,
			Downloader:   tu.MemDownloader,
			Runner:       tu.MemRunner,
		},
		Data: BinaryData{
			Name:       "testbinary",
			Desc:       "Test Binary Program",
			Version:    "2.1",
			BaseURL:    "https://github.com/testbinary/bin/releases/download/bin-{{.Version}}/jq-{{.OSArch}}",
			OSArchData: make(map[string]map[string]string),
		},
	}
}

func TestBinarySimple(t *testing.T) {

	assert := assert.New(t)

	tu, tb := newBinaryStrategy()
	tb.Run([]string{"first", "second"})

	binPath := path.Join(os.Getenv("HOME"), ".local/share/holen/bin/testbinary--2.1")
	remoteUrl := "https://github.com/testbinary/bin/releases/download/bin-2.1/jq-linux_amd64"

	// check download
	assert.Contains(tu.MemDownloader.Files, remoteUrl)
	assert.Contains(tu.MemDownloader.Files[remoteUrl], path.Join(os.Getenv("HOME"), ".local/share/holen/tmp"))
	assert.Contains(tu.MemDownloader.Files[remoteUrl], "testbinary--2.1")

	assert.Contains(tu.MemSystem.StderrMessages[0], "Downloading")
	assert.Contains(tu.MemSystem.StderrMessages[0], remoteUrl)

	assert.Equal(tu.MemRunner.History[0], fmt.Sprintf("%s first second", binPath))
}

func TestBinaryBadImageTemplate(t *testing.T) {
	assert := assert.New(t)

	_, tb := newBinaryStrategy()
	tb.Data.BaseURL = "https://{{.Foo"

	err := tb.Run([]string{"first", "second"})
	assert.NotNil(err)
	assert.Contains(err.Error(), "unclosed action")
}

func TestBinaryArchive(t *testing.T) {

	assert := assert.New(t)

	tu, tb := newBinaryStrategy()
	tb.Data.UnpackPath = "testbinary"
	tb.Data.BaseURL = "https://github.com/testbinary/bin/releases/download/bin-{{.Version}}/testbinary-{{.OSArch}}.zip"
	tu.MemSystem.ArchiveFiles["testbinary-linux_amd64.zip"] = []string{"testbinary"}

	err := tb.Run([]string{"first", "second"})
	assert.Nil(err)

	binPath := path.Join(os.Getenv("HOME"), ".local/share/holen/bin/testbinary--2.1")
	remoteUrl := "https://github.com/testbinary/bin/releases/download/bin-2.1/testbinary-linux_amd64.zip"

	// check download
	assert.Contains(tu.MemDownloader.Files, remoteUrl)
	assert.Contains(tu.MemDownloader.Files[remoteUrl], path.Join(os.Getenv("HOME"), ".local/share/holen/tmp"))
	assert.Contains(tu.MemDownloader.Files[remoteUrl], "testbinary-linux_amd64.zip")

	assert.Contains(tu.MemSystem.StderrMessages[0], "Downloading")
	assert.Contains(tu.MemSystem.StderrMessages[0], remoteUrl)

	assert.Equal(tu.MemRunner.History[0], fmt.Sprintf("%s first second", binPath))
}

func TestBinaryDownloadPath(t *testing.T) {
	assert := assert.New(t)

	var downloadPathTests = []struct {
		adjustment func(*TestUtils)
		err        error
		result     string
		cleanup    func(*TestUtils)
	}{
		{
			nil,
			nil,
			path.Join(os.Getenv("HOME"), ".local/share/holen/bin"),
			nil,
		},
		{
			func(tu *TestUtils) {
				os.Setenv("XDG_DATA_HOME", "/tmp")
			},
			nil,
			"/tmp/holen/bin",
			func(tu *TestUtils) {
				os.Setenv("XDG_DATA_HOME", "")
			},
		},
		{
			func(tu *TestUtils) {
				os.Setenv("HOME", "")
			},
			fmt.Errorf("$HOME not found"),
			"",
			func(tu *TestUtils) { return },
		},
	}

	for _, test := range downloadPathTests {

		tu, tb := newBinaryStrategy()
		if test.adjustment != nil {
			test.adjustment(tu)
		}

		result, err := tb.DownloadPath()
		if test.err == nil {
			assert.Nil(err)
		} else {
			assert.NotNil(err)
		}
		assert.Equal(result, test.result)
		if test.cleanup != nil {
			test.cleanup(tu)
		}
	}
}

func TestBinaryChecksumBinary(t *testing.T) {
	assert := assert.New(t)

	tempdir, _ := ioutil.TempDir("", "hash")
	defer os.RemoveAll(tempdir)
	filePath := path.Join(tempdir, "testfile")
	assert.Nil(ioutil.WriteFile(filePath, []byte("test contents\n"), 0755))

	var checksumTests = []struct {
		hashdata map[string]string
		result   error
	}{
		{
			map[string]string{"md5sum": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},
			HashMismatch{algo: "md5", checksum: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", hash: "1b3c032e3e4eaad23401e1568879f150"},
		},
		{
			map[string]string{"md5sum": "1b3c032e3e4eaad23401e1568879f150"},
			nil,
		},
		{
			map[string]string{"sha1sum": "40b44f15b4b6690a90792137a03d57c4d2918271"},
			nil,
		},
		{
			map[string]string{"sha256sum": "15721d5068de16cf4eba8d0fe6a563bb177333405323b479dcf5986da440c081"},
			nil,
		},
		{
			map[string]string{
				"md5sum":    "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
				"sha1sum":   "yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy",
				"sha256sum": "15721d5068de16cf4eba8d0fe6a563bb177333405323b479dcf5986da440c081",
			},
			nil,
		},
	}

	for _, test := range checksumTests {

		_, tb := newBinaryStrategy()

		tb.Data.OSArchData[fmt.Sprintf("%s_%s", tb.OS(), tb.Arch())] = test.hashdata
		result := tb.ChecksumBinary(filePath)

		assert.Equal(result, test.result)
	}
}
