`docker2spk` is a simple tool to convert Docker images to sandstorm
packages (`.spk` files).

Note that an arbitrary Docker image will not work; the filesystem must
still be constructed to behave according to sandstorm's expectations. In
particular, this includes construction of /sandstorm-manifest, which
contains all of the application metadata, and arranging for the
container to work inside of sandstorm's sandbox (rather than Docker).

# Building

1. Install a Go toolchain and [dep][dep].
2. From the root of the repository, run:

```sh
dep ensure
go build
```

This will create an executable `./docker2spk`.

# Usage

## Basic example

```sh
# Fetch an image from the docker daemon:
docker save my-image > my-image.tar
# convert it to an `.spk`. $app_id must be one of the keys
# output by `spk listkeys`:
docker2spk my-image.tar -appid $app_id
```

## Reference

```
$ docker2spk -h
Usage of docker2spk:
  -appid string
    	The app id to assign to the package. The private key for this must be available in your sandstorm keyring.
  -imagefile string
    	File containing Docker image to convert (output of "docker save")
  -keyring string
    	Path to sandstorm keyring (default ~/.sandstorm-keyring)
  -out string
    	File name of the resulting spk (default inferred from -image)
$ docker2spk -imagefile
```

# License

Apache 2.0, see COPYING.

[dep]: https://github.com/golang/dep
