package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

func main() {
	m := Manifest{}

	basename := path.Base(os.Args[0])

	if basename == "holen" || basename == "hln" {
		fmt.Println("TODO: parsing holen args")
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
