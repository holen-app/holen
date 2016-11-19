package main

import "fmt"

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
	system := &DefaultSystem{}
	conf, err := NewDefaultConfigClient(system)
	if err != nil {
		return err
	}

	return runInspect(inspectCommand, conf, &LogrusLogger{}, system)
}

func runInspect(inspectCommand InspectCommand, conf ConfigGetter, logger Logger, system System) error {
	var err error

	utility := NameVer{inspectCommand.Args.Name, inspectCommand.Version}

	var manifest *Manifest
	if len(inspectCommand.Manifest) == 0 {
		manifestFinder, err := NewManifestFinder()
		if err != nil {
			return err
		}

		manifest, err = manifestFinder.Find(utility)
		if err != nil {
			return err
		}
	} else {
		manifest, err = LoadManifest(utility, inspectCommand.Manifest, conf, logger, system)
		if err != nil {
			return err
		}
	}

	if len(utility.Version) == 0 {
		allStrategies, err := manifest.LoadAllStrategies(utility)
		if err != nil {
			return err
		}

		for _, strategy := range allStrategies {
			err = strategy.Inspect()
			if err != nil {
				return err
			}
		}
	} else {
		strategies, err := manifest.LoadStrategies(utility)
		if err != nil {
			return err
		}

		for _, strategy := range strategies {
			err = strategy.Inspect()
			if err != nil {
				return err
			}
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
