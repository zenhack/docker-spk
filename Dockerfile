# This Docker file is intended to generate fully-reproducible executables
# of docker-spk. We currently generate x86_64 binaries for MacOS and Linux.
#
# Build it with:
#
#    docker build .
#
# If all goes well, the last line of the output should say:
#
# Successfully built $hash
#
# Where $hash is some hash identifying the image. You can then
# extract the executables from the image by running:
#
#    docker run $hash | tar -zxvf -
#
# This will create a directory tree 'docker-spk-binaries' containing
# the executables.  The executables built from a source tree with
# a given tag should be bit-for-bit identical with the corresponding
# pre-compiled binaries on the releases page:
#
#    https://github.com/zenhack/docker-spk/releases
#
# If it is not, please open a bug:
#
#    https://github.com/zenhack/docker-spk/issues
#
FROM golang:1.11.1-stretch
RUN mkdir /tmp/build-dir
WORKDIR /tmp/build-dir
COPY . .
RUN ./build-release-binaries.sh
CMD cat /tmp/build-dir/docker-spk-binaries.tar.gz
