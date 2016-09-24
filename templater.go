package main

import (
	"bytes"
	"html/template"
)

// Templater contains fields for each piece of data that can be templated.
type Templater struct {
	Version    string
	OS         string
	Arch       string
	OSArch     string
	MappedArch string
}

// Template takes an input string and templates it with the data contained in
// the Templater struct.
func (temp Templater) Template(input string) (string, error) {
	tmpl := template.Must(template.New("temp").Parse(input))
	var output bytes.Buffer

	err := tmpl.Execute(&output, temp)
	if err != nil {
		return "", err
	}

	return output.String(), nil
}
