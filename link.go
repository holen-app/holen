package main

import "fmt"

// LinkCommand specifies options for the link subcommand.
type LinkCommand struct {
	// nothing yet
	ManifestPath string `short:"m" long:"manifest-path" description:"Link manifests in this path."`
	All          bool   `short:"a" long:"all" description:"Link manifests in all manifest paths found."`
	HolenPath    string `short:"h" long:"holen-path" description:"Link to this holen path (defaults to self)."`
	BinPath      string `short:"b" long:"bin-path" description:"Link from this bin path." required:"true"`
}

var linkCommand LinkCommand

// Linking utilities
func (x *LinkCommand) Execute(args []string) error {
	manifestFinder, err := NewManifestFinder()
	if err != nil {
		return err
	}

	if linkCommand.All {
		return manifestFinder.LinkAll(linkCommand.HolenPath, linkCommand.BinPath)
	} else if len(linkCommand.ManifestPath) > 0 {
		return manifestFinder.LinkSingle(linkCommand.ManifestPath, linkCommand.HolenPath, linkCommand.BinPath)
	} else {
		return fmt.Errorf("either --all or --manifest-path argument is required")
	}

	return nil
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
