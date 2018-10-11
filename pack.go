package main

import (
	"archive/tar"
	"flag"
	"io"
	"os"
	"os/exec"

	"github.com/ulikunitz/xz"
	"zenhack.net/go/sandstorm/capnp/spk"
	"zombiezen.com/go/capnproto2"
)

// Build an archive from the docker image, preferring allocation in `seg`
// (and definitely allocating in the same message). The resulting archive
// is an orphan inside the message; it must be attached somewhere for it
// to be reachable.
func buildArchive(dockerImage io.Reader, seg *capnp.Segment, manifest, bridgeCfg []byte) (spk.Archive, error) {
	ret, err := spk.NewArchive(seg)
	if err != nil {
		return ret, err
	}
	img, err := readDockerImage(tar.NewReader(dockerImage))
	if err != nil {
		return ret, err
	}
	tree, err := img.toTree()
	if err != nil {
		return ret, err
	}

	// Add sandstorm metadata to the package:
	tree["sandstorm-manifest"] = &File{data: manifest}
	tree["sandstorm-http-bridge-config"] = &File{data: bridgeCfg}

	// Replace /var with an empty directory, since this is supposed to be
	// per-grain storage (as opposed to shared app storage) anyway. This
	// can make images a bit smaller, since often stuff gets left there.
	// by the build process.
	//
	// Note that the directory still needs to exist, since otherwise
	// it never gets created.
	tree["var"] = &File{kids: Tree{}}

	err = tree.ToArchive(ret)
	return ret, err
}

// Read in the docker image located at filename, and return the raw bytes of a
// capnproto message with an equivalent Archive as its root. The second argument
// is the raw bytes of the file "sandstorm-manifest", which will be added to the
// archive.
func archiveBytesFromFilename(filename string, manifestBytes, bridgeCfgBytes []byte) []byte {
	file, err := os.Open(filename)
	chkfatal("opening image file", err)
	defer file.Close()
	return archiveBytesFromReader(file, manifestBytes, bridgeCfgBytes)
}

func archiveBytesFromDocker(image string, mainfestBytes, bridgeCfgBytes []byte) []byte {
	cmd := exec.Command("docker", "save", image)
	stdout, err := cmd.StdoutPipe()
	chkfatal("Getting standard output from docker save", err)
	defer stdout.Close()
	chkfatal("Starting docker save", cmd.Start())
	defer func() {
		chkfatal("Waiting for docker save", cmd.Wait())
	}()
	return archiveBytesFromReader(stdout, mainfestBytes, bridgeCfgBytes)
}

func archiveBytesFromReader(r io.Reader, manifestBytes, bridgeCfgBytes []byte) []byte {
	archiveMsg, archiveSeg, err := capnp.NewMessage(capnp.SingleSegment([]byte{}))
	chkfatal("allocating a message", err)
	archive, err := buildArchive(r, archiveSeg, manifestBytes, bridgeCfgBytes)
	chkfatal("building the archive", err)
	err = archiveMsg.SetRoot(archive.Struct.ToPtr())
	chkfatal("setting root pointer", err)
	bytes, err := archiveMsg.Marshal()
	chkfatal("marshalling archive message", err)
	return bytes
}

// Flags for the pack subcommand.
type packFlags struct {
	// flags shared with the build command:
	buildFlags

	// other flags:
	imageFile, image string
}

func (f *packFlags) Register() {
	f.buildFlags.Register()
	flag.StringVar(&f.imageFile,
		"imagefile", "",
		"File containing Docker image to convert (output of \"docker save\")",
	)
	flag.StringVar(&f.image,
		"image", "",
		"Name of the image to convert (fetched from the running docker daemon).",
	)
}

func (f *packFlags) Parse() {
	f.buildFlags.Parse()
	if f.imageFile == "" && f.image == "" {
		usageErr("Missing option: -image or -imagefile")
	}
	if f.imageFile != "" && f.image != "" {
		usageErr("Only one of -image or -imagefile may be specified.")
	}
}

func packCmd() {
	pFlags := &packFlags{}
	pFlags.Register()
	pFlags.Parse()
	doPack(pFlags)
}

func doPack(pFlags *packFlags) {
	metadata := getPkgMetadata(pFlags.pkgDefFile, pFlags.pkgDefVar)

	keyring, err := loadKeyring(*keyringPath)
	chkfatal("loading the sandstorm keyring", err)

	if pFlags.altAppKey != "" {
		// The user has requested we use a different key.
		metadata.appId = pFlags.altAppKey
	}

	appPubKey, err := SandstormBase32Encoding.DecodeString(metadata.appId)
	chkfatal("Parsing the app id", err)

	appKeyFile, err := keyring.GetKey(appPubKey)
	chkfatal("Fetching the app private key", err)

	var archiveBytes []byte
	if pFlags.imageFile != "" {
		archiveBytes = archiveBytesFromFilename(pFlags.imageFile, metadata.manifest, metadata.bridgeCfg)
	} else if pFlags.image != "" {
		archiveBytes = archiveBytesFromDocker(pFlags.image, metadata.manifest, metadata.bridgeCfg)
	} else {
		// pFlags.Parse() should have ruled this out.
		panic("impossible")
	}
	sigBytes := signatureMessage(appKeyFile, archiveBytes)

	if pFlags.outFilename == "" {
		// infer output file from app metadata:
		pFlags.outFilename = metadata.name + "-" + metadata.version + ".spk"
	}

	outFile, err := os.Create(pFlags.outFilename)
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
