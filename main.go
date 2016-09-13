package main

import (
	"fmt"
	"os"
	"path"

	"github.com/Sirupsen/logrus"
	flags "github.com/jessevdk/go-flags"
)

type GlobalOptions struct {
	Quiet   func(string) `env:"HLN_QUIET" short:"q" long:"quiet" description:"Show as little information as possible."`
	Verbose func(string) `env:"HLN_VERBOSE" short:"v" long:"verbose" description:"Show verbose debug information."`
	LogJSON func(string) `env:"HLN_LOG_JSON" short:"j" long:"log-json" description:"Log in JSON format."`
}

type InlineOptions struct {
	Verbose func(string) `env:"HLN_VERBOSE" long:"hln-verbose" description:"Show verbose debug information."`
	LogJSON func(string) `env:"HLN_LOG_JSON" long:"hln-log-json" description:"Log in JSON format."`
}

var globalOptions GlobalOptions
var inlineOptions InlineOptions

var parser = flags.NewParser(&globalOptions, flags.Default)
var inlineParser = flags.NewParser(&inlineOptions, flags.PrintErrors|flags.IgnoreUnknown)
var originalArgs []string

func main() {
	basename := path.Base(os.Args[0])
	utilityNameOverride := os.Getenv("HLN_UTILITY")

	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	if (basename == "holen" || basename == "hln") && len(utilityNameOverride) == 0 {

		// configure logging
		logrus.SetLevel(logrus.InfoLevel)

		// options to change log level
		globalOptions.Quiet = func(v string) {
			logrus.SetLevel(logrus.WarnLevel)
		}
		globalOptions.Verbose = func(v string) {
			logrus.SetLevel(logrus.DebugLevel)
		}
		globalOptions.LogJSON = func(v string) {
			logrus.SetFormatter(&logrus.JSONFormatter{})
		}

		if _, err := parser.Parse(); err != nil {
			os.Exit(1)
		}
	} else {

		// configure logging
		logrus.SetLevel(logrus.WarnLevel)

		// options to change log level
		inlineOptions.Verbose = func(v string) {
			logrus.SetLevel(logrus.DebugLevel)
		}
		inlineOptions.LogJSON = func(v string) {
			logrus.SetFormatter(&logrus.JSONFormatter{})
		}

		args, err := inlineParser.Parse()
		if err != nil {
			os.Exit(1)
		}

		if len(utilityNameOverride) > 0 {
			basename = utilityNameOverride
		}

		err = RunUtility(basename, args)
		if err != nil {
			fmt.Printf("Unable to run %s: %s\n", basename, err)
			os.Exit(1)
		}
	}
}
