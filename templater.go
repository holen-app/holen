package main

import (
	"bytes"
	"html/template"
)

type Templater struct {
	Version    string
	OS         string
	Arch       string
	MappedArch string
}

func (temp Templater) Template(input string) (string, error) {
	tmpl := template.Must(template.New("temp").Parse(input))
	var output bytes.Buffer

	err := tmpl.Execute(&output, temp)
	if err != nil {
		return "", err
	}

	return output.String(), nil
}
