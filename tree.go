package main

import (
	"fmt"
	"zenhack.net/go/sandstorm/capnp/spk"
)

type Tree map[string]*File

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

func (t Tree) ToArchive(dest spk.Archive) error {
	files, err := dest.NewFiles(int32(len(t)))
	if err != nil {
		return err
	}
	return insertDir(files, t)

}

func insertDir(dest spk.Archive_File_List, t Tree) error {
	i := 0
	for k, v := range t {
		di := dest.At(i)
		if err := insertFile(di, k, v); err != nil {
			return err
		}
		i++
	}
	return nil
}

func insertFile(dest spk.Archive_File, name string, file *File) error {
	fmt.Println(name)
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
