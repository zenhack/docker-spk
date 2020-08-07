package main

import (
	"flag"

	"zenhack.net/go/sandstorm/exp/spk"
)

func initCmd() {
	flag.Parse()

	pkgdef, err := spk.NewApp()
	chkfatal("Generating app info", err)
	pkgdef.KeyringPath = *keyringPath
	pkgdef.PkgDefPath = "sandstorm-pkgdef.capnp"
	chkfatal("Emitting app scaffolding", pkgdef.Emit())
}
