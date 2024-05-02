## audio streaming

Audio streaming is really complex for beginner (me), `suno-radio` tried three ways for infinite audio streaming.

First of all, I want to keep it working on a very low-end VPS, so I cannot re-encoding the audio files, at least not for per stream.

### MP3

As I can say, it works.

I simply concat frames of MP3 files into the stream, and the browser and the players works great with it.

The provides the best performance, because all the audio files are MP3.

But when I dump the file with ffmpeg, it will show the waring:

```sh
$ ffmpeg -hide_banner -i trending.mp3
[mp3 @ 0xdfa300] invalid concatenated file detected - using bitrate for duration
```

How to remove the warning? Re-encoding.

### Wav

I converted the MP3 files to Wav files with ffmpeg after downloading, and with a Wav header, I can stream all PCM data easily.

But PCM is very large, 10x than MP3 for `suno-radio`.

### Ogg & Opus

This is born for streaming.

1. download MP3
2. convert MP3 to Ogg Opus with ffmpeg
3. parse Ogg without decoding, record the offset of audio packets
4. decode the packets of the EOS page to get the PCM size, which effects the next granule postion
5. re-calculate the granule postions when using pages, without re-encoding
6. write BOS page and `OpusHead` page, then read from the memory buffer of pages

---

### misc

```sh
# generate opus silence
ffmpeg -y -f lavfi -i anullsrc=r=48000:cl=stereo -t 3 -b:a 192k -ac 2 -c:a libopus silence.ogg

# generate mp3 silence
ffmpeg -y -f lavfi -i anullsrc=r=48000:cl=stereo -t 3 -b:a 192k -ac 2 silence.mp3
```

---

### Thanks

- https://en.wikipedia.org/wiki/MP3#/media/File:MP3filestructure.svg
- https://github.com/hajimehoshi/go-mp3/tree/v0.3.4
- https://github.com/u2takey/ffmpeg-go
- https://web.archive.org/web/20240406122535/http://soundfile.sapp.org/doc/WaveFormat/
- https://gist.github.com/andreasjansson/1428176
- https://opus-codec.org/
  - https://opus-codec.org/examples/
- https://github.com/hraban/opus/tree/v2
  - https://github.com/dh1tw/remoteAudio/tree/fafc017fbdda08bd4606ceb5479040b7b1f109c7/audiocodec/opus
- https://docs.fileformat.com/audio/
- https://github.com/bluenviron/mediamtx
- https://xiph.org/flac/format.html#frame_header
- https://github.com/mewkiz/flac
- https://xiph.org/ogg/doc/framing.html
- https://github.com/search?q=%2FOpusTags%2F+language%3AGo&type=code
- https://chenliang.org/2020/03/14/ogg-container-format/
- https://chenliang.org/2020/03/15/opus-format/
- https://chenliang.org/2020/04/17/ogg-encapsulation-for-opus/
- https://stackoverflow.com/questions/36417199/how-to-broadcast-message-using-channel/64536953#64536953
