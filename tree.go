package main

import (
	"sort"
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

// Marshal the contents of a directory into an archive. `dest` must
// already have the correct length.
func insertDir(dest spk.Archive_File_List, t Tree) error {

	// For the sake of reproducable builds, we sort the keys.
	keys := make([]string, 0, len(t))
	for k, _ := range t {
		keys = append(keys, k)
	}
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
