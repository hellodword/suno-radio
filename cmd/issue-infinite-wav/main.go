package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/go-audio/wav"
	"github.com/hellodword/suno-radio/internal/common"
)

func main() {
	_, err := common.ConvertMP3ToWAV("1.mp3", "1.wav")
	if err != nil {
		panic(err)
	}

	src, err := os.Open("1.wav")
	if err != nil {
		panic(err)
	}

	defer src.Close()
	w := wav.NewDecoder(src)
	w.ReadInfo()

	err = w.FwdToPCM()
	if err != nil {
		panic(err)
	}

	dataOffset, err := src.Seek(0, io.SeekCurrent)
	if err != nil {
		panic(err)
	}

	dst, err := os.Create("1-2.wav")
	if err != nil {
		panic(err)
	}
	defer dst.Close()

	fmt.Println(dst.Write([]byte("RIFF")))                 // ChunkID
	fmt.Println(dst.Write([]byte{0, 0, 0, 0}))             // ChunkSize
	fmt.Println(dst.Write([]byte("WAVE")))                 // Format
	fmt.Println(dst.Write([]byte("fmt ")))                 // Subchunk1ID
	fmt.Println(dst.Write([]byte{0x10, 0x00, 0x00, 0x00})) // Subchunk1Size 16 for PCM
	fmt.Println(dst.Write([]byte{0x01, 0x00}))             // AudioFormat PCM = 1
	fmt.Println(dst.Write([]byte{0x02, 0x00}))             // NumChannels      Mono = 1, Stereo = 2

	var temp = make([]byte, 4)

	binary.LittleEndian.PutUint32(temp, w.SampleRate)
	fmt.Println(dst.Write(temp)) // SampleRate
	binary.LittleEndian.PutUint32(temp, w.SampleRate*2*16/8)
	fmt.Println(dst.Write(temp))               // ByteRate
	fmt.Println(dst.Write([]byte{0x04, 0x00})) // BlockAlign
	fmt.Println(dst.Write([]byte{0x10, 0x00})) // BitsPerSample
	fmt.Println(dst.Write([]byte("data")))     // Subchunk2ID
	fmt.Println(dst.Write([]byte{0, 0, 0, 0})) // Subchunk2Size

	for range 5 {
		fmt.Println(src.Seek(dataOffset, io.SeekStart))
		fmt.Println(io.CopyN(dst, src, int64(w.PCMSize)))

		// 3 seconds silence
		fmt.Println(dst.Seek(0, io.SeekCurrent))
		for range 30 {
			dst.Write(make([]byte, 6400)) // 100ms silence
		}
		fmt.Println(dst.Seek(0, io.SeekCurrent))
	}

}
