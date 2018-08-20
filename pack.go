package main

import (
	"archive/tar"
	"flag"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/ulikunitz/xz"
	"zenhack.net/go/sandstorm/capnp/spk"
	"zombiezen.com/go/capnproto2"
)

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

	// Add sandstorm metadata to the package:
	tree["sandstorm-manifest"] = &File{data: manifest}
	tree["sandstorm-http-bridge-config"] = &File{data: bridgeCfg}

	// Remove /var, since sandstorm won't make it available anyway. This
	// can make images a bit smaller, since often stuff gets left there.
	delete(tree, "var")

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

func packCmd() {
	pkgDef := flag.String(
		"pkg-def",
		"sandstorm-pkgdef.capnp:pkgdef",
		"The location from which to read the package definition, of the form\n"+
			"<def-file>:<name>. <def-file> is the name of the file to look in,\n"+
			"and <name> is the name of the constant defining the package\n"+
			"definition.",
	)
	imageName := flag.String("imagefile", "",
		"File containing Docker image to convert (output of \"docker save\")",
	)
	outFilename := flag.String("out", "",
		"File name of the resulting spk (default inferred from -imagefile)",
	)
	altAppKey := flag.String("appkey", "",
		"Sign the package with the specified app key, instead of the one\n"+
			"defined in the package definition. This can be useful if e.g.\n"+
			"you do not have access to the key with which the final app is\n"+
			"published.")
	flag.Parse()

	if *imageName == "" {
		usageErr("Missing option: -imagefile")
	}

	pkgDefParts := strings.SplitN(*pkgDef, ":", 2)
	if len(pkgDefParts) != 2 {
		usageErr("-pkg-def's argument must be of the form <def-file>:<name>")
	}

	// Read in the package definition from sandstorm-pkgdef.capnp. The
	// file will reference some of the .capnp files from Sandstorm, so
	// we output those to a temporary directory and add it to the include
	// path for the capnp command.
	tmpDir, err := saveSchemaFiles()
	chkfatal("Saving temporary schema files", err)
	pkgDefBytes, err := exec.Command(
		"capnp", "eval", "--binary", "-I", tmpDir, pkgDefParts[0], pkgDefParts[1],
	).Output()
	deleteSchemaFiles(tmpDir)
	chkfatal("Reading the package definition", err)

	// There are two pieces of information we want out of the package definition:
	//
	// 1. The app id, which tells us which key to use to sign the package.
	// 2. The manifest, which we embed in the package's archive.

	pkgDefMsg, err := capnp.Unmarshal(pkgDefBytes)
	chkfatal("Parsing the package definition message", err)

	pkgDefVal, err := spk.ReadRootPackageDefinition(pkgDefMsg)
	chkfatal("Parsing the package definition message struct", err)

	pkgManifest, err := pkgDefVal.Manifest()
	chkfatal("Reading the package manifest", err)

	appIdStr, err := pkgDefVal.Id()
	chkfatal("Reading the package's app id", err)

	bridgeCfg, err := pkgDefVal.BridgeConfig()
	chkfatal("Reading the bridge config", err)

	// Generate the contents of the file /sandstorm-manifest
	manifestMsg, manifestSeg, err := capnp.NewMessage(capnp.SingleSegment([]byte{}))
	chkfatal("Allocating a message for the manifest", err)
	rootManifest, err := capnp.NewRootStruct(manifestSeg, pkgManifest.Struct.Size())
	chkfatal("Allocating the root object for the manifest", err)
	chkfatal("Copying manifest", rootManifest.CopyFrom(pkgManifest.Struct))
	manifestBytes, err := manifestMsg.Marshal()
	chkfatal("Marshalling sandstorm-manifest", err)

	// Generate the contents of the file /sandstorm-http-bridge-config
	bridgeCfgMsg, bridgeCfgSeg, err := capnp.NewMessage(capnp.SingleSegment([]byte{}))
	chkfatal("Allocating a message for the bridge config", err)
	rootCfg, err := capnp.NewRootStruct(bridgeCfgSeg, bridgeCfg.Struct.Size())
	chkfatal("Allocating the root object for the bridge config", err)
	chkfatal("Copying bridgeCfg", rootCfg.CopyFrom(bridgeCfg.Struct))
	bridgeCfgBytes, err := bridgeCfgMsg.Marshal()
	chkfatal("Marshalling sandstorm-bridgeCfg", err)

	keyring, err := loadKeyring(*keyringPath)
	chkfatal("loading the sandstorm keyring", err)

	if *altAppKey != "" {
		// The user has requested we use a different key.
		appIdStr = *altAppKey
	}

	appPubKey, err := SandstormBase32Encoding.DecodeString(appIdStr)
	chkfatal("Parsing the app id", err)

	appKeyFile, err := keyring.GetKey(appPubKey)
	chkfatal("Fetching the app private key", err)

	archiveBytes := archiveBytesFromFilename(*imageName, manifestBytes, bridgeCfgBytes)
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
