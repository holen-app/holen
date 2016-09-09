package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type Strategy interface {
	Actions() error
}

type DockerStrategy struct {
	Version     string            `yaml:"version"`
	Image       string            `yaml:"image"`
	MountPwd    bool              `yaml:"mount_pwd"`
	DockerConn  bool              `yaml:"docker_conn"`
	Interactive bool              `yaml:"interactive"`
	ArchMap     map[string]string `yaml:"arch_map"`
}

type BinaryStrategy struct {
	Version string            `yaml:"version"`
	BaseUrl string            `yaml:"base_url"`
	ArchMap map[string]string `yaml:"arch_map"`
}

type Manifest struct {
	Name       string `yaml:"name"`
	Strategies map[string]map[interface{}]interface{}
}

func (rsc DockerStrategy) Actions() error {
	return nil
}

func (rsc BinaryStrategy) Actions() error {
	return nil
}

func RunUtility(s *System, name string) error {

	m := Manifest{}

	parts := strings.Split(name, "--")
	version := ""
	if len(parts) > 1 {
		version = parts[1]
	}

	s.Logger.Debugf("parts of the utility: %s", parts)
	file := fmt.Sprintf("manifests/%s.yaml", parts[0])

	s.Logger.Debugf("attemting to load: %s", file)
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return errors.Wrap(err, "problems with reading file")
	}

	err = yaml.Unmarshal([]byte(data), &m)
	if err != nil {
		return errors.Wrap(err, "problems with unmarshal")
	}

	// load this from config or by detecting environment
	defaultStrategy := "docker"

	strategy, err := loadStrategy(m, defaultStrategy, version)
	if err != nil {
		return errors.Wrap(err, "unable to load strategy")
	}

	fmt.Printf("%v\n", strategy)

	return nil
}

func loadStrategy(m Manifest, s, v string) (Strategy, error) {

	var strat Strategy
	strategy, strategy_ok := m.Strategies[s]

	if !strategy_ok {
		return strat, fmt.Errorf("Strategy %s not found.", s)
	}

	var selectedVersion map[interface{}]interface{}
	versions := strategy["versions"].([]interface{})

	if len(v) > 0 {
		found := false
		for _, verInfo := range versions {
			if verInfo.(map[interface{}]interface{})["version"] == v {
				selectedVersion = verInfo.(map[interface{}]interface{})
				found = true
			}
		}
		if !found {
			return strat, fmt.Errorf("Unable to find version %s", v)
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

	// handle strategy specific keys
	if s == "docker" {
		mount_pwd, mount_pwd_ok := final["mount_pwd"]
		docker_conn, docker_conn_ok := final["docker_conn"]
		interactive, interactive_ok := final["interactive"]
		image, image_ok := final["image"]

		if !image_ok {
			return strat, errors.New("At least 'image' needed for docker strategy to work")
		}

		strat = DockerStrategy{
			Version:     final["version"].(string),
			Image:       image.(string),
			MountPwd:    mount_pwd_ok && mount_pwd.(bool),
			DockerConn:  docker_conn_ok && docker_conn.(bool),
			Interactive: !interactive_ok || interactive.(bool),
			ArchMap:     arch_map,
		}
	} else if s == "binary" {
		base_url, base_url_ok := final["base_url"]

		if !base_url_ok {
			return strat, errors.New("At least 'base_url' needed for binary strategy to work")
		}

		strat = BinaryStrategy{
			Version: final["version"].(string),
			BaseUrl: base_url.(string),
			ArchMap: arch_map,
		}
	}

	return strat, nil
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
