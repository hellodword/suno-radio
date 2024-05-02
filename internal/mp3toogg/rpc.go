package mp3toogg

import (
	"bytes"
	"fmt"
	"net/rpc"
	"os"
	"path"

	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

type MP3ToOggArgs struct {
	Playlist   string
	ClipID     string
	SampleRate int
	Channels   int
}

const MP3ToOggFuncConvert = "MP3ToOgg.Convert"

type MP3ToOgg struct{}

func (t *MP3ToOgg) Convert(args *MP3ToOggArgs, reply *string) error {

	pmp3 := path.Join("data", args.Playlist, fmt.Sprintf("%s.mp3", args.ClipID))
	pogg := path.Join("data", args.Playlist, fmt.Sprintf("%s.ogg", args.ClipID))

	os.Remove(pogg)

	_, err := ConvertMP3ToOgg(pmp3, pogg)
	if err != nil {
		return err
	}

	*reply = pogg

	return nil
}

// func (t *MP3ToOgg) fixEOS(args *MP3ToOggArgs, pogg string) error {

// 	f, err := os.OpenFile(pogg, os.O_RDWR, 0644)
// 	if err != nil {
// 		return err
// 	}
// 	defer f.Close()

// 	d := ogg.NewDecoder(f)

// 	var lastPosition, lastGranule int64

// 	for {
// 		p, err := d.Decode()
// 		if err != nil {
// 			if errors.Is(err, io.EOF) {
// 				break
// 			}
// 			return err
// 		}

// 		if p.Type&ogg.EOS == ogg.EOS {
// 			opusDecoder, err := opus.NewDecoder(args.SampleRate, args.Channels)
// 			if err != nil {
// 				return err
// 			}

// 			opusEncoder, err := opus.NewEncoder(args.SampleRate, args.Channels, opus.AppAudio)
// 			if err != nil {
// 				return err
// 			}

// 			var pcmPos int
// 			var pcm = make([]int16, 2880*2)
// 			for i := range p.Packets {
// 				n, err := opusDecoder.Decode(p.Packets[i], pcm[pcmPos:])
// 				if err != nil {
// 					return err
// 				}
// 				pcmPos += n
// 			}

// 			var packet = make([]byte, 65307)
// 			n, err := opusEncoder.Encode(pcm, packet)
// 			if err != nil {
// 				return err
// 			}

// 			packet = packet[:n]

// 			e := ogg.NewEncoder(p.Serial, f)
// 			_, err = f.Seek(lastPosition, io.SeekStart)
// 			if err != nil {
// 				return err
// 			}

// 			return e.Encode(lastGranule+48000, [][]byte{packet})
// 		}

// 		lastPosition, err = f.Seek(0, io.SeekCurrent)
// 		if err != nil {
// 			return err
// 		}

// 		lastGranule = p.Granule
// 	}

// 	return nil
// }

var rpcClient *rpc.Client

func MP3ToOggInit(address string) error {
	var err error
	rpcClient, err = rpc.DialHTTP("tcp", address)
	if err != nil {
		return err
	}

	return nil
}

func MP3ToOggConvert(args MP3ToOggArgs) (string, error) {
	var reply string
	err := rpcClient.Call(MP3ToOggFuncConvert, args, &reply)
	if err != nil {
		return "", err
	}
	return reply, nil
}

var mutex = make(chan struct{}, 1)

func ConvertMP3ToOgg(src, dst string) (string, error) {
	mutex <- struct{}{}
	errC := make(chan error)

	tmp := dst + ".tmp.ogg"
	var buf = bytes.NewBuffer(nil)

	go func() {
		err := ffmpeg_go.
			Input(src, ffmpeg_go.KwArgs{
				"hide_banner": "",
				"loglevel":    "verbose",
				"threads":     "1",
			}).
			Output(tmp, ffmpeg_go.KwArgs{
				"c:a":     "libopus",
				"threads": "1",
				// "map_metadata": "-1",
			}).
			OverWriteOutput().WithOutput(buf, buf).Run()

		<-mutex
		errC <- err
	}()

	err := <-errC
	if err != nil {
		return buf.String(), err
	}

	err = os.Rename(tmp, dst)
	return buf.String(), err
}
