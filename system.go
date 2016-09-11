package main

import (
	"runtime"

	"github.com/Sirupsen/logrus"
)

type System interface {
	OS() string
	Arch() string
}

type DefaultSystem struct{}

func (ds DefaultSystem) OS() string {
	return runtime.GOOS
}

func (ds DefaultSystem) Arch() string {
	return runtime.GOARCH
}

type Logger interface {
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
}

type LogrusLogger struct{}

func (ll LogrusLogger) Debugf(str string, args ...interface{}) {
	logrus.Debugf(str, args...)
}

func (ll LogrusLogger) Infof(str string, args ...interface{}) {
	logrus.Infof(str, args...)
}

func (ll LogrusLogger) Warnf(str string, args ...interface{}) {
	logrus.Warnf(str, args...)
}

type Downloader interface {
	DownloadFile(string, string) error
	PullDockerImage(string) error
}

type DefaultDownloader struct {
	Logger
	Runner
}

func (dd DefaultDownloader) DownloadFile(url, path string) error {
	dd.Infof("Downloading file from %s to %s", url, path)
	return nil
}

func (dd DefaultDownloader) PullDockerImage(image string) error {
	return dd.RunCommand("docker", []string{"pull", image})
}

type Runner interface {
	RunCommand(string, []string) error
}

type DefaultRunner struct {
	Logger
}

func (dr DefaultRunner) RunCommand(cmd string, args []string) error {
	dr.Infof("Running command %s with args %v", cmd, args)
	return nil
}
