#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# bats tests/ddev-webserver/imagemagick.bats

@test "ImageMagick can convert PNG files" {
  docker cp tests/ddev-webserver/testdata/imagemagick/logo.png ${CONTAINER_NAME}:/tmp/logo.png
  docker exec -t $CONTAINER_NAME magick /tmp/logo.png /tmp/logo_png.jpg
  docker exec -t $CONTAINER_NAME bash -c "magick -list format | grep -i 'Portable Network Graphics' | grep 'rw-'"
}

@test "ImageMagick can convert SVG files" {
  docker cp tests/ddev-webserver/testdata/imagemagick/logo.svg ${CONTAINER_NAME}:/tmp/logo.svg
  docker exec -t $CONTAINER_NAME magick /tmp/logo.svg /tmp/logo_svg.png
  docker exec -t $CONTAINER_NAME bash -c "magick -list format | grep -i 'Scalable Vector Graphics' | grep 'rw+'"
}

@test "ImageMagick can convert PDF files" {
  docker cp tests/ddev-webserver/testdata/imagemagick/ddev.pdf ${CONTAINER_NAME}:/tmp/ddev.pdf
  docker exec -t $CONTAINER_NAME magick /tmp/ddev.pdf /tmp/ddev_pdf.png
  docker exec -t $CONTAINER_NAME bash -c "magick -list format | grep -i 'Portable Document Format' | grep 'rw+'"
}

@test "ImageMagick can convert AVIF files" {
  docker cp tests/ddev-webserver/testdata/imagemagick/logo.avif ${CONTAINER_NAME}:/tmp/logo.avif
  docker exec -t $CONTAINER_NAME magick /tmp/logo.avif /tmp/logo_avif.png
  docker exec -t $CONTAINER_NAME bash -c "magick -list format | grep -i 'AV1 Image File Format' | grep 'rw+'"
}

@test "ImageMagick can convert HEIC files" {
  docker cp tests/ddev-webserver/testdata/imagemagick/logo.heic ${CONTAINER_NAME}:/tmp/logo.heic
  docker exec -t $CONTAINER_NAME magick /tmp/logo.heic /tmp/logo_heic.png
  docker exec -t $CONTAINER_NAME bash -c "magick -list format | grep -i 'High Efficiency Image Format' | grep 'rw+'"
}
