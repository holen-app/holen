package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeMaps(t *testing.T) {
	assert := assert.New(t)

	var mergeMapsTests = []struct {
		desc   string
		map1   map[interface{}]interface{}
		map2   map[interface{}]interface{}
		result map[interface{}]interface{}
	}{
		{
			"Simple maps",
			map[interface{}]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			map[interface{}]interface{}{
				"key1": "value2",
				"key3": "value3",
			},
			map[interface{}]interface{}{
				"key1": "value2",
				"key2": "value2",
				"key3": "value3",
			},
		},
		{
			"Nested matching maps",
			map[interface{}]interface{}{
				"key1": "value1",
				"key2": map[interface{}]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
			},
			map[interface{}]interface{}{
				"key1": "value1",
				"key2": map[interface{}]interface{}{
					"key4": "value4",
					"key5": "value5",
				},
				"key3": "value3",
			},
			map[interface{}]interface{}{
				"key1": "value1",
				"key2": map[interface{}]interface{}{
					"key1": "value1",
					"key2": "value2",
					"key4": "value4",
					"key5": "value5",
				},
				"key3": "value3",
			},
		},
		{
			"Nested maps, new key creates",
			map[interface{}]interface{}{
				"key1": "value1",
			},
			map[interface{}]interface{}{
				"key1": "value1",
				"key2": map[interface{}]interface{}{
					"key4": "value4",
					"key5": "value5",
				},
				"key3": "value3",
			},
			map[interface{}]interface{}{
				"key1": "value1",
				"key2": map[interface{}]interface{}{
					"key4": "value4",
					"key5": "value5",
				},
				"key3": "value3",
			},
		},
		{
			"Nested maps, nil value deletes",
			map[interface{}]interface{}{
				"key1": "value1",
				"key2": map[interface{}]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
			},
			map[interface{}]interface{}{
				"key1": "value1",
				"key2": nil,
				"key3": "value3",
			},
			map[interface{}]interface{}{
				"key1": "value1",
				"key3": "value3",
			},
		},
		{
			"Nested maps, nil value deletes",
			map[interface{}]interface{}{
				"key1": "value1",
				"key2": map[interface{}]interface{}{
					"key2": map[interface{}]interface{}{
						"key1": "value1",
						"key3": "value3",
					},
				},
			},
			map[interface{}]interface{}{
				"key1": "value2",
				"key2": map[interface{}]interface{}{
					"key2": map[interface{}]interface{}{
						"key1": "value2",
						"key4": "value4",
					},
				},
			},
			map[interface{}]interface{}{
				"key1": "value2",
				"key2": map[interface{}]interface{}{
					"key2": map[interface{}]interface{}{
						"key1": "value2",
						"key3": "value3",
						"key4": "value4",
					},
				},
			},
		},
		{
			"OS Arch test",
			map[interface{}]interface{}{
				"os_arch": map[interface{}]interface{}{
					"darwin_amd64": map[interface{}]interface{}{
						"ext":    "OSX64",
						"md5sum": "123abc",
					},
					"linux_amd64": map[interface{}]interface{}{
						"ext":    "Lin64",
						"md5sum": "456def",
					},
				},
			},
			map[interface{}]interface{}{
				"os_arch": map[interface{}]interface{}{
					"darwin_amd64": map[interface{}]interface{}{
						"md5sum": "abc123",
					},
					"linux_amd64": map[interface{}]interface{}{
						"md5sum": "def456",
					},
				},
			},
			map[interface{}]interface{}{
				"os_arch": map[interface{}]interface{}{
					"darwin_amd64": map[interface{}]interface{}{
						"ext":    "OSX64",
						"md5sum": "abc123",
					},
					"linux_amd64": map[interface{}]interface{}{
						"ext":    "Lin64",
						"md5sum": "def456",
					},
				},
			},
		},
	}

	for _, test := range mergeMapsTests {

		assert.Equal(mergeMaps(test.map1, test.map2), test.result)
	}

}

func TestHashFile(t *testing.T) {
	assert := assert.New(t)

	tempdir, _ := ioutil.TempDir("", "hash")
	defer os.RemoveAll(tempdir)
	filePath := path.Join(tempdir, "testfile")
	assert.Nil(ioutil.WriteFile(filePath, []byte("test contents\n"), 0755))

	var err error
	var sum string

	var hashTests = []struct {
		algo string
		hash string
	}{
		{"md5", "1b3c032e3e4eaad23401e1568879f150"},
		{"sha1", "40b44f15b4b6690a90792137a03d57c4d2918271"},
		{"sha256", "15721d5068de16cf4eba8d0fe6a563bb177333405323b479dcf5986da440c081"},
	}

	for _, test := range hashTests {
		sum, err = hashFile(test.algo, filePath)
		assert.Equal(test.hash, sum)
		assert.Nil(err)
	}

}
