package main

import (
	"flag"
	"strings"
)

type buildFlags struct {
	// The flags proper:
	pkgDef, outFilename, altAppKey string

	// The two logical parts of pkgDef:
	pkgDefFile, pkgDefVar string
}

func (f *buildFlags) Register() {
	flag.StringVar(&f.pkgDef,
		"pkg-def",
		"sandstorm-pkgdef.capnp:pkgdef",
		"The location from which to read the package definition, of the form\n"+
			"<def-file>:<name>. <def-file> is the name of the file to look in,\n"+
			"and <name> is the name of the constant defining the package\n"+
			"definition.",
	)
	flag.StringVar(&f.outFilename,
		"out", "",
		"File name of the resulting spk (default inferred from package metadata)",
	)
	flag.StringVar(&f.altAppKey,
		"appkey", "",
		"Sign the package with the specified app key, instead of the one\n"+
			"defined in the package definition. This can be useful if e.g.\n"+
			"you do not have access to the key with which the final app is\n"+
			"published.")
}

func (f *buildFlags) Parse() {
	flag.Parse()
	pkgDefParts := strings.SplitN(f.pkgDef, ":", 2)
	if len(pkgDefParts) != 2 {
		usageErr("-pkg-def's argument must be of the form <def-file>:<name>")
	}
	f.pkgDefFile = pkgDefParts[0]
	f.pkgDefVar = pkgDefParts[1]
}

func buildCmd() {

}
