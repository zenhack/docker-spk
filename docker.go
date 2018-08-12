package main

import (
	"archive/tar"
	"encoding/json"
	"io"
)

type DockerManifestItem struct {
	Config   string
	RepoTags []string
	Layers   []string
}

func getManifest(tarFile *tar.Reader) ([]DockerManifestItem, error) {
	manifestJson, err := getFile(tarFile, "manifest.json")
	if err != nil {
		return nil, err
	}
	ret := []DockerManifestItem{}
	err = json.NewDecoder(manifestJson).Decode(&ret)
	return ret, err
}

type layerIterator struct {
	tarFile io.ReadSeeker
	tarIt   *tarIterator
	layers  []string
	err     error
}

func (it *layerIterator) Next() bool {
	if len(it.layers) == 0 || it.err != nil {
		return false
	}
	_, it.err = it.tarFile.Seek(0, 0)
	if it.err != nil {
		return false
	}
	r, err := getFile(tar.NewReader(it.tarFile), it.layers[0])
	it.err = err
	if it.err != nil {
		return false
	}
	it.tarIt = iterTar(tar.NewReader(r))
	it.layers = it.layers[1:]
	return true
}

func (it *layerIterator) Cur() *tarIterator {
	return it.tarIt
}

func (it *layerIterator) Err() error {
	if it.err == io.EOF {
		return nil
	}
	return it.err
}

func iterLayers(dockerImage io.ReadSeeker) *layerIterator {
	manifest, err := getManifest(tar.NewReader(dockerImage))
	if err != nil {
		return &layerIterator{err: err}
	}
	it := &layerIterator{
		tarFile: dockerImage,
		tarIt:   nil,
		err:     nil,
	}
	for _, v := range manifest {
		it.layers = append(it.layers, v.Layers...)
	}
	return it
}
