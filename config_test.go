package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	assert := assert.New(t)

	tempdir, _ := ioutil.TempDir("", "holen")
	defer os.RemoveAll(tempdir)

	system := NewMemSystem()
	system.Setenv("HOME", tempdir)

	conf, err := NewDefaultConfigClient(system)
	assert.Equal(conf.systemConfig, "/etc/holenconfig")
	assert.Equal(conf.userConfig, fmt.Sprintf("%s/.config/holen/config", tempdir))
	assert.Nil(err)

	tempdir2, _ := ioutil.TempDir("", "holen")
	defer os.RemoveAll(tempdir2)
	system.Setenv("XDG_CONFIG_HOME", tempdir2)
	defer os.Unsetenv("XDG_CONFIG_HOME")
	system.Setenv("HOLEN_SYSTEM_CONFIG", tempdir2)
	defer os.Unsetenv("HOLEN_SYSTEM_CONFIG")

	conf, err = NewDefaultConfigClient(system)
	assert.Equal(conf.systemConfig, fmt.Sprintf("%s/holenconfig", tempdir2))
	assert.Equal(conf.userConfig, fmt.Sprintf("%s/holen/config", tempdir2))
	assert.Nil(err)
}

func TestSetGetUnset(t *testing.T) {
	assert := assert.New(t)
	var err error
	var val string

	tempdir, _ := ioutil.TempDir("", "holen")
	defer os.RemoveAll(tempdir)

	system := NewMemSystem()
	system.Setenv("HOME", tempdir)
	system.Setenv("HOLEN_SYSTEM_CONFIG", tempdir)
	defer os.Unsetenv("HOLEN_SYSTEM_CONFIG")

	conf, err := NewDefaultConfigClient(system)

	err = conf.Set(true, "section.key", "value")
	assert.Nil(err)
	val, err = conf.Get("section.key")
	assert.Equal(val, "value")

	err = conf.Set(false, "section.key", "other")
	assert.Nil(err)
	val, err = conf.Get("section.key")
	assert.Equal(val, "other")

	err = conf.Unset(false, "section.key")
	assert.Nil(err)
	val, err = conf.Get("section.key")
	assert.Equal(val, "value")

	all, err := conf.GetAll()
	assert.Nil(err)
	assert.Equal(all, map[string]string{"section.key": "value"})

	err = conf.Unset(true, "section.key")
	assert.Nil(err)
	val, err = conf.Get("section.key")
	assert.Equal(val, "")
}
