#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# bats tests/ddev-webserver/imagemagick.bats

setup() {
  load setup.sh
}

@test "ImageMagick can convert PNG files" {
  run docker cp tests/ddev-webserver/testdata/imagemagick/logo.png "${CONTAINER_NAME}:/tmp/logo.png"
  assert_success
  run docker exec -t $CONTAINER_NAME magick /tmp/logo.png /tmp/logo_png.jpg
  assert_success
  run docker exec -t $CONTAINER_NAME bash -c "magick -list format | grep -i 'Portable Network Graphics'"
  assert_success
  assert_output --partial "rw-"
}

@test "ImageMagick can convert SVG files" {
  run docker cp tests/ddev-webserver/testdata/imagemagick/logo.svg "${CONTAINER_NAME}:/tmp/logo.svg"
  assert_success
  run docker exec -t $CONTAINER_NAME magick /tmp/logo.svg /tmp/logo_svg.png
  assert_success
  run docker exec -t $CONTAINER_NAME bash -c "magick -list format | grep -i 'Scalable Vector Graphics'"
  assert_success
  assert_output --partial "rw+"
}

@test "ImageMagick can convert PDF files" {
  run docker cp tests/ddev-webserver/testdata/imagemagick/ddev.pdf "${CONTAINER_NAME}:/tmp/ddev.pdf"
  assert_success
  run docker exec -t $CONTAINER_NAME magick /tmp/ddev.pdf /tmp/ddev_pdf.png
  assert_success
  run docker exec -t $CONTAINER_NAME bash -c "magick -list format | grep -i 'Portable Document Format'"
  assert_success
  assert_output --partial "rw+"
}

@test "ImageMagick can convert AVIF files" {
  run docker cp tests/ddev-webserver/testdata/imagemagick/logo.avif "${CONTAINER_NAME}:/tmp/logo.avif"
  assert_success
  run docker exec -t $CONTAINER_NAME magick /tmp/logo.avif /tmp/logo_avif.png
  assert_success
  run docker exec -t $CONTAINER_NAME bash -c "magick -list format | grep -i 'AV1 Image File Format'"
  assert_success
  assert_output --partial "rw+"
}

@test "ImageMagick can convert HEIC files" {
  run docker cp tests/ddev-webserver/testdata/imagemagick/logo.heic "${CONTAINER_NAME}:/tmp/logo.heic"
  assert_success
  run docker exec -t $CONTAINER_NAME magick /tmp/logo.heic /tmp/logo_heic.png
  assert_success
  run docker exec -t $CONTAINER_NAME bash -c "magick -list format | grep -i 'High Efficiency Image Format'"
  assert_success
  assert_output --partial "rw+"
}
