package common

import (
	"bytes"
	"fmt"
	"net/rpc"
	"os"
	"path"

	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

type MP3ToWavConverterArgs struct {
	Playlist string
	ClipID   string
}

const MP3ToWavConverterFunc = "MP3ToWavConverter.Convert"

type MP3ToWavConverter string

func (t *MP3ToWavConverter) Convert(args *MP3ToWavConverterArgs, reply *string) error {
	pmp3 := path.Join("data", args.Playlist, fmt.Sprintf("%s.mp3", args.ClipID))
	pwav := path.Join("data", args.Playlist, fmt.Sprintf("%s.wav", args.ClipID))

	os.Remove(pwav)
	_, err := ConvertMP3ToWAV(pmp3, pwav)
	if err != nil {
		return err
	}

	*reply = pwav
	return nil
}

func MP3ToWavConverterConvert(client *rpc.Client, args MP3ToWavConverterArgs) error {
	var reply string
	return client.Call(MP3ToWavConverterFunc, args, &reply)
}

// TODO WAV and MP3
//
//	choose WAV because frames of difference MP3 files can't be
//	easily concatenated with their frames without re-encoding,
//  while an infinite stream is required here with less encoding

func ConvertMP3ToWAV(src, dst string) (string, error) {
	var buf = bytes.NewBuffer(nil)
	err := ffmpeg_go.
		Input(src, ffmpeg_go.KwArgs{
			"hide_banner": "",
			"loglevel":    "verbose",
			"threads":     "1",
		}).
		Output(dst, ffmpeg_go.KwArgs{
			"acodec":  "pcm_s16le",
			"ac":      "2",
			"ar":      "16000",
			"threads": "1",
			// "map_metadata": "-1",
		}).
		OverWriteOutput().WithOutput(buf, buf).Run()

	return buf.String(), err
}
