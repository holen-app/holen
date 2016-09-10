package main

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type Strategy interface {
	Run() error
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

type ManifestData struct {
	Name       string `yaml:"name"`
	Strategies map[string]map[interface{}]interface{}
}

type Manifest struct {
	Logger
	ConfigGetter
	Data ManifestData
}

func (m *Manifest) LoadStrategy(utility NameVer) (Strategy, error) {

	// load this from config or by detecting environment
	defaultStrategy := "docker"

	var strat Strategy
	strategy, strategy_ok := m.Data.Strategies[defaultStrategy]

	if !strategy_ok {
		return strat, fmt.Errorf("Strategy %s not found.", defaultStrategy)
	}

	var selectedVersion map[interface{}]interface{}
	versions := strategy["versions"].([]interface{})

	if len(utility.Version) > 0 {
		found := false
		for _, verInfo := range versions {
			if verInfo.(map[interface{}]interface{})["version"] == utility.Version {
				selectedVersion = verInfo.(map[interface{}]interface{})
				found = true
			}
		}
		if !found {
			return strat, fmt.Errorf("Unable to find version %s", utility.Version)
		}
	} else {
		selectedVersion = versions[0].(map[interface{}]interface{})
	}

	delete(strategy, "versions")
	// fmt.Printf("%v\n", strategy)
	// fmt.Printf("%v\n", versions)
	// fmt.Printf("%v\n", selectedVersion)
	final := mergeMaps(strategy, selectedVersion)
	// fmt.Printf("%v\n", final)

	// handle common keys
	orig_arch_map, arch_map_ok := final["arch_map"]

	arch_map := make(map[string]string)
	if arch_map_ok {
		for k, v := range orig_arch_map.(map[interface{}]interface{}) {
			arch_map[k.(string)] = v.(string)
		}
	}

	conf, err := NewDefaultConfigClient()
	if err != nil {
		return strat, err
	}

	runner := &DefaultRunner{m.Logger}

	// handle strategy specific keys
	if defaultStrategy == "docker" {
		mount_pwd, mount_pwd_ok := final["mount_pwd"]
		docker_conn, docker_conn_ok := final["docker_conn"]
		interactive, interactive_ok := final["interactive"]
		image, image_ok := final["image"]

		if !image_ok {
			return strat, errors.New("At least 'image' needed for docker strategy to work")
		}

		strat = DockerStrategy{
			Logger:       m.Logger,
			ConfigGetter: conf,
			Downloader:   &DefaultDownloader{m.Logger, runner},
			Runner:       runner,
			Data: DockerData{
				Version:     final["version"].(string),
				Image:       image.(string),
				MountPwd:    mount_pwd_ok && mount_pwd.(bool),
				DockerConn:  docker_conn_ok && docker_conn.(bool),
				Interactive: !interactive_ok || interactive.(bool),
				ArchMap:     arch_map,
			},
		}
	} else if defaultStrategy == "binary" {
		base_url, base_url_ok := final["base_url"]

		if !base_url_ok {
			return strat, errors.New("At least 'base_url' needed for binary strategy to work")
		}

		strat = BinaryStrategy{
			Logger:       m.Logger,
			ConfigGetter: conf,
			Downloader:   &DefaultDownloader{m.Logger, runner},
			Runner:       runner,
			Data: BinaryData{
				Version: final["version"].(string),
				BaseUrl: base_url.(string),
				ArchMap: arch_map,
			},
		}
	}

	return strat, nil
}

func (ds DockerStrategy) Run() error {

	// TODO: template Image

	var err error

	err = ds.PullDockerImage(ds.Data.Image)
	if err != nil {
		return errors.Wrap(err, "can't pull image")
	}

	ds.RunCommand("docker", []string{"run", ds.Data.Image})
	if err != nil {
		return errors.Wrap(err, "can't run image")
	}

	return nil
}

func (bs BinaryStrategy) Run() error {
	return nil
}

func mergeMaps(m1, m2 map[interface{}]interface{}) map[interface{}]interface{} {
	for k, _ := range m1 {
		if vv, ok := m2[k]; ok {
			m1[k] = vv
			delete(m2, k)
		}
	}
	for k, v := range m2 {
		m1[k.(string)] = v
	}

	return m1
}

type NameVer struct {
	Name    string
	Version string
}

func ParseName(utility string) NameVer {
	parts := strings.Split(utility, "--")
	version := ""
	if len(parts) > 1 {
		version = parts[1]
	}

	return NameVer{parts[0], version}
}

func RunUtility(utility string) error {
	manifestFinder, err := NewManifestFinder()
	if err != nil {
		return err
	}

	nameVer := ParseName(utility)

	manifest, err := manifestFinder.Find(nameVer)
	if err != nil {
		return err
	}
	// fmt.Printf("manifest: %# v\n", pretty.Formatter(manifest))

	strategy, err := manifest.LoadStrategy(nameVer)
	if err != nil {
		return err
	}
	// fmt.Printf("%# v\n", pretty.Formatter(strategy))

	return strategy.Run()
}
