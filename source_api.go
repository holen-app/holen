package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type SourcePather interface {
	Paths() []string
}

type SourceManager interface {
	SourcePather
	Add(bool, string, string) error
	List() error
	Update(string) error
	Delete(bool, string) error
}

type Source interface {
	Name() string
	Spec() string
	Info() string
	Update(string) error
	Delete(string) error
}

type RealSourceManager struct {
	Logger
	ConfigClient
	System
}

type GitSource struct {
	System
	Logger
	Runner
	name string
	spec string
}

func (gs GitSource) Name() string {
	return gs.name
}

func (gs GitSource) Spec() string {
	return gs.spec
}

func (gs GitSource) fullUrl() string {
	if strings.HasSuffix(gs.spec, ".git") {
		return gs.spec
	} else if regexp.MustCompile(`^[0-9a-z-_]+/[0-9a-z-_]+$`).MatchString(gs.spec) {
		return fmt.Sprintf("https://github.com/%s.git", gs.spec)
	} else if regexp.MustCompile(`^[^/]+/[^/]+/[^/]+$`).MatchString(gs.spec) {
		return fmt.Sprintf("https://%s.git", gs.spec)
	}
	return ""
}

func (gs GitSource) Info() string {
	return fmt.Sprintf("git source: %s", gs.fullUrl())
}

func (gs GitSource) Update(base string) error {

	clonePath := filepath.Join(base, gs.name)

	if gs.FileExists(clonePath) {
		wd, _ := os.Getwd()
		os.Chdir(clonePath)
		defer os.Chdir(wd)
		return gs.RunCommand("git", []string{"pull"})
	}
	return gs.RunCommand("git", []string{"clone", gs.fullUrl(), clonePath})
}

func (gs GitSource) Delete(base string) error {
	sourcePath := filepath.Join(base, gs.Name())

	return os.RemoveAll(sourcePath)
}

func (rsm RealSourceManager) Add(system bool, name, spec string) error {
	source, err := rsm.getSource(name)
	if err != nil {
		return err
	}

	if source != nil {
		return fmt.Errorf("source %s already exists", name)
	}

	return rsm.Set(system, fmt.Sprintf("source.%s", name), spec)
}

func (rsm RealSourceManager) getSources() ([]Source, error) {
	sources := []Source{}

	allConfig, err := rsm.GetAll()
	if err != nil {
		return sources, err
	}

	runner := &DefaultRunner{rsm.Logger}
	for key, val := range allConfig {
		if strings.HasPrefix(key, "source.") {
			name := strings.TrimPrefix(key, "source.")
			spec := val
			// TODO: support other source types
			sources = append(sources, GitSource{rsm.System, rsm.Logger, runner, name, spec})
		}
	}

	// append the master source
	sources = append(sources, GitSource{rsm.System, rsm.Logger, runner, "main", "justone/holen-manifests"})

	return sources, nil
}

func (rsm RealSourceManager) getSource(name string) (Source, error) {
	sources, err := rsm.getSources()
	if err != nil {
		return nil, err
	}

	var foundSource Source
	for _, source := range sources {
		if source.Name() == name {
			foundSource = source
		}
	}

	return foundSource, nil
}

func (rsm RealSourceManager) manifestsPath() (string, error) {
	dataPath, err := rsm.DataPath()
	if err != nil {
		return "", err
	}

	manifestsPath := filepath.Join(dataPath, "manifests")
	os.MkdirAll(manifestsPath, 0755)

	return manifestsPath, nil
}

func (rsm RealSourceManager) List() error {
	sources, err := rsm.getSources()
	if err != nil {
		return err
	}

	for _, source := range sources {
		rsm.Stdoutf("%s: %s (%s)\n", source.Name(), source.Spec(), source.Info())
	}

	return nil
}

func (rsm RealSourceManager) Update(name string) error {
	sources, err := rsm.getSources()
	if err != nil {
		return err
	}

	manifestsPath, err := rsm.manifestsPath()
	if err != nil {
		return err
	}

	for _, source := range sources {
		// rsm.Stdoutf("%s: %s (%s)\n", source.Name(), source.Spec(), source.Info())
		if len(name) == 0 || name == source.Name() {
			source.Update(manifestsPath)
		}
	}

	return nil
}

func (rsm RealSourceManager) Delete(system bool, name string) error {
	source, err := rsm.getSource(name)
	if err != nil {
		return err
	}

	if source == nil {
		return fmt.Errorf("source %s not found", name)
	}

	manifestsPath, err := rsm.manifestsPath()
	if err != nil {
		return err
	}

	source.Delete(manifestsPath)

	return rsm.Unset(system, fmt.Sprintf("source.%s", name))
}

func NewDefaultSourceManager() (*RealSourceManager, error) {
	system := &DefaultSystem{}
	conf, err := NewDefaultConfigClient(system)
	if err != nil {
		return nil, err
	}

	return &RealSourceManager{
		Logger:       &LogrusLogger{},
		ConfigClient: conf,
		System:       system,
	}, nil
}
