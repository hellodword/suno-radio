# suno-radio

[![Matrix Space](https://img.shields.io/matrix/suno-radio:matrix.org)](https://matrix.to/#/#suno-radio:matrix.org)

Turn a suno playlist into a random music radio!

I'm very new about streaming media, so this tool maybe buggy, PR welcomes!

## Usage

Basicly it's an infinite MP3 stream, so we can play it anywhere:

```sh
curl -s http://127.0.0.1:3000/v1/playlist/trending | \
  ffplay -autoexit -nodisp -loglevel quiet -

curl -s http://127.0.0.1:3000/v1/playlist/trending | \
  mpv -

# a test instance for myself, maybe unstable
curl -s https://revisions-shoes-terrorists-endorsed.trycloudflare.com/v1/playlist/trending | \
  ffplay -autoexit -nodisp -loglevel quiet -
```

- Get all playlists

```sh
curl http://127.0.0.1:3000/v1/playlist
```

- Add a new playlist (only if the `auth` is not empty in the [server.yml](./server.yml))

You can get the playlist id from the URL, for example `cc14084a-2622-4c4b-8258-1f6b4b4f54b3` in the `https://app.suno.ai/playlist/cc14084a-2622-4c4b-8258-1f6b4b4f54b3/`

Also, you'd like to create your own playlist on https://app.suno.ai/me/ and add clips into it.

```sh
curl -X PUT -H 'SUNO-RADIO-AUTH: VMkBqnjDUtQB65a9eDKSFhgAIhs8pPdri7rzrd7RO2w' \
  http://127.0.0.1:3000/v1/playlist/cc14084a-2622-4c4b-8258-1f6b4b4f54b3
```

Bravo! You've got your own music radio! It's hosted on `http://127.0.0.1:3000/v1/playlist/cc14084a-2622-4c4b-8258-1f6b4b4f54b3`

## Build

- Requirements:
  - **Go 1.22+**
  - ffmpeg for generating [silence.mp3](./pkg/suno/silence.mp3)

```sh
git clone --depth=1 https://github.com/hellodword/suno-radio
cd suno-radio

go build -trimpath -ldflags "-s -w" -o suno-radio -buildvcs=false ./cmd/suno-radio
# or build with docker
# docker run --rm -v "$(pwd)":/tmp/src -w /tmp/src golang:1-bullseye \
#  go build -trimpath -ldflags "-s -w" -o suno-radio -buildvcs=false ./cmd/suno-radio

./suno-radio -config ./server.yml
```

On my lowend VPS:

```sh
docker compose stats --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" --no-stream
# NAME               CPU %     MEM USAGE
# suno-radio-pub-1   0.08%     20.28MiB
# suno-radio-app-1   0.00%     12.35MiB
```

## TODO

> https://github.com/search?q=repo%3Ahellodword%2Fsuno-radio%20%2F(%3F-i)%5C%2F%5C%2F%20TODO%2F%20NOT%20path%3A3rd&type=code

## Thanks

- https://coderadio.freecodecamp.org/
- https://en.wikipedia.org/wiki/MP3#/media/File:MP3filestructure.svg
- https://github.com/hajimehoshi/go-mp3/tree/v0.3.4
