package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
	flags "github.com/jessevdk/go-flags"
)

// GlobalOptions are options that are used when holen is run directly.
type GlobalOptions struct {
	Quiet   func() `short:"q" long:"quiet" description:"Show as little information as possible."`
	Verbose func() `short:"v" long:"verbose" description:"Show verbose debug information."`
	LogJSON func() `short:"j" long:"log-json" description:"Log in JSON format."`
}

// InlineOptions are options that are used when holen is run indirectly via a symlink.
type InlineOptions struct {
	Version string       `env:"HLN_VERSION" long:"hln-version" description:"Use specified version."`
	Verbose func(string) `env:"HLN_VERBOSE" long:"hln-verbose" description:"Show verbose debug information."`
	LogJSON func(string) `env:"HLN_LOG_JSON" long:"hln-log-json" description:"Log in JSON format."`
}

var globalOptions GlobalOptions
var inlineOptions InlineOptions

var parser = flags.NewParser(&globalOptions, flags.Default)
var inlineParser = flags.NewParser(&inlineOptions, flags.PrintErrors|flags.IgnoreUnknown)
var originalArgs []string

func main() {
	basename := filepath.Base(os.Args[0])
	if utilityNameOverride := os.Getenv("HLN_UTILITY"); len(utilityNameOverride) > 0 {
		basename = utilityNameOverride
	}

	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	// configure logging
	logrus.SetLevel(logrus.InfoLevel)

	if basename == "holen" || basename == "hln" || strings.HasPrefix(basename, "holen") {
		var utility NameVer
		var manifestFile string

		system := &DefaultSystem{}
		conf, err := NewDefaultConfigClient(system)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if len(os.Args) >= 2 {
			firstArg := os.Args[1]
			fileStat, err := os.Lstat(firstArg)
			if err == nil && fileStat.Mode()&os.ModeSymlink != 0 {
				utility = ParseName(filepath.Base(firstArg))

				manifestFile = firstArg
			} else if strings.HasSuffix(firstArg, ".yaml") {
				name := strings.TrimSuffix(filepath.Base(firstArg), ".yaml")
				utility = NameVer{name, ""}
				manifestFile = firstArg
			} else {
				_, err := LoadManifest(NameVer{}, firstArg, conf, &LogrusLogger{}, system)
				if err == nil {
					utility = ParseName(filepath.Base(firstArg))
					manifestFile = firstArg
				}
			}
		}

		if len(utility.Name) > 0 {
			// options to change log level
			inlineOptions.Verbose = func(v string) {
				logrus.SetLevel(logrus.DebugLevel)
			}
			inlineOptions.LogJSON = func(v string) {
				logrus.SetFormatter(&logrus.JSONFormatter{})
			}

			args, err := inlineParser.ParseArgs(os.Args[2:])

			if len(inlineOptions.Version) > 0 {
				utility.Version = inlineOptions.Version
			}

			manifest, err := LoadManifest(utility, manifestFile, conf, &LogrusLogger{}, system)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			// fmt.Println(os.Args)
			// fmt.Println(os.Args[2:])
			// fmt.Println(args)
			err = manifest.Run(utility, args)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		} else {
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

			if _, err := parser.Parse(); err != nil {
				os.Exit(1)
			}
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

		err = RunUtility(basename, args)
		if err != nil {
			fmt.Printf("Unable to run %s: %s\n", basename, err)
			os.Exit(1)
		}
	}
}

// RunUtility will run the specified utility with arguments.
func RunUtility(utility string, args []string) error {
	manifestFinder, err := NewManifestFinder(true)
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
