package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"

	"github.com/kardianos/osext"
)

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

func findSelfPath() (string, error) {
	var selfPath string
	var err error
	if selfPathOverride := os.Getenv("HLN_SELF_PATH_OVERRIDE"); len(selfPathOverride) > 0 {
		return selfPathOverride, nil
	} else {
		selfPath, err = osext.Executable()
		if err != nil {
			return "", err
		}
	}
	return selfPath, nil
}
func mergeMaps(m1, m2 map[interface{}]interface{}) map[interface{}]interface{} {
	for k := range m1 {
		if vv, ok := m2[k]; ok {
			if vv == nil {
				delete(m1, k)
			} else if _, typeOk := vv.(map[interface{}]interface{}); typeOk {
				m1[k] = mergeMaps(m1[k].(map[interface{}]interface{}), vv.(map[interface{}]interface{}))
			} else {
				m1[k] = vv
			}
			delete(m2, k)
		}
	}
	for k, v := range m2 {
		m1[k.(string)] = v
	}

	return m1
}

func hashFile(algo, filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var hash hash.Hash
	switch algo {
	case "md5":
		hash = md5.New()
	case "sha1":
		hash = sha1.New()
	case "sha256":
		hash = sha256.New()
	}

	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum([]byte(""))), nil
}
