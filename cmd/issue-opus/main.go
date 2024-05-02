package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/hellodword/suno-radio/internal/mp3toogg"
	"github.com/hellodword/suno-radio/internal/ogg"
	"gopkg.in/hraban/opus.v2"
)

func countPackets(packets [][]byte) string {
	s := ""

	s += strconv.Itoa(len(packets)) + "|"
	for i := range packets {
		s += strconv.Itoa(len(packets[i])) + ","
	}
	return s
}

var (
	silence1sEOSPCMLen = 312
	silence1sEOS       = [1][]byte{
		{252, 255, 254},
	}
	silence1sPCMLen = 48000
	silence1s       = [50][]byte{
		{252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254},
		{252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254},
		{252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254},
		{252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254},
		{252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254},
		{252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254},
		{252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254},
		{252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254},
		{252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254},
		{252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254}, {252, 255, 254},
	}
)

func main() {

	_, err := mp3toogg.ConvertMP3ToOgg("2.mp3", "2.ogg")
	if err != nil {
		panic(err)
	}

	f, err := os.Open("2.ogg")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	opusDecoder, err := opus.NewDecoder(48000, 2)
	if err != nil {
		panic(err)
	}

	opusEncoder, err := opus.NewEncoder(48000, 2, opus.AppAudio)
	if err != nil {
		panic(err)
	}

	ff, err := os.Create("2-2.ogg")
	if err != nil {
		panic(err)
	}
	defer ff.Close()

	var serial uint32 = 1

	w := ogg.NewEncoder(serial, ff)

	granule := int64(0)

	granule += 10000

	{
		idh := &ogg.IDHeader{
			Version:              1,
			OutputChannelCount:   2,
			PreSkip:              0,
			InputSampleRate:      48000,
			OutputGainQ7_8:       0,
			ChannelMappingFamily: 0,
		}

		packets, err := idh.Encode()
		if err != nil {
			panic(err)
		}

		err = w.EncodeBOS(0, packets)
		if err != nil {
			panic(err)
		}
	}

	{
		cmh := &ogg.CommentHeader{
			VendorString: "suno-radio",
			UserCommentList: map[string]string{
				"CONTACT": "https://github.com/hellodword/suno-radio",
			},
		}

		packets, err := cmh.Encode()
		if err != nil {
			panic(err)
		}

		err = w.Encode(0, packets)
		if err != nil {
			panic(err)
		}
	}

	for range 1 {
		_, err = f.Seek(0, io.SeekStart)
		if err != nil {
			panic(err)
		}

		for range 3 {
			granule += int64(silence1sPCMLen)
			err = w.Encode(granule, silence1s[:])
			if err != nil {
				panic(err)
			}
		}
		granule += int64(silence1sEOSPCMLen)
		err = w.Encode(granule, silence1sEOS[:])
		if err != nil {
			panic(err)
		}

		d := ogg.NewDecoder(f)

		for {
			p, err := d.Decode()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				panic(err)
			}

			fmt.Println(p.Type, "Granule", p.Granule, p.Serial, countPackets(p.Packets))
			fmt.Println(p.Packets)

			if p.Type&ogg.BOS == ogg.BOS {
				var idh = &ogg.IDHeader{}
				err = idh.Decode(p.Packets)
				if err != nil {
					panic(err)
				}

				fmt.Printf("ID Header %+v\n", idh)
				continue
			}

			if p.Type&ogg.COP == ogg.COP {
				panic(p.Type)
			}

			var cmh = &ogg.CommentHeader{}
			if cmh.Decode(p.Packets) == nil {

				fmt.Printf("Comment Header %+v\n", cmh)

				continue
			}

			if len(p.Packets) > 0 {
				var pcm = make([]int16, 48000*2)

				pos := 0

				for i := range p.Packets {
					n, err := opusDecoder.Decode(p.Packets[i], pcm[pos:])
					if err != nil {
						fmt.Println("type", p.Type, p.Granule)
						panic(err)
					}
					pos += n
					fmt.Println("!Decode", p.Granule, i, n, pos, int(p.Granule%48000))
					pos = int(p.Granule % 48000)

				}

				if p.Type&ogg.EOS == ogg.EOS {
					var packet = make([]byte, 65307)

					secondPCM := make([]int16, 48000*2)

					n, err := opusEncoder.Encode(secondPCM, packet)
					if err != nil {
						panic(err)
					}

					fmt.Println("encode", n)

					fmt.Println(p.Packets)
					p.Packets = [][]byte{packet[:n]}
					fmt.Println(p.Packets)
				}

			}

			inPos, _ := ff.Seek(0, io.SeekCurrent)
			fmt.Println("in ", inPos)

			if len(p.Packets) > 0 {
				for range 1 {
					granule += 48000 //int64(pcmLen)
					err = w.Encode(granule, p.Packets)
					if err != nil {
						panic(err)
					}
				}
			}

			outPos, _ := ff.Seek(0, io.SeekCurrent)
			fmt.Println("out", outPos, outPos-inPos)

		}
	}

	// if false {

	// 	r, oh, err := oggreader.NewWith(f)
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	_ = r

	// 	fmt.Printf("%+v\n", oh)

	// 	for i := range 100 {
	// 		fmt.Println("in")
	// 		fmt.Println(f.Seek(0, io.SeekCurrent))
	// 		data, ph, err := r.ParseNextPage()
	// 		fmt.Println("out")
	// 		fmt.Println(f.Seek(0, io.SeekCurrent))
	// 		if errors.Is(err, io.EOF) {
	// 			break
	// 		}

	// 		fmt.Println("data", len(data))
	// 		fmt.Println("GranulePosition", ph.GranulePosition)

	// 		if i == 2 {
	// 			break
	// 		}

	// 	}

	// }

	// dst, err := os.Create("dst.ogg")
	// if err != nil {
	// 	panic(err)
	// }
	// defer dst.Close()

	// var pos int64 = 138
	// var pos2 int64 = 1273

	// fmt.Println(f.Seek(0, io.SeekStart))
	// fmt.Println(io.CopyN(dst, f, pos))

	// for range 1 {
	// 	fmt.Println(f.Seek(pos, io.SeekStart))
	// 	fmt.Println(io.CopyN(dst, f, pos2-pos))
	// }

	// fmt.Println(f.Seek(pos2, io.SeekStart))
	// fmt.Println(io.Copy(dst, f))

	// s, err := opus.NewStream(f)
	// if err != nil {
	// 	panic(err)
	// }
	// defer s.Close()

	// pcm := make([]int16, 16384)

	// for {
	// 	n, err := s.Read(pcm)
	// 	if errors.Is(err, io.EOF) {
	// 		break
	// 	} else if err != nil {
	// 		panic(err)
	// 	}

	// 	fmt.Println(len(pcm), n)
	// 	// send pcm to audio device here, or write to a .wav file

	// }

}
