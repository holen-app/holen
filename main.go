package main

import (
	"os"
	"path"

	"github.com/Sirupsen/logrus"
	flags "github.com/jessevdk/go-flags"
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

		logger := &LogrusLogger{}
		RunUtility(logger, basename)
	}
}
