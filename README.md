`docker2spk` is a simple tool to convert Docker images to sandstorm
packages (`.spk` files).

Note that an arbitrary Docker image will not work; the filesystem must
still be constructed to behave correctly inside Sandstorm's sandbox
environment.

# Building

1. Install a Go toolchain and [dep][dep].
2. From the root of the repository, run:

```sh
dep ensure
go build
```

This will create an executable `./docker2spk`.

# Quick Start

First, create a sandstorm-pkgdef.capnp in the current directory.
Generating this automatically is planned, but for now you can create
this with spk init (or vagrant-spk init).

Then, get a Docker image to convert. You can use `docker save` to fetch
one from a running Docker daemon:

```
docker save my-image > my-image.tar
```

Finally, run `docker2spk` to convert the image:

```
docker2spk -imagefile my-image.tar
```

This will create a `my-image.spk` based on the docker image and the
information in `sandstorm-manifest.capnp`.

```sh
# Fetch an image from the docker daemon:
docker save my-image > my-image.tar
# convert it to an `.spk`. $app_id must be one of the keys
# output by `spk listkeys`:
docker2spk my-image.tar -appid $app_id
```

# Reference

```
$ docker2spk -h
Usage of docker2spk:
  -imagefile string
        File containing Docker image to convert (output of "docker save")
  -keyring string
        Path to sandstorm keyring (default ~/.sandstorm-keyring)
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
