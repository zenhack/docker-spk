package main

import (
	"archive/tar"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ulikunitz/xz"
	"zenhack.net/go/sandstorm/capnp/spk"
	"zombiezen.com/go/capnproto2"
)

var (
	imageName   = flag.String("image", "", "Docker image to convert")
	outFilename = flag.String("out", "package.spk", "where to save the spk")

	ErrNotADir = errors.New("Not a directory")
)

func dirname(name string) string {
	return filepath.Clean(filepath.Dir(name))
}

func basename(name string) string {
	return filepath.Clean(filepath.Base(name))
}

func chkfatal(context string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v", context, err)
		os.Exit(1)
	}
}

// Build a map of the number of files inside of each directory in the
// docker image. Later on, this enables us to allocate lists of the
// correct size in the capnproto message.
func getDirSizes(dockerImage io.ReadSeeker) map[string]int {
	dirSizes := map[string]int{}

	layerIt := iterLayers(dockerImage)
	for layerIt.Next() {
		tarIt := layerIt.Cur()
		for tarIt.Next() {
			hdr := tarIt.Cur()
			name := filepath.Clean(hdr.Name)
			parentName := dirname(name)
			switch hdr.Typeflag {
			case tar.TypeDir:
				// Make sure the dir is actually in the map. Will
				// set the initial count to 0 if not, otherwise
				// will leave it unchanged.
				dirSizes[name] = dirSizes[name]
				fallthrough
			case tar.TypeReg, tar.TypeSymlink:
				dirSizes[parentName]++
			}
		}
		chkfatal("tar file", tarIt.Err())
	}
	chkfatal("layer", layerIt.Err())
	return dirSizes
}

// Return whether the tar.Header's Typeflag field indicates a file type that
// we support inside of an archive -- symlinks, regular files, and directories.
func supportedTypeFlag(hdr *tar.Header) bool {
	flag := hdr.Typeflag
	return flag == tar.TypeDir ||
		flag == tar.TypeReg ||
		flag == tar.TypeSymlink
}

// Build an archive from the docker image, preferring allocation in `seg`
// (and definitely allocating in the same message). The resulting archive
// is an orphan inside the message; it must be attached somewhere for it
// to be reachable.
func buildArchive(dockerImage io.ReadSeeker, seg *capnp.Segment) (spk.Archive, error) {
	dirSizes := getDirSizes(dockerImage)
	for k, v := range dirSizes {
		fmt.Fprintf(os.Stderr, "%q = %d\n", k, v)
	}

	ret, err := spk.NewArchive(seg)
	if err != nil {
		return ret, err
	}

	_, err = dockerImage.Seek(0, 0)
	if err != nil {
		return ret, err
	}

	allFiles := map[string]spk.Archive_File{}

	var (
		nextChild func(name string) (spk.Archive_File, error)
		getParent func(name string) (spk.Archive_File, error)
	)
	nextChild = func(name string) (spk.Archive_File, error) {
		parent, err := getParent(name)
		dir, err := parent.Directory()
		if err != nil {
			return spk.Archive_File{}, err
		}
		parentName := dirname(name)
		child := dir.At(dir.Len() - dirSizes[parentName])
		err = child.SetName(basename(name))
		dirSizes[parentName]--
		return child, err
	}

	getParent = func(name string) (spk.Archive_File, error) {
		var err error
		parentName := dirname(name)
		ret, ok := allFiles[parentName]
		if !ok {
			ret, err = nextChild(parentName)
			if err != nil {
				return ret, err
			}
			_, err = ret.NewDirectory(int32(dirSizes[parentName]))
			if err != nil {
				return ret, err
			}
			allFiles[parentName] = ret
		}
		return ret, nil
	}

	// TODO: we don't actually use this node, just its children -- it would
	// be good to avoid it being in the message.
	root, err := spk.NewArchive_File(seg)
	if err != nil {
		return ret, err
	}
	rootFiles, err := root.NewDirectory(int32(dirSizes["."]))
	if err != nil {
		return ret, err
	}
	allFiles["."] = root

	layerIt := iterLayers(dockerImage)
	for layerIt.Next() {
		tarIt := layerIt.Cur()
		for tarIt.Next() {
			hdr := tarIt.Cur()
			if !supportedTypeFlag(hdr) {
				continue
			}
			name := filepath.Clean(hdr.Name)
			this, ok := allFiles[name]
			if ok {
				continue
			}
			this, err := nextChild(name)
			if err != nil {
				return ret, err
			}
			allFiles[name] = this
			switch hdr.Typeflag {
			case tar.TypeDir:
				_, err = this.NewDirectory(int32(dirSizes[name]))
			case tar.TypeSymlink:
				err = this.SetSymlink(hdr.Linkname)
			case tar.TypeReg:
				bytes, err := ioutil.ReadAll(tarIt.Reader())
				if err != nil {
					return ret, err
				}
				// We treat an executable bit for anyone as an
				// executable.
				if hdr.FileInfo().Mode().Perm()&0111 == 0 {
					err = this.SetRegular(bytes)
				} else {
					err = this.SetExecutable(bytes)
				}
			}
			if err != nil {
				return ret, err
			}
		}
	}
	if err != nil {
		return ret, err
	}
	err = ret.SetFiles(rootFiles)
	return ret, err
}

func archiveBytesFromFilename(filename string) []byte {
	file, err := os.Open(filename)
	chkfatal("opening image file", err)
	defer file.Close()
	archiveMsg, archiveSeg, err := capnp.NewMessage(capnp.SingleSegment([]byte{}))
	chkfatal("allocating a message", err)
	archive, err := buildArchive(file, archiveSeg)
	chkfatal("building the archive", err)
	err = archiveMsg.SetRoot(archive.Struct.ToPtr())
	chkfatal("setting root pointer", err)
	bytes, err := archiveMsg.Marshal()
	chkfatal("marshalling archive message", err)
	return bytes
}

func signatureMessage(archiveBytes []byte) []byte {
	sigMsg, sigSeg, err := capnp.NewMessage(capnp.SingleSegment([]byte{}))
	chkfatal("Allocating message for signature", err)
	_, err = spk.NewRootSignature(sigSeg)
	chkfatal("Allocating signature in message", err)
	bytes, err := sigMsg.Marshal()
	chkfatal("marshalling signature message", err)
	return bytes
}

func main() {
	flag.Parse()
	archiveBytes := archiveBytesFromFilename(*imageName)
	sigBytes := signatureMessage(archiveBytes)

	outFile, err := os.Create(*outFilename)
	chkfatal("opening output file", err)
	defer outFile.Close()

	_, err = outFile.Write(spk.MagicNumber)
	chkfatal("writing magic number", err)

	compressedOut, err := xz.NewWriter(outFile)
	chkfatal("creating compressed output", err)

	_, err = compressedOut.Write(sigBytes)
	chkfatal("Writing signature", err)

	_, err = compressedOut.Write(archiveBytes)
	chkfatal("Writing archive", err)

	chkfatal("Finalizing the compression", compressedOut.Close())
}
