package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/yapingcat/gomedia/go-codec"
)

func main() {
	f, err := os.Create("output.mp3")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// fmt.Println(f.Write(suno.MP3Header))
	// fmt.Println(f.Write(suno.MP3Silence100ms))
	// fmt.Println(f.Write(suno.MP3Silence100ms[len(suno.MP3Header):]))
	// fmt.Println(f.Write(suno.MP3Silence100ms[len(suno.MP3Header):]))

	data, _ := os.ReadFile("3.mp3")
	fmt.Println("Get Mp3 file size", len(data))

	// fmt.Println(hex.Dump(data[:45]))
	// fmt.Println(hex.Dump(data[len(data)/2 : len(data)/2+45]))

	// fmt.Println(len(data) / 2)
	// fmt.Println(len(data[:len(data)/2]))
	// fmt.Println(len(data[len(data)/2:]))
	// fmt.Println(bytes.Compare(data[:len(data)/2][45:45+576], data[len(data)/2:][45:45+576]))

	// fmt.Println("\n\n" + hex.Dump(data[:len(data)/2][45:45+576]))

	// fmt.Println("\n\n" + hex.Dump(data[len(data)/2:][45:45+576]))

	i := 0
	for range 1 {

		codec.SplitMp3Frames(data, func(head *codec.MP3FrameHead, frame []byte) {
			fmt.Println("Get mp3 Frame", len(frame))
			fmt.Printf("mp3 frame head %+v\n", head)
			fmt.Printf("mp3 bitrate:%d,samplerate:%d,channelcount:%d\n", head.GetBitRate(), head.GetSampleRate(), head.GetChannelCount())

			// fmt.Println("frame[3]", "in ", frame[3])
			// if i > 0 && head.ModeExtension != 0 {
			// 	frame[3] = 0x64
			// }
			// fmt.Println("frame[3]", "out", frame[3])

			// f.Write(frame)

			i++

			fmt.Println(hex.Dump(frame))

			// if i > 9 {
			// 	panic(1)
			// }
		})
	}

	// // fmt.Println(len(suno.MP3Header), 4077-4032)
}
