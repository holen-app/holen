package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

type MemConfig struct {
	SystemConfig map[string]string
	UserConfig   map[string]string
}

func (mc *MemConfig) Get(key string) (string, error) {
	if val, ok := mc.UserConfig[key]; ok {
		return val, nil
	}
	if val, ok := mc.SystemConfig[key]; ok {
		return val, nil
	}
	return "", nil
}

func (mc *MemConfig) Unset(system bool, key string) error {
	if system {
		delete(mc.SystemConfig, key)
	} else {
		delete(mc.UserConfig, key)
	}

	return nil
}

func (mc *MemConfig) Set(system bool, key, value string) error {
	if system {
		mc.SystemConfig[key] = value
	} else {
		mc.UserConfig[key] = value
	}

	return nil
}

func (mc *MemConfig) GetAll() (map[string]string, error) {
	combined := make(map[string]string)

	for k, v := range mc.SystemConfig {
		combined[k] = v
	}
	for k, v := range mc.UserConfig {
		combined[k] = v
	}

	return combined, nil
}

func NewMemConfig() *MemConfig {
	return &MemConfig{
		make(map[string]string),
		make(map[string]string),
	}
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
	History           []string
	FailCheckCmds     map[string]bool
	FailCmds          map[string]error
	CommandOutputCmds map[string]string
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

func (mr *MemRunner) ExecCommand(command string, args []string) error {
	return mr.RunCommand(command, args)
}

func (mr *MemRunner) CommandOutputToFile(command string, args []string, outputFile string) error {
	if mr.CommandOutputCmds == nil {
		mr.CommandOutputCmds = make(map[string]string)
	}

	fullCommand := strings.Join(append([]string{command}, args...), " ")
	mr.CommandOutputCmds[fullCommand] = outputFile

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
	Env            map[string]string
}

func NewMemSystem() *MemSystem {
	return &MemSystem{
		runtime.GOOS,
		runtime.GOARCH,
		1000,
		1000,
		make(map[string]bool),
		[]string{},
		[]string{},
		make(map[string][]string),
		map[string]string{"HOME": os.Getenv("HOME")},
	}
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

func (ms *MemSystem) Getenv(key string) string {
	val, ok := ms.Env[key]
	if ok {
		return val
	}

	return ""
}

// TODO: remove the duplication here and in system.go
func (ms *MemSystem) DataPath() (string, error) {
	var holenPath string
	if xdgDataHome := ms.Getenv("XDG_DATA_HOME"); len(xdgDataHome) > 0 {
		holenPath = filepath.Join(xdgDataHome, "holen")
	} else {
		var home string
		if home = ms.Getenv("HOME"); len(home) == 0 {
			return "", fmt.Errorf("$HOME environment variable not found")
		}
		holenPath = filepath.Join(home, ".local", "share", "holen")
	}
	os.MkdirAll(holenPath, 0755)

	return holenPath, nil
}

func (ms *MemSystem) Setenv(key, value string) {
	ms.Env[key] = value
}
