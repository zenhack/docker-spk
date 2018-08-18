package main

import (
	"bytes"
	"crypto/sha512"
	"errors"
	"io"
	"os"

	"golang.org/x/crypto/nacl/sign"

	"zenhack.net/go/sandstorm/capnp/spk"
	"zombiezen.com/go/capnproto2"
)

var (
	ErrKeyNotFound = errors.New("Key not found in keyring")
)

// The sandstorm keyring, typically stored at ~/.sandstorm-keyring.
type Keyring []spk.KeyFile

// Get a key from the keyring.
func (k Keyring) GetKey(targetPubKey []byte) (spk.KeyFile, error) {
	// simple linear search.
	for _, keyFile := range k {
		pubKey, err := keyFile.PublicKey()
		if err != nil {
			return spk.KeyFile{}, err
		}
		if bytes.Compare(targetPubKey, pubKey) == 0 {
			return keyFile, nil
		}
	}
	return spk.KeyFile{}, ErrKeyNotFound
}

// Load the sandstorm keyring from a named file.
func loadKeyring(filename string) (Keyring, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ret := Keyring{}
	dec := capnp.NewDecoder(file)

	for {
		msg, err := dec.Decode()
		switch err {
		case nil:
			keyFile, err := spk.ReadRootKeyFile(msg)
			if err != nil {
				return nil, err
			}
			ret = append(ret, keyFile)
		case io.EOF:
			return ret, nil
		default:
			return nil, err
		}
	}
}

// Compute the signature of a package, given the raw bytes of the archive
// message. Returns the raw bytes of the signature message. Calls chkfatal
// if any error occurs.
func signatureMessage(key spk.KeyFile, archiveBytes []byte) []byte {
	pubKey, err := key.PublicKey()
	chkfatal("Accessing public key", err)

	privKey, err := key.PrivateKey()
	chkfatal("Accessing private key", err)

	// the go nacl library expects an array, not a slice:
	var naclPrivKey [64]byte
	copy(naclPrivKey[:], privKey)

	sigMsg, sigSeg, err := capnp.NewMessage(capnp.SingleSegment([]byte{}))
	chkfatal("Allocating message for signature", err)
	sig, err := spk.NewRootSignature(sigSeg)
	chkfatal("Allocating signature in message", err)

	hash := sha512.Sum512(archiveBytes)
	sigbytes := sign.Sign([]byte{}, hash[:], &naclPrivKey)

	chkfatal("Adding public key to signature message",
		sig.SetPublicKey(pubKey))
	chkfatal("Adding signature to message",
		sig.SetSignature(sigbytes))

	bytes, err := sigMsg.Marshal()
	chkfatal("marshalling signature message", err)
	return bytes
}
