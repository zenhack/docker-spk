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

Then, get a Docker image to convert. Invoke:

```
docker-spk pack -image <image-name>
```

...to use the image `<image-name>`, fetched from a running Docker
daemon.

This will create an `.spk` file, with the name derived from the app name
and version defined in `sandstorm-manifest.capnp`.

You can also use `docker save` to fetch the image manually and specify
the file name via `-imagefile`:

```
docker save my-image > my-image.tar
docker-spk pack -imagefile my-image.tar
```

# Examples

The `examples/` directory contains some examples that may be useful in
seeing how to package apps with `docker-spk`.

# Reference


See `docker-spk -h`.

# License

Apache 2.0, see COPYING.

[dep]: https://github.com/golang/dep
