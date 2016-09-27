package main

import "fmt"

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
	History []string
}

func (mr *MemRunner) CheckCommand(command string, args []string) bool {
	return true
}

func (mr *MemRunner) RunCommand(command string, args []string) error {
	mr.History = append(mr.History, fmt.Sprint(append([]string{command}, args...)))

	return nil
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

	return nil
}

func (md *MemDownloader) PullDockerImage(image string) error {
	md.DockerImages = append(md.DockerImages, image)

	return nil
}

type MemSystem struct {
	MOS   string
	MArch string
	MUID  int
	MGID  int
	Files map[string]bool
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
