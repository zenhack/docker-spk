package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"zenhack.net/go/sandstorm/capnp/spk"
)

// A set of files inside of a directory.
type Tree map[string]*File

// A node in a file system. We store the docker file system in this format
// between reading the image and converting it to an spk.Archive.
type File struct {
	// If this is a directory, the contents of the directory by relative
	// path. Otherwise, this will be nil.
	kids Tree

	// If this is a regular file, the bytes of the file (otherwise nil)
	data []byte

	// Whether this is an executable. Only meaningful for regular files.
	isExe bool

	// If this is a symlink, the target of the symlink. Otherwise "".
	target string
}

// Return whether the file is a directory.
func (f *File) isDir() bool {
	return f.kids != nil
}

// Merge the argument into this tree. Directories are merged recursively.
// Otherwise, files in the argument take precedence. The argument should
// not be used afterwards.
func (t Tree) Merge(other Tree) {
	for k, vOther := range other {
		vThis, ok := t[k]
		if ok && vThis.isDir() && vOther.isDir() {
			vThis.kids.Merge(vOther.kids)
		} else {
			t[k] = vOther
		}
	}
}

// Convert the tree into an sandstorm pacakge archive.
func (t Tree) ToArchive(dest spk.Archive) error {
	files, err := dest.NewFiles(int32(len(t)))
	if err != nil {
		return err
	}
	return insertDir(files, t)

}

func getKeys(t Tree) []string {
	keys := make([]string, 0, len(t))
	for k, _ := range t {
		keys = append(keys, k)
	}
	return keys
}

// Marshal the contents of a directory into an archive. `dest` must
// already have the correct length.
func insertDir(dest spk.Archive_File_List, t Tree) error {

	// For the sake of reproducable builds, we sort the keys.
	keys := getKeys(t)
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	for i, k := range keys {
		if err := insertFile(dest.At(i), k, t[k]); err != nil {
			return err
		}
	}
	return nil
}

// Marshal a single file into an archive.
func insertFile(dest spk.Archive_File, name string, file *File) error {
	err := dest.SetName(name)
	if err != nil {
		return err
	}
	switch {
	case file.isDir():
		kids, err := dest.NewDirectory(int32(len(file.kids)))
		if err == nil {
			err = insertDir(kids, file.kids)
		}
	case file.data != nil && file.isExe:
		err = dest.SetExecutable(file.data)
	case file.data != nil && !file.isExe:
		err = dest.SetRegular(file.data)
	default:
		err = dest.SetSymlink(file.target)
	}
	return err
}

// Look for whiteout files in the tree, and remove both them and the files
// to which they point. See:
//
// https://github.com/moby/moby/blob/master/image/spec/v1.md
func removeWhiteout(t Tree) {
	keys := getKeys(t)
	for _, name := range keys {
		if strings.HasPrefix(name, ".wh.") {
			delete(t, name)
			delete(t, name[len(".wh."):])
		}
	}
	for _, file := range t {
		if file.isDir() {
			removeWhiteout(file.kids)
		}
	}
}

// Read a File from the local directory at `root`.
func readLocalFS(root string) (*File, error) {
	fi, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}
	mode := fi.Mode()
	typ := mode & os.ModeType
	switch typ {
	case os.ModeDir:
		t, err := readLocalFSTree(root)
		return &File{kids: t}, err
	case os.ModeSymlink:
		target, err := os.Readlink(root)
		return &File{target: target}, err
	case 0:
		// regular file
		data, err := ioutil.ReadFile(root)
		return &File{
			data:  data,
			isExe: mode&0111 != 0,
		}, err
	default:
		return nil, fmt.Errorf("%q: unsupported file type: '%v'", root, typ)
	}
}

// Read the local directory at `root` into a tree.
func readLocalFSTree(root string) (Tree, error) {
	f, err := os.Open(root)
	if err != nil {
		return nil, err
	}
	fis, err := f.Readdir(0)
	f.Close()
	if err != nil {
		return nil, err
	}
	ret := make(Tree, len(fis))
	for _, fi := range fis {
		node, err := readLocalFS(root + "/" + fi.Name())
		if err != nil {
			return nil, err
		}
		ret[fi.Name()] = node
	}
	return ret, nil
}
