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
	system := &DefaultSystem{}
	conf, err := NewDefaultConfigClient(system)
	if err != nil {
		return err
	}

	return runLink(linkCommand, conf, &LogrusLogger{}, system)
}

func runLink(linkCommand LinkCommand, conf ConfigGetter, logger Logger, system System) error {
	selfPath, err := findSelfPath()
	manifestFinder, err := NewManifestFinder(selfPath, conf, logger, system)

	if err != nil {
		return err
	}

	if linkCommand.All {
		for _, manifestPath := range manifestFinder.Paths() {
			err = manifestFinder.Link(manifestPath, linkCommand.HolenPath, linkCommand.BinPath)
			if err != nil {
				return err
			}
		}
	} else if len(linkCommand.ManifestPath) > 0 {
		return manifestFinder.Link(linkCommand.ManifestPath, linkCommand.HolenPath, linkCommand.BinPath)
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
