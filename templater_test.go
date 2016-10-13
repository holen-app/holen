package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplater(t *testing.T) {
	assert := assert.New(t)

	temp := Templater{
		Version:      "1.7",
		OS:           "linux",
		Arch:         "amd64",
		OSArch:       "linux_amd64",
		MappedOSArch: "x86_64",
	}

	var output string
	var err error

	output, err = temp.Template("{{.Version")
	assert.NotNil(err)
	assert.Contains(err.Error(), "unclosed action")
	assert.Equal(output, "")

	output, err = temp.Template("{{.Vern}}")
	assert.NotNil(err)
	assert.Contains(err.Error(), "can't evaluate field")
	assert.Equal(output, "")

	output, err = temp.Template("{{.Version}}")
	assert.Nil(err)
	assert.Equal(output, "1.7")
}
