package main

import (
	"encoding/base32"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
