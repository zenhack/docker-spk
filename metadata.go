package main

import (
	"bytes"
	"os"
	"os/exec"

	"zenhack.net/go/sandstorm/capnp/spk"
	"zombiezen.com/go/capnproto2"
)

type pkgMetadata struct {
	manifest, bridgeCfg  []byte
	appId, name, version string
}

func getPkgMetadata(pkgDefFile, pkgDefVar string) *pkgMetadata {
	// Read in the package definition from sandstorm-pkgdef.capnp. The
	// file will reference some of the .capnp files from Sandstorm, so
	// we output those to a temporary directory and add it to the include
	// path for the capnp command.
	tmpDir, err := saveSchemaFiles()
	chkfatal("Saving temporary schema files", err)
	cmd := exec.Command(
		"capnp", "eval", "--binary", "-I", tmpDir, pkgDefFile, pkgDefVar,
	)
	stderrBuf := &bytes.Buffer{}
	cmd.Stderr = stderrBuf
	pkgDefBytes, err := cmd.Output()
	deleteSchemaFiles(tmpDir)
	if err != nil {
		os.Stderr.Write(stderrBuf.Bytes())
	}
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

	appTitle, err := pkgManifest.AppTitle()
	chkfatal("Getting app title", err)

	nameText, err := appTitle.DefaultText()
	chkfatal("Getting app name", err)

	appMarketingVersion, err := pkgManifest.AppMarketingVersion()
	chkfatal("Getting appMarketingVersion", err)

	versionText, err := appMarketingVersion.DefaultText()
	chkfatal("Getting version text", err)

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

	return &pkgMetadata{
		manifest:  manifestBytes,
		bridgeCfg: bridgeCfgBytes,
		appId:     appIdStr,
		name:      nameText,
		version:   versionText,
	}
}
