package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/kr/pretty"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type ManifestFinder interface {
	Find(NameVer) (*Manifest, error)
}

type DefaultManifestFinder struct {
	Logger
	ConfigGetter
	SelfPath string
}

func NewManifestFinder(selfPath string, conf ConfigGetter, logger Logger) (*DefaultManifestFinder, error) {
	return &DefaultManifestFinder{
		Logger:       logger,
		ConfigGetter: conf,
		SelfPath:     selfPath,
	}, nil
}

func (dmf DefaultManifestFinder) Find(utility NameVer) (*Manifest, error) {
	var paths []string

	holenPath := os.Getenv("HLN_PATH")
	if len(holenPath) > 0 {
		paths = append(paths, holenPath)
	}

	configHolenPath, err := dmf.Get("manifest.path")
	if err == nil && len(configHolenPath) > 0 {
		paths = append(paths, configHolenPath)
	}

	holenPathPost := os.Getenv("HLN_PATH_POST")
	if len(holenPathPost) > 0 {
		paths = append(paths, holenPathPost)
	}

	allPaths := strings.Join(paths, ":")

	if len(allPaths) == 0 {
		allPaths = path.Join(path.Dir(dmf.SelfPath), "manifests")
	}

	dmf.Debugf("all paths: %s", allPaths)

	var manifestPath string
	for _, p := range strings.Split(allPaths, ":") {

		tryPath := path.Join(p, fmt.Sprintf("%s.yaml", utility.Name))
		dmf.Debugf("trying: %s", tryPath)
		if _, err := os.Stat(tryPath); err == nil {
			dmf.Debugf("found manifest: %s", tryPath)
			manifestPath = tryPath
			break
		}
	}

	if len(manifestPath) == 0 {
		return nil, fmt.Errorf("unable to find manifest for %s", utility.Name)
	}

	return LoadManifest(utility, manifestPath, dmf.ConfigGetter, dmf.Logger)
}

func LoadManifest(utility NameVer, manifestPath string, conf ConfigGetter, logger Logger) (*Manifest, error) {
	md := ManifestData{}

	logger.Debugf("attemting to load: %s", manifestPath)
	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return nil, errors.Wrap(err, "problems with reading file")
	}

	err = yaml.Unmarshal([]byte(data), &md)
	if err != nil {
		return nil, errors.Wrap(err, "problems with unmarshal")
	}

	md.Name = utility.Name

	runner := &DefaultRunner{logger}
	manifest := &Manifest{
		Logger:       logger,
		ConfigGetter: conf,
		Data:         md,
		Runner:       runner,
		System:       &DefaultSystem{},
		Downloader:   &DefaultDownloader{logger, runner},
	}
	logger.Debugf("manifest found: %# v", pretty.Formatter(manifest))

	return manifest, nil
}

type ManifestData struct {
	Name       string
	Desc       string `yaml:"desc"`
	Strategies map[string]map[interface{}]interface{}
}

type Manifest struct {
	Logger
	ConfigGetter
	Runner
	System
	Downloader
	Data ManifestData
}

func (m *Manifest) LoadStrategies(utility NameVer) ([]Strategy, error) {

	// default
	priority := "docker,binary"

	if configPriority, err := m.Get("strategy.priority"); err == nil && len(configPriority) > 0 {
		priority = configPriority
	}

	m.Debugf("Priority order: %s", priority)

	var strategies []Strategy

	var selectedStrategy string
	var foundStrategy map[interface{}]interface{}
	for _, try := range strings.Split(priority, ",") {
		try = strings.TrimSpace(try)
		if strategy, strategyOk := m.Data.Strategies[try]; strategyOk {
			selectedStrategy = try
			foundStrategy = strategy

			var selectedVersion map[interface{}]interface{}
			versions := foundStrategy["versions"].([]interface{})

			if len(utility.Version) > 0 {
				found := false
				for _, verInfo := range versions {
					if verInfo.(map[interface{}]interface{})["version"] == utility.Version {
						selectedVersion = verInfo.(map[interface{}]interface{})
						found = true
					}
				}
				if !found {
					m.Debugf("strategy %s does not have version %s", try, utility.Version)
					continue
				}
			} else {
				selectedVersion = versions[0].(map[interface{}]interface{})
			}

			delete(foundStrategy, "versions")
			// fmt.Printf("%v\n", strategy)
			// fmt.Printf("%v\n", versions)
			// fmt.Printf("%v\n", selectedVersion)
			final := mergeMaps(foundStrategy, selectedVersion)
			// fmt.Printf("%v\n", final)

			// handle common keys
			var osArchData map[string]map[string]string
			if origOsArch, osArchMapOk := final["os_arch"]; osArchMapOk {
				osArchData = m.processOSArchMap(origOsArch)
			}

			commonUtility := m.generateCommon()

			// handle strategy specific keys
			strat, err := m.loadStrategy(selectedStrategy, final, commonUtility, osArchData)
			if err != nil {
				return strategies, errors.Wrap(err, "error loading strategy")
			}

			strategies = append(strategies, strat)
		}
	}

	m.Debugf("found strategies: %# v", pretty.Formatter(strategies))

	return strategies, nil
}

