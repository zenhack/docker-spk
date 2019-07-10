package main

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"io/ioutil"
	slashpath "path"
	"regexp"
)

// An item in the json array in the docker image's manifest.json.
type DockerManifestItem struct {
	Config   string
	RepoTags []string
	Layers   []string
}

// Information we need about a docker image.
type DockerImage struct {
	// The decoded layers of the docker image. The keys are the paths to
	// the layers' tarballs within the image.
	Layers map[string]Tree

	// The contents of the docker image's manifest.json
	Manifest []DockerManifestItem
}

// regular expression matching paths to layers inside the docker image.
var layerRegexp = regexp.MustCompile("^[0-9a-f]{64}/layer\\.tar$")

// Convert a tarball into a map from (full) paths to Files. Skips any file
// that is not a symlink, directory, or regular file.
//
// Note that the result is *not* a valid Tree; Trees are hierarchical,
// this is just a flat map from full paths to Files. Files which are
// directories do not have their contents populated.
func buildAbsFileMap(r *tar.Reader) (map[string]*File, error) {
	it := iterTar(r)
	ret := map[string]*File{}
	for it.Next() {
		hdr := it.Cur()
		name := slashpath.Clean(hdr.Name)
		switch hdr.Typeflag {
		case tar.TypeSymlink:
			ret[name] = &File{
				target: hdr.Linkname,
			}
		case tar.TypeDir:
			ret[name] = &File{
				kids: Tree{},
			}
		case tar.TypeReg:
			data, err := ioutil.ReadAll(r)
			if err != nil {
				return nil, err
			}
			ret[name] = &File{
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
//
// `abs` should be the return value from buildAbsFIleMap
func addRelFile(abs map[string]*File, absPath string) error {
	if absPath == "." {
		// The root of the file system (see the documentation
		// for path.Clean). There is no parent directory, so
		// just return.
		return nil
	}
	file := abs[absPath]
	if file == nil {
		// empty directory
		file = &File{
			kids: Tree{},
		}
		abs[absPath] = file
	}
	dirPath, relPath := slashpath.Split(absPath)

	dirPath = slashpath.Clean(dirPath)
	relPath = slashpath.Clean(relPath)

	if dirPath != "." {
		// make sure all our ancestors are present.
		if err := addRelFile(abs, dirPath); err != nil {
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
	for absPath := range abs {
		err := addRelFile(abs, absPath)
		if err != nil {
			return nil, err
		}
	}
	return root.kids, nil
}

// Unmarshal a layer tarball from within a docker image into a Tree.
func readLayer(r *tar.Reader) (Tree, error) {
	absMap, err := buildAbsFileMap(r)
	if err != nil {
		return nil, err
	}
	return buildTree(absMap)
}

// Unmarshal a docker image from a tarball.
func readDockerImage(r *tar.Reader) (*DockerImage, error) {
	ret := &DockerImage{
		Layers:   map[string]Tree{},
		Manifest: []DockerManifestItem{},
	}
	it := iterTar(r)
	for it.Next() {
		cur := it.Cur()
		if cur.Name == "manifest.json" {
			if err := json.NewDecoder(r).Decode(&ret.Manifest); err != nil {
				return nil, err
			}
		} else {
			if !layerRegexp.Match([]byte(cur.Name)) {
				continue
			}
			layer, err := readLayer(tar.NewReader(r))
			if err != nil {
				return nil, err
			}
			ret.Layers[cur.Name] = layer
		}
	}
	return ret, it.Err()

}

// Convert the docker image into a tree for the entire filesystem (merging
// the individual layers).
func (di *DockerImage) toTree() (Tree, error) {
	tree := Tree{}
	for _, manifest := range di.Manifest {
		for _, layer := range manifest.Layers {
			tree.Merge(di.Layers[layer])
		}
		removeWhiteout(tree)
	}
	return tree, nil
}
