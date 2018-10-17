package main

import "fmt"

type AddSourceCommand struct {
	System bool `short:"s" long:"system" description:"Modify system level configuration."`
	Args   struct {
		Name string `description:"source name" positional-arg-name:"<name>"`
		Spec string `description:"source spec" positional-arg-name:"<spec>"`
	} `positional-args:"yes" required:"yes"`
}

type ListSourceCommand struct{}

type UpdateSourceCommand struct {
	Args struct {
		Name string `description:"source name" positional-arg-name:"<name>"`
	} `positional-args:"yes"`
}

type DeleteSourceCommand struct {
	System bool `short:"s" long:"system" description:"Modify system level configuration."`
	Args   struct {
		Name string `description:"source name" positional-arg-name:"<name>"`
	} `positional-args:"yes" required:"yes"`
}

type ShowSourceCommand struct {
	Path bool `short:"p" long:"path" description:"Show local filesystem path."`
	Spec bool `short:"s" long:"spec" description:"Show source specification."`
	Args struct {
		Name string `description:"source name" positional-arg-name:"<name>"`
	} `positional-args:"yes" required:"yes"`
}

type SourceCommand struct {
	Add    AddSourceCommand    `command:"add" description:"Add a source"`
	List   ListSourceCommand   `command:"list" alias:"ls" description:"List sources"`
	Update UpdateSourceCommand `command:"update" alias:"up" alias:"fetch" description:"Update sources"`
	Delete DeleteSourceCommand `command:"delete" alias:"rm" description:"Delete source"`
	Show   ShowSourceCommand   `command:"show" description:"Show source information"`
}

func (r *AddSourceCommand) Execute(args []string) error {
	sourceManager, err := NewDefaultSourceManager()
	if err != nil {
		return err
	}

	return sourceManager.Add(r.System, r.Args.Name, r.Args.Spec)
}

func (r *ListSourceCommand) Execute(args []string) error {
	sourceManager, err := NewDefaultSourceManager()
	if err != nil {
		return err
	}

	return sourceManager.List()
}

func (r *UpdateSourceCommand) Execute(args []string) error {
	sourceManager, err := NewDefaultSourceManager()
	if err != nil {
		return err
	}

	return sourceManager.Update(r.Args.Name)
}

func (r *DeleteSourceCommand) Execute(args []string) error {
	sourceManager, err := NewDefaultSourceManager()
	if err != nil {
		return err
	}

	return sourceManager.Delete(r.System, r.Args.Name)
}

func (r *ShowSourceCommand) Execute(args []string) error {
	sourceManager, err := NewDefaultSourceManager()
	if err != nil {
		return err
	}

	if r.Path {
		return sourceManager.Show(r.Args.Name, "path")
	} else if r.Spec {
		return sourceManager.Show(r.Args.Name, "spec")
	}

	return fmt.Errorf("please specify -p/--path or -s/--spec")
}

func init() {
	var sourceCommand SourceCommand

	cmd, err := parser.AddCommand("source",
		"Manage manifest sources.",
		"",
		&sourceCommand)

	cmd.Aliases = append(cmd.Aliases, "src")

	if err != nil {
		fmt.Println(err)
	}
}
