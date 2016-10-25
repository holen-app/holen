package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type MemConfig struct {
	Config map[string]string
}

func (mc *MemConfig) Get(key string) (string, error) {
	if val, ok := mc.Config[key]; ok {
		return val, nil
	}
	return "", nil
}

func (mc *MemConfig) Set(key, value string) {
	if mc.Config == nil {
		mc.Config = make(map[string]string)
	}

	mc.Config[key] = value
}

type MemLogger struct {
	Debugs []string
	Infos  []string
	Warns  []string
}

func (ml *MemLogger) Debugf(str string, args ...interface{}) {
	ml.Debugs = append(ml.Debugs, fmt.Sprintf(str, args...))
}

func (ml *MemLogger) Infof(str string, args ...interface{}) {
	ml.Infos = append(ml.Infos, fmt.Sprintf(str, args...))
}

func (ml *MemLogger) Warnf(str string, args ...interface{}) {
	ml.Warns = append(ml.Warns, fmt.Sprintf(str, args...))
}

type MemRunner struct {
	History       []string
	FailCheckCmds map[string]bool
	FailCmds      map[string]error
}

func (mr *MemRunner) CheckCommand(command string, args []string) bool {
	fail, ok := mr.FailCheckCmds[strings.Join(append([]string{command}, args...), " ")]

	if !ok {
		return true
	} else {
		return !fail
	}
}

func (mr *MemRunner) RunCommand(command string, args []string) error {
	fullCommand := strings.Join(append([]string{command}, args...), " ")
	mr.History = append(mr.History, fullCommand)

	e, ok := mr.FailCmds[fullCommand]

	if !ok {
		return nil
	} else {
		return e
	}
}

func (mr *MemRunner) FailCheck(fullCommand string) {
	if mr.FailCheckCmds == nil {
		mr.FailCheckCmds = make(map[string]bool)
	}

	mr.FailCheckCmds[fullCommand] = true
}

func (mr *MemRunner) FailCommand(fullCommand string, err error) {
	if mr.FailCmds == nil {
		mr.FailCmds = make(map[string]error)
	}

	mr.FailCmds[fullCommand] = err
}

type MemDownloader struct {
	Files        map[string]string
	DockerImages []string
}

func (md *MemDownloader) DownloadFile(url, path string) error {
	if md.Files == nil {
		md.Files = make(map[string]string)
	}

	md.Files[url] = path
	os.Create(path)

	return nil
}

func (md *MemDownloader) PullDockerImage(image string) error {
	md.DockerImages = append(md.DockerImages, image)

	return nil
}

type MemSystem struct {
	MOS            string
	MArch          string
	MUID           int
	MGID           int
	Files          map[string]bool
	StderrMessages []string
	StdoutMessages []string
	ArchiveFiles   map[string][]string
}

func (ms MemSystem) OS() string {
	return ms.MOS
}

func (ms MemSystem) Arch() string {
	return ms.MArch
}

func (ms MemSystem) UID() int {
	return ms.MUID
}

func (ms MemSystem) GID() int {
	return ms.MGID
}

func (ms MemSystem) FileExists(localPath string) bool {
	present, ok := ms.Files[localPath]
	return ok && present
}

func (ms MemSystem) MakeExecutable(localPath string) error {
	return nil
}

func (ms *MemSystem) Stderrf(message string, args ...interface{}) {
	ms.StderrMessages = append(ms.StderrMessages, fmt.Sprintf(message, args...))
}

func (ms *MemSystem) Stdoutf(message string, args ...interface{}) {
	ms.StdoutMessages = append(ms.StdoutMessages, fmt.Sprintf(message, args...))
}

func (ms *MemSystem) UnpackArchive(archive, destPath string) error {
	os.MkdirAll(destPath, 0755)

	baseName := path.Base(archive)

	// this only handles files in the "root" of the archive, no subpaths
	if paths, ok := ms.ArchiveFiles[baseName]; ok {
		for _, path := range paths {
			os.Create(filepath.Join(destPath, path))
		}
	}
	return nil
}
