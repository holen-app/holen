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

	paths := make([]string, 0)

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

	manifest := &Manifest{
		Logger:       dmf.Logger,
		ConfigGetter: dmf.ConfigGetter,
		Data:         md,
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
	Data ManifestData
}

func (m *Manifest) LoadStrategies(utility NameVer) ([]Strategy, error) {

	// default
	priority := "docker,binary"

	if configPriority, err := m.Get("strategy.priority"); err == nil && len(configPriority) > 0 {
		priority = configPriority
	}

	m.Debugf("Priority order: %s", priority)

	strategies := make([]Strategy, 0)

	var selectedStrategy string
	var foundStrategy map[interface{}]interface{}
	for _, try := range strings.Split(priority, ",") {
		try = strings.TrimSpace(try)
		if strategy, strategy_ok := m.Data.Strategies[try]; strategy_ok {
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
			orig_os_arch_map, os_arch_map_ok := final["os_arch_map"]

			os_arch_map := make(map[string]string)
			if os_arch_map_ok {
				for k, v := range orig_os_arch_map.(map[interface{}]interface{}) {
					os_arch_map[k.(string)] = v.(string)
				}
			}

			conf, err := NewDefaultConfigClient()
			if err != nil {
				return strategies, err
			}

			runner := &DefaultRunner{m.Logger}
			commonUtility := &StrategyCommon{
				System:       &DefaultSystem{},
				Logger:       m.Logger,
				ConfigGetter: conf,
				Downloader:   &DefaultDownloader{m.Logger, runner},
				Runner:       runner,
			}

			// handle strategy specific keys
			if selectedStrategy == "docker" {
				mount_pwd, mount_pwd_ok := final["mount_pwd"]
				docker_conn, docker_conn_ok := final["docker_conn"]
				interactive, interactive_ok := final["interactive"]
				pid_host, pid_host_ok := final["pid_host"]
				terminal, terminal_ok := final["terminal"]
				image, image_ok := final["image"]

				if !image_ok {
					return strategies, errors.New("At least 'image' needed for docker strategy to work")
				}

				if !terminal_ok {
					terminal = ""
				}

				strategies = append(strategies, DockerStrategy{
					StrategyCommon: commonUtility,
					Data: DockerData{
						Name:        m.Data.Name,
						Desc:        m.Data.Desc,
						Version:     final["version"].(string),
						Image:       image.(string),
						MountPwd:    mount_pwd_ok && mount_pwd.(bool),
						DockerConn:  docker_conn_ok && docker_conn.(bool),
						Interactive: !interactive_ok || interactive.(bool),
						PidHost:     !pid_host_ok || pid_host.(bool),
						Terminal:    terminal.(string),
						OSArchMap:   os_arch_map,
					},
				})
			} else if selectedStrategy == "binary" {
				base_url, base_url_ok := final["base_url"]

				if !base_url_ok {
					return strategies, errors.New("At least 'base_url' needed for binary strategy to work")
				}

				strategies = append(strategies, BinaryStrategy{
					StrategyCommon: commonUtility,
					Data: BinaryData{
						Name:      m.Data.Name,
						Desc:      m.Data.Desc,
						Version:   final["version"].(string),
						BaseUrl:   base_url.(string),
						OSArchMap: os_arch_map,
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
