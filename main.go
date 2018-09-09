package main

import (
	"encoding/base32"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	keyringPath = flag.String(
		"keyring",
		os.Getenv("HOME")+"/.sandstorm-keyring",
		"Path to sandstorm keyring",
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
	subCommands := map[string]func(){
		"pack": packCmd,
		"init": initCmd,
	}
	flag.Usage = func() {
		keys := []string{}
		for k, _ := range subCommands {
			keys = append(keys, k)
		}
		fmt.Fprintf(os.Stderr,
			"Usage: %s ( %s ) <flags>\n"+
				"where <flags> =\n",
			os.Args[0],
			strings.Join(keys, " | "),
		)
		flag.PrintDefaults()
		os.Exit(1)
	}
	if len(os.Args) < 2 {
		flag.Usage()
	}
	cmd := os.Args[1]
	fn, ok := subCommands[cmd]
	if ok {
		arg0 := os.Args[0]
		// We have to chop of the subcommand or the parser gets confused later:
		os.Args = os.Args[1:]
		flag.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage of %s %s:\n", arg0, cmd)
			flag.PrintDefaults()
		}
		fn()
		return
	}
	switch cmd {
	case "-h", "-help", "--help", "help":
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", os.Args[1])
	}
	flag.Usage()
}
