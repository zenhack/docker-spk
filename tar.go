package main

import (
	"archive/tar"
	"io"
)

// Get a tarIterator for the reader.
func iterTar(r *tar.Reader) *tarIterator {
	return &tarIterator{r: r}
}

// A tarIterator makes it more convienent to iterate over the contents of
// a tar archive.
//
// Basic usage:
//
// it := iterTar(tarReader)
//
// for it.Next() {
// 	hdr := it.Cur()
//      // do stuff with hdr
// }
//
// if it.Err() != nil {
// 	// handle errors
// }
type tarIterator struct {
	r   *tar.Reader
	cur *tar.Header
	err error
}

// Return a reader which will read from the current file.
func (it *tarIterator) Reader() io.Reader {
	return it.r
}

// Move to the next file in the archive. If there are no more files, or an
// error occurs, return false. it.Err() may be used to distinguish these
// cases.
func (it *tarIterator) Next() bool {
	if it.err != nil {
		return false
	}
	it.cur, it.err = it.r.Next()
	return it.err == nil
}

// Return any error that has occurred when processing the file. If the
// error was io.EOF, nil will be returned instead.
func (it *tarIterator) Err() error {
	if it.err == io.EOF {
		return nil
	}
	return it.err
}

// Return the header for the current file.
func (it *tarIterator) Cur() *tar.Header {
	return it.cur
}
