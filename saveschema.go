package main

import (
	"io/ioutil"
	"os"
)

func deleteSchemaFiles(path string) {
	for k, _ := range CapnpFileMap {
		os.Remove(path + "/sandstorm/" + k)
	}
	os.Remove(path + "/sandstorm")
	os.Remove(path)
}

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
