package main

import (
	"archive/tar"
	"encoding/base32"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
	"zenhack.net/go/sandstorm/capnp/spk"
	"zombiezen.com/go/capnproto2"
)

var (
	// Sandstorm uses a custom base32 alphabet.
	SandstormBase32Encoding = base32.NewEncoding("0123456789acdefghjkmnpqrstuvwxyz").
				WithPadding(base32.NoPadding)

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
		fmt.Fprintf(os.Stderr, "%s: %v\n", context, err)
		os.Exit(1)
	}
}

// Build an archive from the docker image, preferring allocation in `seg`
// (and definitely allocating in the same message). The resulting archive
// is an orphan inside the message; it must be attached somewhere for it
// to be reachable.
func buildArchive(dockerImage io.Reader, seg *capnp.Segment, manifest []byte) (spk.Archive, error) {
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
	tree["sandstorm-manifest"] = &File{
		data: manifest,
	}
	err = tree.ToArchive(ret)
	return ret, err
}

// Read in the docker image located at filename, and return the raw bytes of a
// capnproto message with an equivalent Archive as its root. The second argument
// is the raw bytes of the file "sandstorm-manifest", which will be added to the
// archive.
func archiveBytesFromFilename(filename string, manifestBytes []byte) []byte {
	file, err := os.Open(filename)
	chkfatal("opening image file", err)
	defer file.Close()
	archiveMsg, archiveSeg, err := capnp.NewMessage(capnp.SingleSegment([]byte{}))
	chkfatal("allocating a message", err)
	archive, err := buildArchive(file, archiveSeg, manifestBytes)
	chkfatal("building the archive", err)
	err = archiveMsg.SetRoot(archive.Struct.ToPtr())
	chkfatal("setting root pointer", err)
	bytes, err := archiveMsg.Marshal()
	chkfatal("marshalling archive message", err)
	return bytes
}

func usageErr(info string) {
	fmt.Fprintln(os.Stderr, info)
	fmt.Fprintln(os.Stderr)
	flag.Usage()
	os.Exit(1)
}

func main() {
	flag.Parse()

	if *imageName == "" {
		usageErr("Missing option: -image")
	}

	pkgDefParts := strings.SplitN(*pkgDef, ":", 2)
	if len(pkgDefParts) != 2 {
		usageErr("-pkg-def's argument must be of the form <def-file>:<name>")
	}

	if *keyringPath == "" {
		// The user didn't specify a keyring; use the default.
		*keyringPath = os.Getenv("HOME") + "/.sandstorm-keyring"
	}

	tmpDir, err := saveSchemaFiles()
	chkfatal("Saving temporary schema files", err)
	pkgDefBytes, err := exec.Command(
		"capnp", "eval", "--binary", "-I", tmpDir, pkgDefParts[0], pkgDefParts[1],
	).Output()
	deleteSchemaFiles(tmpDir)
	chkfatal("Reading the package definition", err)

	pkgDefMsg, err := capnp.Unmarshal(pkgDefBytes)
	chkfatal("Parsing the package definition message", err)

	pkgDefVal, err := spk.ReadRootPackageDefinition(pkgDefMsg)
	chkfatal("Parsing the package definition message struct", err)

	pkgManifest, err := pkgDefVal.Manifest()
	chkfatal("Reading the package manifest", err)

	appIdStr, err := pkgDefVal.Id()
	chkfatal("Reading the package's app id", err)

	// We're being a bit lazy here; rather than copy the manifest to a
	// new message, we just make it the root of the existing message.
	err = pkgDefMsg.SetRoot(pkgManifest.Struct.ToPtr())
	chkfatal("Setting the root pointer in the manifest message", err)

	manifestBytes, err := pkgDefMsg.Marshal()
	chkfatal("Marshalling sandstorm-manifest", err)

	keyring, err := loadKeyring(*keyringPath)
	chkfatal("loading the sandstorm keyring", err)

	appPubKey, err := SandstormBase32Encoding.DecodeString(appIdStr)
	chkfatal("Parsing the app id", err)

	appKeyFile, err := keyring.GetKey(appPubKey)
	chkfatal("Fetching the app private key", err)

	archiveBytes := archiveBytesFromFilename(*imageName, manifestBytes)
	sigBytes := signatureMessage(appKeyFile, archiveBytes)

	if *outFilename == "" {
		// infer output file from input file.
		stem := *imageName
		if strings.HasSuffix(stem, ".tar") {
			stem = stem[:len(*imageName)-len(".tar")]
		}
		stem += ".spk"
		*outFilename = stem
	}

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
