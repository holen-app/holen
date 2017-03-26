package main

import "fmt"

// RunCommand specifies options for the run subcommand.
type RunCommand struct {
	Version string `short:"v" long:"version" description:"Run this version of the utility."`
	Args    struct {
		Name string `description:"utility name" positional-arg-name:"<name>"`
	} `positional-args:"yes"`
}

var runCommand RunCommand

// Runing utilities
func (x *RunCommand) Execute(args []string) error {
	manifestFinder, err := NewManifestFinder(true)
	if err != nil {
		return err
	}

	nameVer := NameVer{runCommand.Args.Name, runCommand.Version}

	manifest, err := manifestFinder.Find(nameVer)
	if err != nil {
		return err
	}

	return manifest.Run(nameVer, args)
}

func init() {
	_, err := parser.AddCommand("run",
		"Run utilities.",
		"",
		&runCommand)

	if err != nil {
		fmt.Println(err)
	}
}
