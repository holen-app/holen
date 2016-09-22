package main

import "strings"

type NameVer struct {
	Name    string
	Version string
}

func ParseName(utility string) NameVer {
	parts := strings.Split(utility, "--")
	version := ""
	if len(parts) > 1 {
		version = parts[1]
	}

	return NameVer{parts[0], version}
}

func mergeMaps(m1, m2 map[interface{}]interface{}) map[interface{}]interface{} {
	for k := range m1 {
		if vv, ok := m2[k]; ok {
			m1[k] = vv
			delete(m2, k)
		}
	}
	for k, v := range m2 {
		m1[k.(string)] = v
	}

	return m1
}
