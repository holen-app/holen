package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
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

	res, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to download %s", url))
	}

	out, err := os.Create(path)
	defer out.Close()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to create file %s", path))
	}

	_, err = io.Copy(out, res.Body)
	if err != nil {
		return errors.Wrap(err, "unable to save downloaded file")
	}
	res.Body.Close()

	return nil
}

func (dd DefaultDownloader) PullDockerImage(image string) error {
	return dd.RunCommand("docker", []string{"pull", image})
}

type Runner interface {
	RunCommand(string, []string) error
	CheckCommand(string, []string) bool
}

type DefaultRunner struct {
	Logger
}

func (dr DefaultRunner) CheckCommand(command string, args []string) bool {
	dr.Infof("Checking command %s with args %v", command, args)

	return exec.Command(command, args...).Run() == nil
}

func (dr DefaultRunner) RunCommand(command string, args []string) error {
	dr.Infof("Running command %s with args %v", command, args)

	// TODO: investigate using syscall.Exec

	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	// adapted from http://stackoverflow.com/questions/10385551/get-exit-code-go
	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			os.Exit(status.ExitStatus())
		} else {
			os.Exit(1)
		}
	} else {
		return errors.Wrap(err, "unable to run command")
	}
	// end adapted from

	return nil
}
