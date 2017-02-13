package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Linker interface {
	Link(string, string) error
}

type FileLinker struct {
	System
	Logger
	Target, BinPath string
}

func (fl FileLinker) Link(name, version string) error {
	// fl.Stdoutf("Linking %s (v: %s) to %s in %s\n", name, version, fl.Target, fl.BinPath)

	targetPath, err := filepath.Rel(fl.BinPath, fl.Target)
	if err != nil {
		return err
	}

	if strings.HasSuffix(targetPath, fl.Target) {
		targetPath = fl.Target
	}

	fl.Debugf("  target path %s", targetPath)

	fullBinPath := filepath.Join(fl.BinPath, name)
	if len(version) > 0 {
		fullBinPath = fmt.Sprintf("%s--%s", fullBinPath, version)
	}
	fl.Debugf("  full bin path %s", fullBinPath)

	fl.Debugf("linking %s to %s", targetPath, fullBinPath)

	err = removeOldLink(fullBinPath)
	if err != nil {
		return err
	}

	// symlink to holen
	err = os.Symlink(fl.Target, fullBinPath)
	if err != nil {
		return err
	}
	return nil
}

type AliasLinker struct {
	System
}

func (al AliasLinker) Link(name, version string) error {
	var versionSuffix, versionFlag string
	if len(version) > 0 {
		versionSuffix = fmt.Sprintf("--%s", version)
		versionFlag = fmt.Sprintf(" --version %s", version)
	}

	al.Stdoutf("alias %s%s=\"holen run%s %s -- \"\n", name, versionSuffix, versionFlag, name)

	return nil
}

type ScriptLinker struct {
	System
	BinPath string
}

func (sl ScriptLinker) Link(name, version string) error {
	// sl.Stdoutf("Creating small script for %s (v: %s) in %s", name, version, sl.BinPath)

	fullBinPath := filepath.Join(sl.BinPath, name)
	if len(version) > 0 {
		fullBinPath = fmt.Sprintf("%s--%s", fullBinPath, version)
	}

	err := removeOldLink(fullBinPath)
	if err != nil {
		return err
	}

	var versionFlag string
	if len(version) > 0 {
		versionFlag = fmt.Sprintf(" --version %s", version)
	}

	scriptTmpl := `#!/bin/sh
exec holen run{{ .VersionFlag }} {{ .Name }} -- "$@"
`

	scriptData := struct {
		VersionFlag, Name string
	}{
		versionFlag, name,
	}

	tmpl := template.Must(template.New("script").Parse(scriptTmpl))
	var scriptBytes bytes.Buffer

	err = tmpl.Execute(&scriptBytes, scriptData)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fullBinPath, scriptBytes.Bytes(), 0755)
}

func removeOldLink(fullPath string) error {
	fileStat, err := os.Lstat(fullPath)
	if err == nil {
		if fileStat.Mode()&os.ModeSymlink != 0 {
			target, _ := os.Readlink(fullPath)

			// TODO: check more thoroughly for links that are created by holen
			if strings.HasSuffix(target, "holen") || strings.HasSuffix(target, ".yaml") {
				os.Remove(fullPath)
				return nil
			}
			return fmt.Errorf("non-holen symlink found at %s", fullPath)
		} else if fileStat.Mode().IsRegular() {
			if fileStat.Size() < 500 {
				data, _ := ioutil.ReadFile(fullPath)
				if strings.Contains(string(data), "holen run") {
					os.Remove(fullPath)
					return nil
				}
			}
		}
		return fmt.Errorf("non-holen file found at %s", fullPath)
	}

	return nil
}
