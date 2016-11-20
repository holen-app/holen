package main

import "fmt"

// ListCommand specifies options for the list subcommand.
type ListCommand struct {
	Source string `short:"s" long:"source" description:"Only look for manifests in this source"`
}

var listCommand ListCommand

// Listing utilities
func (x *ListCommand) Execute(args []string) error {
	manifestFinder, err := NewManifestFinder()
	if err != nil {
		return err
	}

	return manifestFinder.List(listCommand.Source)
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
