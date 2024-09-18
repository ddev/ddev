#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# bats tests/ddev-websever-dev/imagemagick.bats

@test "ImageMagick can convert PNG files" {
  docker cp tests/ddev-webserver/testdata/imagemagick/logo.png ${CONTAINER_NAME}:/tmp/logo.png
  docker exec -t $CONTAINER_NAME convert /tmp/logo.png /tmp/logo_png.jpg
}

@test "ImageMagick can convert SVG files" {
  docker cp tests/ddev-webserver/testdata/imagemagick/logo.svg ${CONTAINER_NAME}:/tmp/logo.svg
  docker exec -t $CONTAINER_NAME convert /tmp/logo.svg /tmp/logo_svg.png
}

@test "ImageMagick can convert PDF files" {
  docker cp tests/ddev-webserver/testdata/imagemagick/ddev.pdf ${CONTAINER_NAME}:/tmp/ddev.pdf
  docker exec -t $CONTAINER_NAME convert /tmp/ddev.pdf /tmp/ddev_pdf.png
}
