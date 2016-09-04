package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/Sirupsen/logrus"
	flags "github.com/jessevdk/go-flags"

	yaml "gopkg.in/yaml.v2"
)

type GlobalOptions struct {
	Quiet   func() `short:"q" long:"quiet" description:"Show as little information as possible."`
	Verbose func() `short:"v" long:"verbose" description:"Show verbose debug information."`
	LogJSON func() `short:"j" long:"log-json" description:"Log in JSON format."`
}

var globalOptions GlobalOptions
var parser = flags.NewParser(&globalOptions, flags.Default)
var originalArgs []string

func main() {
	m := Manifest{}

	basename := path.Base(os.Args[0])

	if basename == "holen" || basename == "hln" {

		// configure logging
		logrus.SetLevel(logrus.InfoLevel)
		logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

		// options to change log level
		globalOptions.Quiet = func() {
			logrus.SetLevel(logrus.WarnLevel)
		}
		globalOptions.Verbose = func() {
			logrus.SetLevel(logrus.DebugLevel)
		}
		globalOptions.LogJSON = func() {
			logrus.SetFormatter(&logrus.JSONFormatter{})
		}
		originalArgs = os.Args
		if _, err := parser.Parse(); err != nil {
			os.Exit(1)
		}
	} else {

		parts := strings.Split(basename, "--")
		version := ""
		if len(parts) > 1 {
			version = parts[1]
		}

		// fmt.Println(parts)
		file := fmt.Sprintf("manifests/%s.yaml", parts[0])
		// fmt.Println(file)
		data, err := ioutil.ReadFile(file)
		if err != nil {
			log.Fatalf("error: %v", err)
		}

		err = yaml.Unmarshal([]byte(data), &m)
		if err != nil {
			log.Fatalf("error: %v", err)
		}

		// load this from config or by detecting environment
		defaultStrategy := "docker"

		strategy, err := loadStrategy(m, defaultStrategy, version)
		if err != nil {
			log.Fatalf("error: %v", err)
		}

		fmt.Printf("%v\n", strategy)
	}
}
