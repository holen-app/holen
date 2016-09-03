package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	ini "gopkg.in/ini.v1"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	m := Manifest{}

	basename := path.Base(os.Args[0])

	if basename == "holen" || basename == "hln" {
		fmt.Println("TODO: parsing holen args")
		// cfg, err := ini.LooseLoad("/home/nate/.holenconfig")
		// if err != nil {
		// 	log.Fatalf("error: %v", err)
		// }
		// cfg.Section("strategy").NewKey("preferred", "docker")
		// // cfg.Section("test").DeleteKey("name2")
		// // if len(cfg.Section("test").Keys()) == 0 {
		// // 	cfg.DeleteSection("test")
		// // }
		// cfg.SaveToIndent("/home/nate/.holenconfig", "    ")
		// // fmt.Println(cfg.Section("test").Key("name").String())

		cfg, err := ini.LooseLoad("/etc/holenconfig", "/home/nate/.holenconfig", "holenconfig")
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		// fmt.Println(cfg)
		for _, section := range cfg.Sections() {
			if len(section.Keys()) > 0 {
				for _, key := range section.Keys() {
					fmt.Printf("%s.%s = %s\n", section.Name(), key.Name(), key.Value())
				}
			}
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
