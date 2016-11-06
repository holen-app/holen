package main

import "fmt"

// ListCommand specifies options for the list subcommand.
type ListCommand struct {
	// nothing yet
}

var listCommand ListCommand

// Listing utilities
func (x *ListCommand) Execute(args []string) error {
	system := &DefaultSystem{}
	conf, err := NewDefaultConfigClient(system)
	if err != nil {
		return err
	}

	return runList(listCommand, conf, &LogrusLogger{}, system)
}

func runList(listCommand ListCommand, conf ConfigGetter, logger Logger, system System) error {
	selfPath, err := findSelfPath()
	manifestFinder, err := NewManifestFinder(selfPath, conf, logger, system)

	if err != nil {
		return err
	}

	return manifestFinder.List()
}

func init() {
	_, err := parser.AddCommand("list",
		"List utilities.",
		"",
		&listCommand)

	if err != nil {
		fmt.Println(err)
	}
}
