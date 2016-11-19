package main

import "fmt"

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

type RealSourceManager struct {
	ConfigClient
	System
}

func (rsm RealSourceManager) Add(system bool, name, spec string) error {
	key := fmt.Sprintf("source.%s", name)

	val, err := rsm.Get(key)
	if err != nil {
		return err
	}

	if len(val) > 0 {
		return fmt.Errorf("source %s already exists", name)
	}

	return rsm.Set(system, key, spec)
}

func (rsm RealSourceManager) List() error {
	rsm.Stdoutf("TODO: Listing sources\n")

	return nil
}

func (rsm RealSourceManager) Update(name string) error {
	rsm.Stdoutf("TODO: Updating %s\n", name)

	// TODO: check for nonexisting source
	return nil
}

func (rsm RealSourceManager) Delete(system bool, name string) error {
	rsm.Stdoutf("Deleting %s\n", name)

	// TODO: check for nonexisting source
	return rsm.Unset(system, fmt.Sprintf("source.%s", name))
}

func NewDefaultSourceManager() (*RealSourceManager, error) {
	system := &DefaultSystem{}
	conf, err := NewDefaultConfigClient(system)
	if err != nil {
		return nil, err
	}

	return &RealSourceManager{
		ConfigClient: conf,
		System:       system,
	}, nil
}
