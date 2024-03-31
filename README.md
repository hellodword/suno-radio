# suno-radio

Turn a suno playlist into a random music radio!

## Usage

Basicly it's an infinite MP3 stream, so we can play it anywhere:

```sh
curl -s http://127.0.0.1:3000/v1/playlist/trending | ffplay -autoexit -nodisp -loglevel quiet -

curl -s http://127.0.0.1:3000/v1/playlist/trending | mpv -

# a test instance for myself, maybe unstable
curl -s https://revisions-shoes-terrorists-endorsed.trycloudflare.com/v1/playlist/trending | ffplay -autoexit -nodisp -loglevel quiet -
```

I'm very new about streaming media, so this tool maybe buggy, PR welcomes!

## Build

- Requirements:
  - **Go 1.22+**
  - ffmpeg for generating [silence.mp3](./pkg/suno/silence.mp3)

```sh
git clone --depth=1 https://github.com/hellodword/suno-radio
cd suno-radio

go build -trimpath -ldflags "-s -w" -o suno-radio -buildvcs=false ./cmd
# or build with docker
# docker run --rm -v "$(pwd)":/tmp/src -w /tmp/src golang:1-bullseye go build -trimpath -ldflags "-s -w" -o suno-radio -buildvcs=false ./cmd

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
