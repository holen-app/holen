package main

import (
	"fmt"
	"io/ioutil"

	"github.com/kr/pretty"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

type ManifestFinder interface {
	Find(string) (*Manifest, error)
}

type DefaultManifestFinder struct {
	Logger
	ConfigGetter
}

func NewManifestFinder() (*DefaultManifestFinder, error) {
	conf, err := NewDefaultConfigClient()
	if err != nil {
		return nil, err
	}

	logger := &LogrusLogger{}
	return &DefaultManifestFinder{
		Logger:       logger,
		ConfigGetter: conf,
	}, nil
}

func (dmf DefaultManifestFinder) Find(utility NameVer) (*Manifest, error) {
	md := ManifestData{}

	file := fmt.Sprintf("manifests/%s.yaml", utility.Name)

	dmf.Infof("attemting to load: %s", file)
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrap(err, "problems with reading file")
	}

	err = yaml.Unmarshal([]byte(data), &md)
	if err != nil {
		return nil, errors.Wrap(err, "problems with unmarshal")
	}

	manifest := &Manifest{
		Logger:       dmf.Logger,
		ConfigGetter: dmf.ConfigGetter,
		Data:         md,
	}
	dmf.Debugf("manifest found: %# v\n", pretty.Formatter(manifest))

	return manifest, nil
}
