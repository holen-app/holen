package main

import "github.com/pkg/errors"

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
	Logger
	ConfigGetter
	Downloader
	Runner
	Data DockerData
}

type BinaryData struct {
	Version string            `yaml:"version"`
	BaseUrl string            `yaml:"base_url"`
	ArchMap map[string]string `yaml:"arch_map"`
}

type BinaryStrategy struct {
	Logger
	ConfigGetter
	Downloader
	Runner
	Data BinaryData
}

func (ds DockerStrategy) Run(args []string) error {

	// TODO: template Image

	var err error

	err = ds.PullDockerImage(ds.Data.Image)
	if err != nil {
		return errors.Wrap(err, "can't pull image")
	}

	ds.RunCommand("docker", append([]string{"run", ds.Data.Image}, args...))
	if err != nil {
		return errors.Wrap(err, "can't run image")
	}

	return nil
}

func (bs BinaryStrategy) Run(args []string) error {
	return nil
}
