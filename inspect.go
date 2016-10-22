package main

import "fmt"

// InspectCommand specifies options for the inspect subcommand.
type InspectCommand struct {
	Manifest string `short:"m" long:"manifest" description:"Manifest file, specify to override search."`
	Name     string `short:"n" long:"name" description:"Name of the utility to inspect." required:"true"`
	Version  string `short:"v" long:"version" description:"Version of the utility to inspect. (optional)"`
}

var inspectCommand InspectCommand

// Execute handles setting, getting, and listing configuration values.
func (x *InspectCommand) Execute(args []string) error {
	conf, err := NewDefaultConfigClient()
	if err != nil {
		return err
	}
	logger := &LogrusLogger{}

	// TODO: handle case where manifest not specified (i.e. search path)
	manifest, err := LoadManifest(NameVer{inspectCommand.Name, inspectCommand.Version}, inspectCommand.Manifest, conf, logger)
	if err != nil {
		return err
	}

	allStrategies, err := manifest.LoadAllStrategies()
	if err != nil {
		return err
	}

	for strategyName, strategies := range allStrategies {
		manifest.UserMessage("%s\n", strategyName)
		for _, strategy := range strategies {
			strategy.Inspect()
		}
	}
	// fmt.Printf("strategies: %# v\n", pretty.Formatter(allStrategies))
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
