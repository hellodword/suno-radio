# suno-radio

[![Matrix Space](https://img.shields.io/matrix/suno-radio:matrix.org)](https://matrix.to/#/#suno-radio:matrix.org)

![](./images/logo.png)

Turn a suno playlist into a random music radio!

I'm very new about streaming media, so this tool maybe buggy, PR welcomes!

## Usage

Basicly it's an ogg, so we can play it almost anywhere, browsers (not the iOS safari), players, or in the cli:

```sh
curl -s http://127.0.0.1:3000/v1/playlist/trending | \
  ffplay -hide_banner -autoexit -nodisp -

curl -s http://127.0.0.1:3000/v1/playlist/trending | \
  mpv -
```

- Get all playlists

```sh
curl http://127.0.0.1:3000/v1/playlist
```

- Add a new playlist (only if the `auth` is not empty in the [server.yml](./server.yml))

You can get the playlist id from the URL, for example `cc14084a-2622-4c4b-8258-1f6b4b4f54b3` in the `https://app.suno.ai/playlist/cc14084a-2622-4c4b-8258-1f6b4b4f54b3/`

Also, you'd like to create your own playlist on https://app.suno.ai/me/ and add clips into it.

```sh
# foo is the alias of the playlist, name it with a [0-9a-z_-]{3,32} string
curl -X PUT -H 'SUNO-RADIO-AUTH: VMkBqnjDUtQB65a9eDKSFhgAIhs8pPdri7rzrd7RO2w' \
  http://127.0.0.1:3000/v1/playlist/cc14084a-2622-4c4b-8258-1f6b4b4f54b3/foo
```

Bravo! You've got your own music radio! It's hosted on `http://127.0.0.1:3000/v1/playlist/cc14084a-2622-4c4b-8258-1f6b4b4f54b3` and `http://127.0.0.1:3000/v1/playlist/foo`

Delete the playlist by id:

```sh
curl -X DELETE -H 'SUNO-RADIO-AUTH: VMkBqnjDUtQB65a9eDKSFhgAIhs8pPdri7rzrd7RO2w' \
  http://127.0.0.1:3000/v1/playlist/cc14084a-2622-4c4b-8258-1f6b4b4f54b3
```

## Online demo

This is an instance for myself, hosted on a very low-end VPS, so it's unstable:

```sh
# list available playlists
curl -s https://looked-livecam-formula-novel.trycloudflare.com/v1/playlist | jq

# There're some alias same with the official playlists
curl -s https://looked-livecam-formula-novel.trycloudflare.com/v1/playlist/trending | \
  ffplay -hide_banner -autoexit -nodisp -

curl -s https://looked-livecam-formula-novel.trycloudflare.com/v1/playlist/weekly | \
  ffplay -hide_banner -autoexit -nodisp -

curl -s https://looked-livecam-formula-novel.trycloudflare.com/v1/playlist/monthly | \
  ffplay -hide_banner -autoexit -nodisp -

curl -s https://looked-livecam-formula-novel.trycloudflare.com/v1/playlist/top | \
  ffplay -hide_banner -autoexit -nodisp -
```

## Build

- Requirements:
  - **Go 1.22+**
  - ffmpeg for converting mp3 to ogg

```sh
git clone --depth=1 https://github.com/hellodword/suno-radio
cd suno-radio

./scripts/build.sh
```

## TODO

> https://github.com/search?q=repo%3Ahellodword%2Fsuno-radio%20%2F(%3F-i)%5C%2F%5C%2F%20TODO%2F%20NOT%20path%3A3rd&type=code

## Thanks

- https://coderadio.freecodecamp.org/
