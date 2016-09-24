package main

import "runtime"

type NullConfigGetter struct{}

func (ncg NullConfigGetter) Get(key string) (string, error) {
	return "", nil
}

type NullLogger struct{}

func (nl NullLogger) Debugf(str string, args ...interface{}) {
}

func (nl NullLogger) Infof(str string, args ...interface{}) {
}

func (nl NullLogger) Warnf(str string, args ...interface{}) {
}

type NullRunner struct{}

func (nr NullRunner) CheckCommand(command string, args []string) bool {
	return true
}

func (nr NullRunner) RunCommand(command string, args []string) error {
	return nil
}

type NullDownloader struct{}

func (nd NullDownloader) DownloadFile(url, path string) error {
	return nil
}

func (nd NullDownloader) PullDockerImage(image string) error {
	return nil
}

type NullSystem struct{}

func (ns NullSystem) OS() string {
	return runtime.GOOS
}

func (ns NullSystem) Arch() string {
	return runtime.GOARCH
}

func (ns NullSystem) UID() int {
	return 1000
}

func (ns NullSystem) GID() int {
	return 1000
}

func (ns NullSystem) FileExists(localPath string) bool {
	return true
}

func (ns NullSystem) MakeExecutable(localPath string) error {
	return nil
}
