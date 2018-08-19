package main

import (
	"archive/tar"
	"encoding/base32"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"zenhack.net/go/sandstorm/capnp/spk"
	"zombiezen.com/go/capnproto2"
)

var (
	// Sandstorm uses a custom base32 alphabet when displaying
	// app-ids/public keys.
	SandstormBase32Encoding = base32.NewEncoding("0123456789acdefghjkmnpqrstuvwxyz").
				WithPadding(base32.NoPadding)

	ErrNotADir = errors.New("Not a directory")
)

// Command line arguments:
var (
	imageName = flag.String("imagefile", "",
		"File containing Docker image to convert (output of \"docker save\")",
	)
	outFilename = flag.String("out", "",
		"File name of the resulting spk (default inferred from -imagefile)",
	)
	keyringPath = flag.String("keyring", "",
		"Path to sandstorm keyring (default ~/.sandstorm-keyring)",
	)

	pkgDef = flag.String(
		"pkg-def",
		"sandstorm-pkgdef.capnp:pkgdef",
		"The location from which to read the package definition, of the form\n"+
			"<def-file>:<name>. <def-file> is the name of the file to look in,\n"+
			"and <name> is the name of the constant defining the package\n"+
			"definition.",
	)
)

// wrapper around filepath.Dir that also canonicalizes the result.
func dirname(name string) string {
	return filepath.Clean(filepath.Dir(name))
}

// wrapper around filepath.Base that also canonicalizes the result.
func basename(name string) string {
	return filepath.Clean(filepath.Base(name))
}

// If the error is not nil, display an error message to the user based on
// `context` and `err`, and exit the with a failing status.
func chkfatal(context string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", context, err)
		os.Exit(1)
	}
}

// Build an archive from the docker image, preferring allocation in `seg`
// (and definitely allocating in the same message). The resulting archive
// is an orphan inside the message; it must be attached somewhere for it
// to be reachable.
func buildArchive(dockerImage io.Reader, seg *capnp.Segment, manifest, bridgeCfg []byte) (spk.Archive, error) {
	ret, err := spk.NewArchive(seg)
	if err != nil {
		return ret, err
	}
	img, err := readDockerImage(tar.NewReader(dockerImage))
	if err != nil {
		return ret, err
	}
	tree, err := img.toTree()
	if err != nil {
		return ret, err
	}
	tree["sandstorm-manifest"] = &File{data: manifest}
	tree["sandstorm-http-bridge-config"] = &File{data: bridgeCfg}
	err = tree.ToArchive(ret)
	return ret, err
}

// Read in the docker image located at filename, and return the raw bytes of a
// capnproto message with an equivalent Archive as its root. The second argument
// is the raw bytes of the file "sandstorm-manifest", which will be added to the
// archive.
func archiveBytesFromFilename(filename string, manifestBytes, bridgeCfgBytes []byte) []byte {
	file, err := os.Open(filename)
	chkfatal("opening image file", err)
	defer file.Close()
	archiveMsg, archiveSeg, err := capnp.NewMessage(capnp.SingleSegment([]byte{}))
	chkfatal("allocating a message", err)
	archive, err := buildArchive(file, archiveSeg, manifestBytes, bridgeCfgBytes)
	chkfatal("building the archive", err)
	err = archiveMsg.SetRoot(archive.Struct.ToPtr())
	chkfatal("setting root pointer", err)
	bytes, err := archiveMsg.Marshal()
	chkfatal("marshalling archive message", err)
	return bytes
}

// Report a usage error to the user. Displays the string `info` and the
// documentation for the command line arguments, and exits with a failing
// status.
func usageErr(info string) {
	fmt.Fprintln(os.Stderr, info)
	fmt.Fprintln(os.Stderr)
	flag.Usage()
	os.Exit(1)
}

func main() {
	flag.Parse()
	packCmd()
}
