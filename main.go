package main

import (
	"fmt"
	"io/ioutil"
	"log"

	yaml "gopkg.in/yaml.v2"
)

func main() {
	m := Manifest{}

	data, err := ioutil.ReadFile("manifests/docker-compose.yaml")
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	err = yaml.Unmarshal([]byte(data), &m)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	// load this from config or by detecting environment
	defaultStrategy := "docker"

	strategy, err := loadStrategy(m, defaultStrategy, "")
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Printf("%v\n", strategy)
}
