package main

import (
	"crypto/rand"
	"encoding/binary"
	"flag"
	"os"
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

	keyFile, err := GenerateKey()
	chkfatal("Generating a key", err)

	pubKey, err := keyFile.PublicKey()
	chkfatal("Getting public key", err)

	params := &PkgDefParams{
		AppId:    SandstormBase32Encoding.EncodeToString(pubKey),
		SchemaId: randU64() | (1 << 63),
	}

	keyBytes, err := keyFile.Struct.Message().Marshal()
	chkfatal("Serializing app key", err)

	keyringFile, err := os.OpenFile(*keyringPath, os.O_WRONLY|os.O_APPEND, 0600)
	chkfatal("Opening the keyring for writing", err)
	defer keyringFile.Close()

	_, err = keyringFile.Write(keyBytes)
	chkfatal("Saving the app key", err)

	pkgDefFile, err := os.Create("sandstorm-pkgdef.capnp")
	chkfatal("Creating sandstorm-pkgdef.capnp", err)
	chkfatal("Outputting sandstorm-pkgdef.capnp", PkgDefTemplate.Execute(pkgDefFile, params))
}
