#! /usr/bin/env bash

set -e
set -x

mkdir -p dist

docker kill suno-radio-builder && sleep 1s || true
docker rm suno-radio-builder || true

docker run --name suno-radio-builder --rm -v "$(pwd)":/tmp/src -w /tmp/src -d golang:1-bullseye sleep infinity

docker exec suno-radio-builder \
  go build -trimpath -ldflags "-s -w" -o dist/suno-radio -buildvcs=false ./cmd/suno-radio

docker exec suno-radio-builder \
  go build -trimpath -ldflags "-s -w" -o dist/mp3-to-ogg -buildvcs=false ./cmd/mp3-to-ogg

docker kill suno-radio-builder && sleep 1s || true
docker rm suno-radio-builder || true

cp docker-compose.deploy.yml dist/docker-compose.yml
cp server.yml dist/server.yml

rm -f dist/suno-radio.zip
zip -j dist/suno-radio.zip dist/suno-radio dist/mp3-to-ogg dist/server.yml dist/docker-compose.yml
