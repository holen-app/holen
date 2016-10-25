package main

import (
	"fmt"
	"os"
)

// InspectCommand specifies options for the inspect subcommand.
type InspectCommand struct {
	Manifest string `short:"m" long:"manifest" description:"Manifest file, specify to override search."`
	Version  string `short:"v" long:"version" description:"Version of the utility to inspect. (optional)"`
	Args     struct {
		Name string `description:"Name of utility."`
	} `positional-args:"yes" required:"yes"`
}

var inspectCommand InspectCommand

// Execute inspecting utilities and how they are run
func (x *InspectCommand) Execute(args []string) error {
	conf, err := NewDefaultConfigClient()
	if err != nil {
		return err
	}
	logger := &LogrusLogger{}

	return runInspect(inspectCommand, conf, logger)
}

func runInspect(inspectCommand InspectCommand, conf ConfigGetter, logger Logger) error {
	var err error

	utility := NameVer{inspectCommand.Args.Name, inspectCommand.Version}

	var manifest *Manifest
	if len(inspectCommand.Manifest) == 0 {
		selfPath, err := findSelfPath()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		manifestFinder, err := NewManifestFinder(selfPath, conf, logger)
		if err != nil {
			return err
		}

		manifest, err = manifestFinder.Find(utility)
		if err != nil {
			return err
		}
	} else {
		manifest, err = LoadManifest(utility, inspectCommand.Manifest, conf, logger)
		if err != nil {
			return err
		}
	}

	if len(utility.Version) == 0 {
		allStrategies, err := manifest.LoadAllStrategies()
		if err != nil {
			return err
		}

		for strategyName, strategies := range allStrategies {
			manifest.Stderrf("%s\n", strategyName)
			for _, strategy := range strategies {
				strategy.Inspect()
			}
		}
		// fmt.Printf("strategies: %# v\n", pretty.Formatter(allStrategies))
	} else {
		strategies, err := manifest.LoadStrategies(utility)
		if err != nil {
			return err
		}

		for _, strategy := range strategies {
			strategy.Inspect()
		}
	}
	return nil
}

func init() {
	_, err := parser.AddCommand("inspect",
		"Inspect information about a utility.",
		"",
		&inspectCommand)

	if err != nil {
		fmt.Println(err)
	}
}
