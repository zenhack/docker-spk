package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
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
	bFlags := &buildFlags{}
	bFlags.Register()
	bFlags.Parse()

	cmd := exec.Command("docker", "build", "-q", ".")
	cmd.Stderr = os.Stderr
	out, err := cmd.StdoutPipe()
	chkfatal("Creating pipe for docker build", err)
	chkfatal("Starting docker build", cmd.Start())
	r := bufio.NewScanner(out)

	image := ""
	for r.Scan() {
		line := r.Text()
		fmt.Println(line)
		image = line
	}
	chkfatal("Parsing output from docker build", err)
	chkfatal("Problem invoking docker build", cmd.Wait())
	if image == "" {
		fmt.Fprintln(os.Stderr,
			"Could not determine image id built by docker build.")
		os.Exit(1)
	}

	doPack(&packFlags{
		buildFlags: *bFlags,
		image:      image,
	})
}
