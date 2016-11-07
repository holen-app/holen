package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/kr/pretty"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type ManifestFinder interface {
	Find(NameVer) (*Manifest, error)
	List() error
	Link(string, string, string) error
}

type DefaultManifestFinder struct {
	Logger
	ConfigGetter
	System
	SelfPath string
}

func NewManifestFinder(selfPath string, conf ConfigGetter, logger Logger, system System) (*DefaultManifestFinder, error) {
	return &DefaultManifestFinder{
		Logger:       logger,
		ConfigGetter: conf,
		System:       system,
		SelfPath:     selfPath,
	}, nil
}

func (dmf DefaultManifestFinder) Find(utility NameVer) (*Manifest, error) {

	var manifestPath string
	for _, p := range dmf.Paths() {

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

	return LoadManifest(utility, manifestPath, dmf.ConfigGetter, dmf.Logger, dmf.System)
}

func (dmf DefaultManifestFinder) Paths() []string {

	var paths []string

	holenPath := dmf.Getenv("HLN_PATH")
	if len(holenPath) > 0 {
		paths = append(paths, strings.Split(holenPath, ":")...)
	}

	configHolenPath, err := dmf.Get("manifest.path")
	if err == nil && len(configHolenPath) > 0 {
		paths = append(paths, strings.Split(configHolenPath, ":")...)
	}

	holenPathPost := dmf.Getenv("HLN_PATH_POST")
	if len(holenPathPost) > 0 {
		paths = append(paths, strings.Split(holenPathPost, ":")...)
	}

	paths = append(paths, path.Join(path.Dir(dmf.SelfPath), "manifests"))

	dmf.Debugf("all paths: %s", strings.Join(paths, ":"))

	return paths
}

func (dmf DefaultManifestFinder) List() error {
	utilityInfo := make(map[string]int)
	for _, p := range dmf.Paths() {
		dmf.eachManifestPath(p, func(name, fileName string) error {
			utilityInfo[name]++
			return nil
		})
	}

	utilityNames := make([]string, len(utilityInfo))

	// from http://stackoverflow.com/a/27848197
	i := 0
	for name := range utilityInfo {
		utilityNames[i] = name
		i++
	}
	// end from http://stackoverflow.com/a/27848197

	sort.Strings(utilityNames)

	for _, name := range utilityNames {
		dmf.Stdoutf("%v\n", name)
	}

	return nil
}

func (dmf DefaultManifestFinder) eachManifestPath(manifestPath string, callback func(name, fileName string) error) error {
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return nil
	}

	files, err := ioutil.ReadDir(manifestPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".yaml") {
			name := strings.TrimSuffix(file.Name(), ".yaml")
			err := callback(name, file.Name())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (dmf DefaultManifestFinder) Link(manifestPath, holenPath, binPath string) error {

	binPath, _ = filepath.Abs(binPath)
	manifestPath, _ = filepath.Abs(manifestPath)

	// TODO: should we create binPath if non-exist?

	if len(holenPath) == 0 {
		holenPath = dmf.SelfPath
	}

	dmf.Debugf("linking from utilities found in %s to %s in %s", manifestPath, holenPath, binPath)

	seenUtilities := make(map[string]bool)
	err := dmf.eachManifestPath(manifestPath, func(name, fileName string) error {
		_, ok := seenUtilities[name]
		if ok {
			dmf.Debugf(" seen %s already, skipping", name)
			return nil
		}
		seenUtilities[name] = true
		dmf.Debugf(" linking %s", name)

		fullBinPath := filepath.Join(binPath, name)
		dmf.Debugf("  full bin path %s", fullBinPath)

		relativePath, err := filepath.Rel(binPath, holenPath)
		if err != nil {
			return err
		}
		dmf.Debugf("  relative path %s", relativePath)

		// load up the manifest
		manifest, err := LoadManifest(ParseName(name), filepath.Join(manifestPath, fileName), dmf.ConfigGetter, dmf.Logger, dmf.System)
		if err != nil {
			return err
		}

		strategies, err := manifest.LoadAllStrategies(ParseName(name))
		if err != nil {
			return err
		}

		// link all found versions
		for _, strategy := range strategies {
			err = dmf.linkToHolen(relativePath, fmt.Sprintf("%s--%s", fullBinPath, strategy.Version()))
			if err != nil {
				return err
			}
		}

		// link utility without version number
		return dmf.linkToHolen(relativePath, fullBinPath)
	})

	if err != nil {
		return err
	}

	return nil
}

func (dmf DefaultManifestFinder) linkToHolen(relativePath, fullBinPath string) error {
	err := os.Symlink(relativePath, fullBinPath)
	if err != nil {
		if linkerr, ok := err.(*os.LinkError); ok {
			if errno, ok := linkerr.Err.(syscall.Errno); ok {
				if errno == syscall.EEXIST {
					// TODO: check if file is a symlink and if it points to
					// the correct place.  if not the correct holen, fix it.
					return nil
				}
			}
		}
		return err
	}

	return nil
}

// TODO: see if this can be made a method attached to ManifestFinder
func LoadManifest(utility NameVer, manifestPath string, conf ConfigGetter, logger Logger, system System) (*Manifest, error) {
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
		System:       system,
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

func (m *Manifest) StrategyOrder(utility NameVer) []string {
	// default
	allPriorities := []string{"docker", "binary", "cmdio"}

	priorities := []string{}

	xPriorityKeys := []string{
		fmt.Sprintf("strategy.%s.%s.xpriority", utility.Name, utility.Version),
		fmt.Sprintf("strategy.%s.xpriority", utility.Name),
		"strategy.xpriority",
	}

	for _, key := range xPriorityKeys {
		if value, err := m.Get(key); err == nil && len(value) > 0 {
			priorities = strings.Split(value, ",")
		}
	}

	if len(priorities) == 0 {

		priorityKeys := []string{
			fmt.Sprintf("strategy.%s.%s.priority", utility.Name, utility.Version),
			fmt.Sprintf("strategy.%s.priority", utility.Name),
			"strategy.priority",
		}

		for _, key := range priorityKeys {
			if value, err := m.Get(key); err == nil && len(value) > 0 {
				priorities = strings.Split(value, ",")
				skips := make(map[string]bool)
				for _, prio := range priorities {
					skips[prio] = true
				}

				for _, otherPriority := range allPriorities {
					if _, ok := skips[otherPriority]; !ok {
						priorities = append(priorities, otherPriority)
					}
				}
			}
		}
	}

	if len(priorities) == 0 {
		priorities = allPriorities
	}

	m.Debugf("Priority order: %s", priorities)

	return priorities
}

func (m *Manifest) LoadStrategies(utility NameVer) ([]Strategy, error) {

	strategyOrder := m.StrategyOrder(utility)
	var strategies []Strategy

	var selectedStrategy string
	var foundStrategy map[interface{}]interface{}
	for _, try := range strategyOrder {
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

func (m *Manifest) LoadAllStrategies(utility NameVer) ([]Strategy, error) {

	strategyOrder := m.StrategyOrder(utility)
	var strategies []Strategy

	commonUtility := m.generateCommon()
	// for strategyName, strategy := range m.Data.Strategies {
	for _, strategyName := range strategyOrder {
		strategyName = strings.TrimSpace(strategyName)

		strategy, ok := m.Data.Strategies[strategyName]
		if !ok {
			continue
		}

		versions := strategy["versions"].([]interface{})
		delete(strategy, "versions")

		for _, version := range versions {
			final := mergeMaps(copyMap(strategy), version.(map[interface{}]interface{}))
			var osArchData map[string]map[string]string
			if origOsArch, osArchMapOk := final["os_arch"]; osArchMapOk {
				osArchData = m.processOSArchMap(origOsArch)
			}

			strat, err := m.loadStrategy(strategyName, final, commonUtility, osArchData)
			if err != nil {
				return nil, errors.Wrap(err, "error loading strategy")
			}
			// allStrategies[strategyName] = append(allStrategies[strategyName], strat)
			strategies = append(strategies, strat)
		}
	}
	return strategies, nil
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
