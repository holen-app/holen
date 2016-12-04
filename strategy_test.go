package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
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
		MemSystem:     NewMemSystem(),
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
	td.Data.PwdWorkdir = true
	td.Data.Terminal = "always"
	assert.Nil(td.Run([]string{"first", "second"}))

	wd, _ := os.Getwd()
	assert.Equal(tu.MemRunner.History[0], fmt.Sprintf("docker run -i -v /var/run/docker.sock:/var/run/docker.sock --pid host --volume %s:/test --workdir /test --volume %s:%s --workdir %s -u 1000:1000 -t --rm testdocker:1.9 first second", wd, wd, wd, wd))
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

func TestDockerBootstrapScript(t *testing.T) {
	assert := assert.New(t)

	var tests = []struct {
		modify func(*TestUtils, *DockerStrategy)
		err    error
	}{
		{
			func(tu *TestUtils, td *DockerStrategy) {},
			nil,
		},
		{
			func(tu *TestUtils, td *DockerStrategy) {
				tu.MemRunner.FailCommand("docker run --rm -i testdocker:1.9 cat /bootstrap", fmt.Errorf("fail"))
			},
			fmt.Errorf("fail"),
		},
	}

	for _, test := range tests {
		tu, td := newDockerStrategy()
		test.modify(tu, td)
		td.Data.BootstrapScript = "/bootstrap"

		result := td.Run([]string{"first", "second"})

		if test.err != nil {
			assert.NotNil(result)
		} else {
			assert.Nil(result)
			assert.Contains(tu.MemRunner.CommandOutputCmds, "docker run --rm -i testdocker:1.9 cat /bootstrap")

			// check env vars for actual run
			var envs [][]string
			for _, v := range tu.MemRunner.HistoryEnv {
				envs = append(envs, v)
			}
			assert.Equal([]string{"DOCKER_IMAGE=testdocker:1.9"}, envs[0])

			assert.Contains(tu.MemRunner.History[0], "/execute first second")
		}
	}
}

func TestDockerInspect(t *testing.T) {
	assert := assert.New(t)

	tu, td := newDockerStrategy()
	td.Inspect()
	completeOutput := strings.Join(tu.MemSystem.StdoutMessages, "")

	assert.Contains(completeOutput, "final image: testdocker:1.9")
	assert.Contains(completeOutput, "final command: docker run --rm testdocker:1.9 [args]")
}

func newBinaryStrategy() (*TestUtils, *BinaryStrategy) {
	tu := &TestUtils{
		MemSystem:     NewMemSystem(),
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
			Name:    "testbinary",
			Desc:    "Test Binary Program",
			Version: "2.1",
			BaseURL: "https://github.com/testbinary/bin/releases/download/bin-{{.Version}}/jq-{{.OSArch}}",
			OSArchData: map[string]map[string]string{
				"linux_amd64": map[string]string{
					"md5sum": "d41d8cd98f00b204e9800998ecf8427e",
				},
			},
		},
	}
}

func TestBinarySimple(t *testing.T) {

	assert := assert.New(t)

	tu, tb := newBinaryStrategy()
	err := tb.Run([]string{"first", "second"})
	assert.Nil(err)

	binPath := path.Join(tu.MemSystem.Getenv("HOME"), ".local/share/holen/bin/testbinary--2.1")
	remoteUrl := "https://github.com/testbinary/bin/releases/download/bin-2.1/jq-linux_amd64"

	// check download
	assert.Contains(tu.MemDownloader.Files, remoteUrl)
	assert.Contains(tu.MemDownloader.Files[remoteUrl], path.Join(tu.MemSystem.Getenv("HOME"), ".local/share/holen/tmp"))
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

	binPath := path.Join(tu.MemSystem.Getenv("HOME"), ".local/share/holen/bin/testbinary--2.1")
	remoteUrl := "https://github.com/testbinary/bin/releases/download/bin-2.1/testbinary-linux_amd64.zip"

	// check download
	assert.Contains(tu.MemDownloader.Files, remoteUrl)
	assert.Contains(tu.MemDownloader.Files[remoteUrl], path.Join(tu.MemSystem.Getenv("HOME"), ".local/share/holen/tmp"))
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
	}{
		{
			nil,
			nil,
			path.Join(os.Getenv("HOME"), ".local/share/holen/bin"),
		},
		{
			func(tu *TestUtils) {
				tu.MemSystem.Setenv("XDG_DATA_HOME", "/tmp")
			},
			nil,
			"/tmp/holen/bin",
		},
		{
			func(tu *TestUtils) {
				tu.MemSystem.Setenv("HOME", "")
			},
			fmt.Errorf("$HOME not found"),
			"",
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

func TestBinaryInspect(t *testing.T) {
	assert := assert.New(t)

	tu, tb := newBinaryStrategy()
	tb.Inspect()
	completeOutput := strings.Join(tu.MemSystem.StdoutMessages, "")

	assert.Contains(completeOutput, "final url: https://github.com/testbinary/bin/releases/download/bin-2.1/jq-linux_amd64")
	assert.Contains(completeOutput, "checksum with md5: d41d8cd98f00b204e9800998ecf8427e")
}

func newCmdioStrategy() (*TestUtils, *CmdioStrategy) {
	tu := &TestUtils{
		MemSystem:     NewMemSystem(),
		MemLogger:     &MemLogger{},
		MemConfig:     &MemConfig{},
		MemDownloader: &MemDownloader{},
		MemRunner:     &MemRunner{},
	}
	return tu, &CmdioStrategy{
		StrategyCommon: &StrategyCommon{
			System:       tu.MemSystem,
			Logger:       tu.MemLogger,
			ConfigGetter: tu.MemConfig,
			Downloader:   tu.MemDownloader,
			Runner:       tu.MemRunner,
		},
		Data: CmdioData{
			Name:       "testbinary",
			Desc:       "Test Binary Program",
			Version:    "2.1",
			Command:    "testbinary--{{.Version}}",
			OSArchData: map[string]map[string]string{},
		},
	}
}

func TestCmdioSimple(t *testing.T) {

	assert := assert.New(t)

	tu, tc := newCmdioStrategy()
	err := tc.Run([]string{"first", "second"})
	assert.Nil(err)

	assert.Equal(tu.MemRunner.History[0], "ssh alpha.cmd.io testbinary--2.1 first second")
}

func TestCmdioInspect(t *testing.T) {
	assert := assert.New(t)

	tu, tc := newCmdioStrategy()
	tc.Inspect()
	completeOutput := strings.Join(tu.MemSystem.StdoutMessages, "")

	assert.Contains(completeOutput, "final command: testbinary--2.1")
}

func TestCmdioCommandFailed(t *testing.T) {
	assert := assert.New(t)

	tu, tc := newCmdioStrategy()
	tu.MemRunner.FailCommand("ssh alpha.cmd.io testbinary--2.1 first second", fmt.Errorf("bad output"))
	err := tc.Run([]string{"first", "second"})
	assert.NotNil(err)
	assert.Contains(err.Error(), "bad output")
}
