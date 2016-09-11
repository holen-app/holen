package main

import "github.com/pkg/errors"

type StrategyCommon struct {
	System
	Logger
	ConfigGetter
	Downloader
	Runner
}

func (sc *StrategyCommon) Templater(version string, archMap map[string]string, system System) Templater {
	// TODO: extract MappedArch
	return Templater{
		Version:    version,
		OS:         system.OS(),
		Arch:       system.Arch(),
		MappedArch: "test",
	}
}

type Strategy interface {
	Run([]string) error
}

type DockerData struct {
	Version     string            `yaml:"version"`
	Image       string            `yaml:"image"`
	MountPwd    bool              `yaml:"mount_pwd"`
	DockerConn  bool              `yaml:"docker_conn"`
	Interactive bool              `yaml:"interactive"`
	ArchMap     map[string]string `yaml:"arch_map"`
}

type DockerStrategy struct {
	*StrategyCommon
	Data DockerData
}

type BinaryData struct {
	Version string            `yaml:"version"`
	BaseUrl string            `yaml:"base_url"`
	ArchMap map[string]string `yaml:"arch_map"`
}

type BinaryStrategy struct {
	*StrategyCommon
	Data BinaryData
}

func (ds DockerStrategy) Run(args []string) error {

	// TODO: template Image

	var err error

	temp := ds.Templater(ds.Data.Version, ds.Data.ArchMap, ds.System)
	image, err := temp.Template(ds.Data.Image)
	if err != nil {
		return errors.Wrap(err, "unable to template image name")
	}

	err = ds.PullDockerImage(image)
	if err != nil {
		return errors.Wrap(err, "can't pull image")
	}

	ds.RunCommand("docker", append([]string{"run", image}, args...))
	if err != nil {
		return errors.Wrap(err, "can't run image")
	}

	return nil
}

func (bs BinaryStrategy) Run(args []string) error {
	return nil
}
