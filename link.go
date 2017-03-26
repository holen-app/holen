package main

import "fmt"

// LinkCommand specifies options for the link subcommand.
type LinkCommand struct {
	All        bool   `short:"a" long:"all" description:"Link all manifests in all manifest paths found"`
	Source     string `short:"s" long:"source" description:"Only look for manifests in this source"`
	BinPath    string `short:"b" long:"bin-path" description:"Link from this bin path"`
	Type       string `short:"t" long:"type" description:"Type of link to create" choice:"script" choice:"alias" choice:"holen" choice:"manifest"`
	OnlyLatest bool   `short:"o" long:"only-latest" description:"Only link the latest; not all versions"`
	Args       struct {
		Name string `description:"utility name" positional-arg-name:"<name>"`
	} `positional-args:"yes"`
}

var linkCommand LinkCommand

// Linking utilities
func (x *LinkCommand) Execute(args []string) error {
	manifestFinder, err := NewManifestFinder(true)
	if err != nil {
		return err
	}

	if linkCommand.All {
		return manifestFinder.LinkAllUtilities(linkCommand.Type, linkCommand.Source, linkCommand.BinPath, linkCommand.OnlyLatest)
	} else if len(linkCommand.Args.Name) > 0 {
		return manifestFinder.LinkSingleUtility(linkCommand.Type, linkCommand.Args.Name, linkCommand.Source, linkCommand.BinPath, linkCommand.OnlyLatest)
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
