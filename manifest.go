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

func NewManifestFinder(selfPath string) (*DefaultManifestFinder, error) {
	conf, err := NewDefaultConfigClient()
	if err != nil {
		return nil, err
	}

	logger := &LogrusLogger{}
	return &DefaultManifestFinder{
		Logger:       logger,
		ConfigGetter: conf,
		SelfPath:     selfPath,
	}, nil
}

func (dmf DefaultManifestFinder) Find(utility NameVer) (*Manifest, error) {
	md := ManifestData{}

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

	dmf.Infof("attemting to load: %s", manifestPath)
	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return nil, errors.Wrap(err, "problems with reading file")
	}

	err = yaml.Unmarshal([]byte(data), &md)
	if err != nil {
		return nil, errors.Wrap(err, "problems with unmarshal")
	}

	md.Name = utility.Name

	runner := &DefaultRunner{dmf.Logger}
	manifest := &Manifest{
		Logger:       dmf.Logger,
		ConfigGetter: dmf.ConfigGetter,
		Data:         md,
		Runner:       runner,
		System:       &DefaultSystem{},
		Downloader:   &DefaultDownloader{dmf.Logger, runner},
	}
	dmf.Debugf("manifest found: %# v", pretty.Formatter(manifest))

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
			origOsArchMap, osArchMapOk := final["os_arch_map"]

			osArchMap := make(map[string]string)
			if osArchMapOk {
				for k, v := range origOsArchMap.(map[interface{}]interface{}) {
					osArchMap[k.(string)] = v.(string)
				}
			}

			commonUtility := &StrategyCommon{
				System:       m.System,
				Logger:       m.Logger,
				ConfigGetter: m.ConfigGetter,
				Downloader:   m.Downloader,
				Runner:       m.Runner,
			}

			// handle strategy specific keys
			if selectedStrategy == "docker" {
				mountPwd, mountPwdOk := final["mount_pwd"]
				dockerConn, dockerConnOk := final["docker_conn"]
				interactive, interactiveOk := final["interactive"]
				pidHost, pidHostOk := final["pid_host"]
				terminal, terminalOk := final["terminal"]
				image, imageOk := final["image"]
				mountPwdAs, mountPwdAsOk := final["mount_pwd_as"]
				runAsUser, runAsUserOk := final["run_as_user"]

				if !imageOk {
					return strategies, errors.New("At least 'image' needed for docker strategy to work")
				}

				if !terminalOk {
					terminal = ""
				}
				if !mountPwdAsOk {
					mountPwdAs = ""
				}

				strategies = append(strategies, DockerStrategy{
					StrategyCommon: commonUtility,
					Data: DockerData{
						Name:        m.Data.Name,
						Desc:        m.Data.Desc,
						Version:     final["version"].(string),
						Image:       image.(string),
						MountPwd:    mountPwdOk && mountPwd.(bool),
						DockerConn:  dockerConnOk && dockerConn.(bool),
						Interactive: !interactiveOk || interactive.(bool),
						PidHost:     pidHostOk && pidHost.(bool),
						Terminal:    terminal.(string),
						MountPwdAs:  mountPwdAs.(string),
						RunAsUser:   runAsUserOk && runAsUser.(bool),
						OSArchMap:   osArchMap,
					},
				})
			} else if selectedStrategy == "binary" {
				baseURL, baseURLOk := final["base_url"]

				if !baseURLOk {
					return strategies, errors.New("At least 'base_url' needed for binary strategy to work")
				}

				strategies = append(strategies, BinaryStrategy{
					StrategyCommon: commonUtility,
					Data: BinaryData{
						Name:      m.Data.Name,
						Desc:      m.Data.Desc,
						Version:   final["version"].(string),
						BaseURL:   baseURL.(string),
						OSArchMap: osArchMap,
					},
				})
			}

		}
	}

	m.Debugf("found strategies: %# v", pretty.Formatter(strategies))

	return strategies, nil
}

func (m *Manifest) Run(utility NameVer, args []string) error {
	strategies, err := m.LoadStrategies(utility)
	if err != nil {
		return err
	}

	for _, strategy := range strategies {
		err = strategy.Run(args)
		if err != nil {
			// keep going if it's a reason to skip
			if _, ok := err.(*SkipError); !ok {
				return err
			}
		}
	}

	return nil
}
