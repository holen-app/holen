package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kr/pretty"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type ManifestFinder interface {
	Find(NameVer) (*Manifest, error)
	List(string, bool) error
	LinkAllUtilities(string, string, string) error
	LinkSingleUtility(string, string, string, string) error
	DefaultLinkBinPath() string
}

type DefaultManifestFinder struct {
	Logger
	ConfigGetter
	System
	SourcePather
	SelfPath string
}

func NewManifestFinder() (*DefaultManifestFinder, error) {
	selfPath, err := findSelfPath()
	if err != nil {
		return nil, err
	}

	system := &DefaultSystem{}
	conf, err := NewDefaultConfigClient(system)
	if err != nil {
		return nil, err
	}

	sourceManager, err := NewDefaultSourceManager()
	if err != nil {
		return nil, err
	}

	return &DefaultManifestFinder{
		Logger:       &LogrusLogger{},
		ConfigGetter: conf,
		System:       system,
		SourcePather: sourceManager,
		SelfPath:     selfPath,
	}, nil
}

func (dmf DefaultManifestFinder) Find(utility NameVer) (*Manifest, error) {

	var manifestPath string
	sourcePaths, err := dmf.Paths("")
	if err != nil {
		return nil, err
	}

	for _, p := range sourcePaths {

		tryPath := filepath.Join(p, fmt.Sprintf("%s.yaml", utility.Name))
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

type listInfo struct {
	name, desc string
	count      int
}

func (dmf DefaultManifestFinder) List(source string, desc bool) error {
	utilityInfo := make(map[string]*listInfo)

	sourcePaths, err := dmf.Paths(source)
	if err != nil {
		return err
	}

	for _, p := range sourcePaths {
		dmf.eachManifestPath(p, func(name, fileName string) error {
			if info, ok := utilityInfo[name]; !ok {
				info := &listInfo{name, "", 1}
				man, err := LoadManifest(ParseName(name), filepath.Join(p, fileName), dmf.ConfigGetter, dmf.Logger, dmf.System)
				if err == nil {
					info.desc = man.Data.Desc
				}
				utilityInfo[name] = info
			} else {
				info.count++
			}

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
		if desc {
			dmf.Stdoutf("%s: %s\n", name, utilityInfo[name].desc)
		} else {
			dmf.Stdoutf("%s\n", name)
		}
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

func (dmf DefaultManifestFinder) LinkAllUtilities(linkType, source, binPath string) error {
	return dmf.linkUtilities(true, linkType, "", source, binPath)
}

func (dmf DefaultManifestFinder) LinkSingleUtility(linkType, name, source, binPath string) error {
	return dmf.linkUtilities(false, linkType, name, source, binPath)
}

func (dmf DefaultManifestFinder) DefaultLinkBinPath() string {

	envPath := dmf.Getenv("HLN_LINK_BIN_PATH")
	if len(envPath) > 0 {
		return envPath
	}

	configPath, err := dmf.Get("link.bin_path")
	if err == nil && len(configPath) > 0 {
		return configPath
	}

	homePath := dmf.Getenv("HOME")
	if len(homePath) > 0 {
		return filepath.Join(homePath, "bin")
	}

	return ""
}

func (dmf DefaultManifestFinder) linkUtilities(all bool, linkType, name, source, binPath string) error {

	if len(binPath) == 0 {
		binPath = dmf.DefaultLinkBinPath()
	}

	binPath, err := homedir.Expand(binPath)
	if err != nil {
		return err
	}

	// TODO: should we create binPath if non-exist?
	binPath, _ = filepath.Abs(binPath)
	binPath, _ = filepath.EvalSymlinks(binPath)

	seenUtilities := make(map[string]bool)

	sourcePaths, err := dmf.Paths(source)
	if err != nil {
		return err
	}

	if len(linkType) == 0 {
		configLinkType, err := dmf.Get("link.type")
		if err == nil && len(configLinkType) > 0 {
			linkType = configLinkType
		} else {
			linkType = "script"
		}
	}

	var linker Linker
	if linkType == "manifest" || linkType == "holen" {
		linker = &FileLinker{dmf.System, dmf.Logger, dmf.SelfPath, binPath}
	} else if linkType == "script" {
		linker = &ScriptLinker{dmf.System, binPath}
	} else if linkType == "alias" {
		linker = &AliasLinker{dmf.System}
	}

	for _, manifestPath := range sourcePaths {
		manifestPath, _ = filepath.Abs(manifestPath)

		if all {
			dmf.Debugf("linking all utilities found in %s in %s", manifestPath, binPath)
			err := dmf.eachManifestPath(manifestPath, func(name, fileName string) error {
				_, ok := seenUtilities[name]
				if ok {
					dmf.Debugf(" seen %s already, skipping", name)
					return nil
				}
				seenUtilities[name] = true
				dmf.Debugf(" linking %s", name)

				if linkType == "manifest" {
					if fileLinker, ok := linker.(*FileLinker); ok {
						fileLinker.Target = filepath.Join(manifestPath, fileName)
					}
				}

				err := dmf.linkUtility(linker, name, filepath.Join(manifestPath, fileName))

				if err != nil {
					return err
				}

				return nil
			})

			if err != nil {
				return err
			}
		} else {
			tryPath := filepath.Join(manifestPath, fmt.Sprintf("%s.yaml", name))
			dmf.Debugf("trying path %s", tryPath)
			// TODO: check if manifestPath is executable and warn
			if dmf.FileExists(tryPath) {
				if linkType == "manifest" {
					if fileLinker, ok := linker.(*FileLinker); ok {
						fileLinker.Target = tryPath
					}
				}

				// link
				err := dmf.linkUtility(linker, name, tryPath)

				if err != nil {
					return err
				}
				return nil
			}
		}
	}
	return nil
}

func (dmf DefaultManifestFinder) linkUtility(linker Linker, name, manifestPath string) error {
	// load up the manifest
	manifest, err := LoadManifest(ParseName(name), manifestPath, dmf.ConfigGetter, dmf.Logger, dmf.System)
	if err != nil {
		return err
	}

	strategies, err := manifest.LoadAllStrategies(ParseName(name))
	if err != nil {
		return err
	}

	// link all found versions
	for _, strategy := range strategies {
		err = linker.Link(name, strategy.Version())
		if err != nil {
			return err
		}
	}

	// link utility without version number
	return linker.Link(name, "")
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
	// by default, higher priority is given for those that have least impact on
	// system and can be shared:
	//   1. cmdio - over an ssh connection, zero local footprint
	//   2. docker - easy distribution, shared between multiple users
	//   3. binary - static binary download
	// Temporarily move cmdio to the last until it's GA
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
		pwdWorkdir, pwdWorkdirOk := strategyData["pwd_workdir"]
		bootstrapScript, bootstrapScriptOk := strategyData["bootstrap_script"]

		if !imageOk {
			return dummy, errors.New("At least 'image' needed for docker strategy to work")
		}

		if !terminalOk {
			terminal = ""
		}
		if !mountPwdAsOk {
			mountPwdAs = ""
		}
		if !bootstrapScriptOk {
			bootstrapScript = ""
		}

		return DockerStrategy{
			StrategyCommon: common,
			Data: DockerData{
				Name:            m.Data.Name,
				Desc:            m.Data.Desc,
				Version:         strategyData["version"].(string),
				Image:           image.(string),
				MountPwd:        mountPwdOk && mountPwd.(bool),
				DockerConn:      dockerConnOk && dockerConn.(bool),
				Interactive:     !interactiveOk || interactive.(bool),
				PidHost:         pidHostOk && pidHost.(bool),
				Terminal:        terminal.(string),
				MountPwdAs:      mountPwdAs.(string),
				RunAsUser:       runAsUserOk && runAsUser.(bool),
				PwdWorkdir:      pwdWorkdirOk && pwdWorkdir.(bool),
				BootstrapScript: bootstrapScript.(string),
				OSArchData:      osArchData,
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
	} else if strategyType == "cmdio" {
		command, commandOk := strategyData["command"]

		if !commandOk {
			return dummy, errors.New("At least 'command' needed for cmdio strategy to work")
		}

		return CmdioStrategy{
			StrategyCommon: common,
			Data: CmdioData{
				Name:       m.Data.Name,
				Desc:       m.Data.Desc,
				Version:    strategyData["version"].(string),
				Command:    command.(string),
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
