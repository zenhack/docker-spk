package main

import (
	"io/ioutil"
	"os"
)

// Save the sandstorm schema files in a temporary directory, and return the path
// to that directory.
func saveSchemaFiles() (string, error) {
	path, err := ioutil.TempDir("", "docker-spk-tmp-schema")
	if err != nil {
		return "", err
	}

	err = os.Mkdir(path+"/sandstorm", 0700)
	if err != nil {
		deleteSchemaFiles(path)
		return "", err
	}
	for k, v := range CapnpFileMap {
		err = ioutil.WriteFile(path+"/sandstorm/"+k, []byte(v), 0600)
		if err != nil {
			deleteSchemaFiles(path)
			return "", err
		}
	}
	return path, nil
}

// Delete the temporary path, and the schema files underneath it (as created
// by saveSchemaFiles()`.
func deleteSchemaFiles(path string) {
	for k, _ := range CapnpFileMap {
		os.Remove(path + "/sandstorm/" + k)
	}
	os.Remove(path + "/sandstorm")
	os.Remove(path)
}
