# This Docker file is intended to generate a fully-reproducible executable
# of docker-spk.
#
# Build it with:
#
#    docker build .
#
# If all goes well, the last line of the output should say:
#
# Successfully build $hash
#
# Where $hash is some hash identifying the image. You can then
# extract the executable from the image by running:
#
#    docker run $hash > docker-spk
#
# (substituting the actual hash for $hash, of course). Before
# running it, you will have to mark it as executable:
#
#    chmod +x docker-spk
#
# The executable built from a source tree with a given tag should
# be bit-for-bit identical with the corresponding pre-compiled
# binary on the releases page:
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
RUN CGO_ENABLED=0 go build -ldflags '-w -s'
CMD cat /tmp/build-dir/docker-spk
