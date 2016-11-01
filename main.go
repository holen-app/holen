package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/Sirupsen/logrus"
	flags "github.com/jessevdk/go-flags"
)

// GlobalOptions are options that are used when holen is run directly.
type GlobalOptions struct {
	Quiet   func(string) `env:"HLN_QUIET" short:"q" long:"quiet" description:"Show as little information as possible."`
	Verbose func(string) `env:"HLN_VERBOSE" short:"v" long:"verbose" description:"Show verbose debug information."`
	LogJSON func(string) `env:"HLN_LOG_JSON" short:"j" long:"log-json" description:"Log in JSON format."`
}

// InlineOptions are options that are used when holen is run indirectly via a symlink.
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
	if utilityNameOverride := os.Getenv("HLN_UTILITY"); len(utilityNameOverride) > 0 {
		basename = utilityNameOverride
	}

	selfPath, err := findSelfPath()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	// configure logging
	logrus.SetLevel(logrus.InfoLevel)

	if basename == "holen" || basename == "hln" || strings.HasPrefix(basename, "holen") {

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

		err = RunUtility(selfPath, basename, args)
		if err != nil {
			fmt.Printf("Unable to run %s: %s\n", basename, err)
			os.Exit(1)
		}
	}
}

// RunUtility will run the specified utility with arguments.
func RunUtility(selfPath, utility string, args []string) error {
	conf, err := NewDefaultConfigClient()
	if err != nil {
		return err
	}

	logger := &LogrusLogger{}
	manifestFinder, err := NewManifestFinder(selfPath, conf, logger)
	if err != nil {
		return err
	}

	nameVer := ParseName(utility)

	manifest, err := manifestFinder.Find(nameVer)
	if err != nil {
		return err
	}

	return manifest.Run(nameVer, args)
}
