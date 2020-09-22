sudo apt-get install docker-ce-cli binfmt-support qemu-user-static

BUILDX_BINARY_URL="https://github.com/docker/buildx/releases/download/v0.4.1/buildx-v0.4.1.linux-amd64"

curl --output docker-buildx \
    --silent --show-error --location --fail --retry 3 \
    "$BUILDX_BINARY_URL"

mkdir -p ~/.docker/cli-plugins
mv docker-buildx ~/.docker/cli-plugins
chmod a+x ~/.docker/cli-plugins/docker-buildx
