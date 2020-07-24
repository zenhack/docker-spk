package main

import (
	"crypto/rand"
	"encoding/binary"
	"flag"
	"os"

	"zenhack.net/go/sandstorm/exp/spk"
)

type PkgDefParams struct {
	SchemaId uint64
	AppId    string
}

func randU64() uint64 {
	// It would probably be fine to use math/rand.Uint64 for this, but
	// elsewhere randomly generating a key, and I(zenhack) am generally
	// nervous about mixing cryptographc and non-cryptographic rngs in
	// nearby code.
	var ret uint64
	if err := binary.Read(rand.Reader, binary.LittleEndian, &ret); err != nil {
		panic(err)
	}
	return ret
}

func initCmd() {
	flag.Parse()

	key, err := spk.GenerateKey(nil)
	chkfatal("Generating a key", err)

	appId, err := key.AppId()
	chkfatal("Getting public key", err)

	params := &PkgDefParams{
		AppId:    appId.String(),
		SchemaId: randU64() | (1 << 63),
	}

	chkfatal("Saving the app key", key.AddToFile(*keyringPath))

	pkgDefFile, err := os.Create("sandstorm-pkgdef.capnp")
	chkfatal("Creating sandstorm-pkgdef.capnp", err)
	chkfatal("Outputting sandstorm-pkgdef.capnp", PkgDefTemplate.Execute(pkgDefFile, params))
}