func (m *Manifest) generateCommon() *StrategyCommon {
	return &StrategyCommon{
		System:       m.System,
		Logger:       m.Logger,
		ConfigGetter: m.ConfigGetter,
		Downloader:   m.Downloader,
		Runner:       m.Runner,
	}
}

func (m *Manifest) processOSArchMap(in interface{}) map[string]map[string]string {
	osArchData := make(map[string]map[string]string)
	for k, v := range in.(map[interface{}]interface{}) {
		archMap := make(map[string]string)
		if v != nil {
			for k2, v2 := range v.(map[interface{}]interface{}) {
				archMap[k2.(string)] = v2.(string)
			}
		}
		osArchData[k.(string)] = archMap
	}

	return osArchData
}

func (m *Manifest) loadStrategy(strategyType string, strategyData map[interface{}]interface{}, common *StrategyCommon, osArchData map[string]map[string]string) (Strategy, error) {
	var dummy Strategy

	if strategyType == "docker" {
		mountPwd, mountPwdOk := strategyData["mount_pwd"]
		dockerConn, dockerConnOk := strategyData["docker_conn"]
		interactive, interactiveOk := strategyData["interactive"]
		pidHost, pidHostOk := strategyData["pid_host"]
		terminal, terminalOk := strategyData["terminal"]
		image, imageOk := strategyData["image"]
		mountPwdAs, mountPwdAsOk := strategyData["mount_pwd_as"]
		runAsUser, runAsUserOk := strategyData["run_as_user"]

		if !imageOk {
			return dummy, errors.New("At least 'image' needed for docker strategy to work")
		}

		if !terminalOk {
			terminal = ""
		}
		if !mountPwdAsOk {
			mountPwdAs = ""
		}

		return DockerStrategy{
			StrategyCommon: common,
			Data: DockerData{
				Name:        m.Data.Name,
				Desc:        m.Data.Desc,
				Version:     strategyData["version"].(string),
				Image:       image.(string),
				MountPwd:    mountPwdOk && mountPwd.(bool),
				DockerConn:  dockerConnOk && dockerConn.(bool),
				Interactive: !interactiveOk || interactive.(bool),
				PidHost:     pidHostOk && pidHost.(bool),
				Terminal:    terminal.(string),
				MountPwdAs:  mountPwdAs.(string),
				RunAsUser:   runAsUserOk && runAsUser.(bool),
				OSArchData:  osArchData,
			},
		}, nil
	} else if strategyType == "binary" {
		baseURL, baseURLOk := strategyData["base_url"]
		unpackPath, unpackPathOk := strategyData["unpack_path"]

		if !baseURLOk {
			return dummy, errors.New("At least 'base_url' needed for binary strategy to work")
		}
		if !unpackPathOk {
			unpackPath = ""
		}

		return BinaryStrategy{
			StrategyCommon: common,
			Data: BinaryData{
				Name:       m.Data.Name,
				Desc:       m.Data.Desc,
				Version:    strategyData["version"].(string),
				BaseURL:    baseURL.(string),
				UnpackPath: unpackPath.(string),
				OSArchData: osArchData,
			},
		}, nil
	}

	return dummy, errors.New("No strategy type")
}

func (m *Manifest) LoadAllStrategies() (map[string][]Strategy, error) {

	allStrategies := make(map[string][]Strategy)

	commonUtility := m.generateCommon()
	for strategyName, strategy := range m.Data.Strategies {

		versions := strategy["versions"].([]interface{})
		delete(strategy, "versions")

		for _, version := range versions {
			final := mergeMaps(strategy, version.(map[interface{}]interface{}))
			var osArchData map[string]map[string]string
			if origOsArch, osArchMapOk := final["os_arch"]; osArchMapOk {
				osArchData = m.processOSArchMap(origOsArch)
			}

			strat, err := m.loadStrategy(strategyName, final, commonUtility, osArchData)
			if err != nil {
				return nil, errors.Wrap(err, "error loading strategy")
			}
			allStrategies[strategyName] = append(allStrategies[strategyName], strat)
		}
	}
	return allStrategies, nil
}

func (m *Manifest) Run(utility NameVer, args []string) error {
	strategies, err := m.LoadStrategies(utility)
	if err != nil {
		return err
	}

	for _, strategy := range strategies {
		err = strategy.Run(args)
		if err == nil {
			break
		} else {
			// keep going if it's a reason to skip
			if _, ok := err.(*SkipError); !ok {
				return err
			}
		}
	}

	return nil
}
