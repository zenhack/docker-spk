package main

import (
	"archive/tar"
	"io"
)

// Get a reader for the specified file name inside the archive.
// The reader must not be used after modifying 'r'.
func getFile(r *tar.Reader, path string) (io.Reader, error) {
	hdr, err := r.Next()
	for err == nil {
		if hdr.Name == path {
			return r, nil
		}
		hdr, err = r.Next()
	}
	return nil, err
}

func iterTar(r *tar.Reader) *tarIterator {
	return &tarIterator{r: r}
}

type tarIterator struct {
	r   *tar.Reader
	cur *tar.Header
	err error
}

func (it *tarIterator) Reader() io.Reader {
	return it.r
}

func (it *tarIterator) Next() bool {
	if it.err != nil {
		return false
	}
	it.cur, it.err = it.r.Next()
	return it.err == nil
}

func (it *tarIterator) Err() error {
	if it.err == io.EOF {
		return nil
	}
	return it.err
}

func (it *tarIterator) Cur() *tar.Header {
	return it.cur
}
