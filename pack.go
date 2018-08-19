package main

import (
	"os"
	"os/exec"
	"strings"

	"github.com/ulikunitz/xz"
	"zenhack.net/go/sandstorm/capnp/spk"
	"zombiezen.com/go/capnproto2"
)

func packCmd() {
	if *imageName == "" {
		usageErr("Missing option: -imagefile")
	}

	pkgDefParts := strings.SplitN(*pkgDef, ":", 2)
	if len(pkgDefParts) != 2 {
		usageErr("-pkg-def's argument must be of the form <def-file>:<name>")
	}

	if *keyringPath == "" {
		// The user didn't specify a keyring; use the default.
		*keyringPath = os.Getenv("HOME") + "/.sandstorm-keyring"
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
