package main

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"io/ioutil"
	"path/filepath"
	"regexp"
)

type DockerManifestItem struct {
	Config   string
	RepoTags []string
	Layers   []string
}

type DockerImage struct {
	Layers   map[string]Tree
	Manifest []DockerManifestItem
}

// regular expression matching paths to layers inside the docker image.
var layerRegexp = regexp.MustCompile("^([0-9a-f]{64})/layer\\.tar$")

// Convert a tarball into a map from (full) paths to Files. Skips any file
// that is not a symlink, directory, or regular file.
func buildAbsFileMap(r *tar.Reader) (map[string]*File, error) {
	it := iterTar(r)
	ret := map[string]*File{}
	for it.Next() {
		hdr := it.Cur()
		switch hdr.Typeflag {
		case tar.TypeSymlink:
			ret[hdr.Name] = &File{
				target: hdr.Linkname,
			}
		case tar.TypeDir:
			ret[hdr.Name] = &File{
				kids: Tree{},
			}
		case tar.TypeReg:
			data, err := ioutil.ReadAll(r)
			if err != nil {
				return nil, err
			}
			ret[hdr.Name] = &File{
				data: data,
				// We treat an executable bit for anyone as an
				// executable.
				isExe: hdr.FileInfo().Mode().Perm()&0111 != 0,
			}
		}
	}
	return ret, it.Err()
}

// Insert the file at absPath into the .kids attribute of its parent directory.
// Adds the parent directory to abs if it does not already exist. An error is
// returned if abs already contains a file at absPath's parent that is not a
// directory.
func addRelFile(abs map[string]*File, absPath string) error {
	file := abs[absPath]
	if file == nil {
		// empty directory
		file = &File{
			kids: Tree{},
		}
		abs[absPath] = file
	}
	dirPath, relPath := filepath.Split(absPath)

	dirPath = filepath.Clean(dirPath)
	relPath = filepath.Clean(relPath)

	if dirPath != "." {
		err := addRelFile(abs, dirPath)
		if err != nil {
			return err
		}
	}
	dir := abs[dirPath]
	if dir.kids == nil {
		return errors.New("Conflict: non-directory has child nodes")
	}
	dir.kids[relPath] = file
	return nil
}

// Build a tree of files based on abs, which should be the return value from
// buildAbsFileMap.
func buildTree(abs map[string]*File) (Tree, error) {
	root := &File{
		kids: Tree{},
	}
	abs["."] = root
	for absPath, _ := range abs {
		err := addRelFile(abs, absPath)
		if err != nil {
			return nil, err
		}
	}
	return root.kids, nil
}

func readLayer(r *tar.Reader) (Tree, error) {
	absMap, err := buildAbsFileMap(r)
	if err != nil {
		return nil, err
	}
	return buildTree(absMap)
}

func readDockerImage(r *tar.Reader) (*DockerImage, error) {
	ret := &DockerImage{
		Layers:   map[string]Tree{},
		Manifest: []DockerManifestItem{},
	}
	it := iterTar(r)
	for it.Next() {
		cur := it.Cur()
		if cur.Name == "manifest.json" {
			ret := &DockerImage{}
			if err := json.NewDecoder(r).Decode(&ret.Manifest); err != nil {
				return nil, err
			}
		} else {
			matches := layerRegexp.FindSubmatch([]byte(cur.Name))
			if len(matches) < 2 {
				continue
			}
			layer, err := readLayer(tar.NewReader(r))
			if err != nil {
				return nil, err
			}
			ret.Layers[string(matches[1])] = layer
		}
	}
	return ret, it.Err()

}

func (di *DockerImage) toTree() (Tree, error) {
	tree := Tree{}
	for _, manifest := range di.Manifest {
		for _, layer := range manifest.Layers {
			tree.Merge(di.Layers[layer])
		}
	}
	return tree, nil
}
