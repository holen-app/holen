package main

import "fmt"

// LinkCommand specifies options for the link subcommand.
type LinkCommand struct {
	All     bool   `short:"a" long:"all" description:"Link all manifests in all manifest paths found"`
	Source  string `short:"s" long:"source" description:"Only look for manifests in this source"`
	BinPath string `short:"b" long:"bin-path" description:"Link from this bin path"`
	Args    struct {
		Name string `description:"utility name" positional-arg-name:"<name>"`
	} `positional-args:"yes"`
}

var linkCommand LinkCommand

// Linking utilities
func (x *LinkCommand) Execute(args []string) error {
	manifestFinder, err := NewManifestFinder()
	if err != nil {
		return err
	}

	if linkCommand.All {
		return manifestFinder.LinkAllUtilities(linkCommand.Source, linkCommand.BinPath)
	} else if len(linkCommand.Args.Name) > 0 {
		return manifestFinder.LinkSingleUtility(linkCommand.Args.Name, linkCommand.Source, linkCommand.BinPath)
	} else {
		return fmt.Errorf("either --all or <name> argument is required")
	}
}

func init() {
	_, err := parser.AddCommand("link",
		"Link utilities.",
		"",
		&linkCommand)

	if err != nil {
		fmt.Println(err)
	}
}
