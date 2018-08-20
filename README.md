`docker-spk` is a tool to develop sandstorm packages using Docker to
build the root filesystems.

It is a work in progress, but already supports converting docker images
to sandstorm packages (`.spk` files), and signing and populating them
with metadata based on `sandstorm-pkgdef.capnp`.

Note that:

* It is not possible to automatically convert an arbitrary Docker image
  and have it work; the filesystem must be constructed to behave
  correctly inside Sandstorm's sandbox environment.
* Docker is only used to build the root filesystem of the app.
  Accordingly, Dockerfile instructions like `CMD`, `EXPOSE`,
  `STOPSIGNAL`, etc, which do not modify the image's filesystem, have
  no effect on the app. For other forms of customization, edit
  `sandstorm-pkgdef.capnp`.

# Building

1. Install a Go toolchain and [dep][dep].
2. From the root of the repository, run:

```sh
dep ensure
go build
```

This will create an executable `./docker-spk`.

# Quick Start

First, generate a sandstorm-pkgdef.capnp in the current directory:

```
docker-spk init
```

The tool will automatically generate a keypair for your app, and save it
in your keyring (by default `~/.sandstorm-keyring`, but this can be
overridden with the `-keyring` flag).

Edit the file to match your app. In particular, you will want to change
the command used to launch the app, near the bottom of the file.

Then, get a Docker image to convert. You can use `docker save` to fetch
one from a running Docker daemon:

```
docker save my-image > my-image.tar
```

Finally, run `docker-spk pack` to convert the image:

```
docker-spk pack -imagefile my-image.tar
```

This will create a `my-image.spk` based on the docker image and the
information in `sandstorm-manifest.capnp`.

# Examples

The `examples/` directory contains some examples that may be useful in
seeing how to package apps with `docker-spk`.

# Reference

```
$ docker-spk -h
Usage: docker-spk ( init | pack ) <flags>
where <flags> =
  -keyring string
        Path to sandstorm keyring (default "~/.sandstorm-keyring")

$ docker-spk init -h
Usage of docker-spk init:
  -keyring string
        Path to sandstorm keyring (default "~/.sandstorm-keyring")

$ docker-spk pack -h
Usage of docker-spk pack:
  -imagefile string
        File containing Docker image to convert (output of "docker save")
  -keyring string
        Path to sandstorm keyring (default "~/.sandstorm-keyring")
  -out string
        File name of the resulting spk (default inferred from -imagefile)
  -pkg-def string
        The location from which to read the package definition, of the form
        <def-file>:<name>. <def-file> is the name of the file to look in,
        and <name> is the name of the constant defining the package
        definition. (default "sandstorm-pkgdef.capnp:pkgdef")
```

# License

Apache 2.0, see COPYING.

[dep]: https://github.com/golang/dep
