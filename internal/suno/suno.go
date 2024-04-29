package suno

import (
	"bytes"
	_ "embed"
	"encoding/binary"
)

const (
	PlaylistTrending = "1190bf92-10dc-4ce5-968a-7a377f37f984"
	PlaylistNew      = "cc14084a-2622-4c4b-8258-1f6b4b4f54b3"

	DefaultWavSampleRate = 16000
	PCMLenOf100ms        = 6400
)

var PCMSilence100ms = make([]byte, PCMLenOf100ms)
var infiniteWavHeader []byte

func init() {
	var buf = bytes.NewBuffer(nil)

	buf.Write([]byte("RIFF"))                 // ChunkID
	buf.Write([]byte{0, 0, 0, 0})             // ChunkSize
	buf.Write([]byte("WAVE"))                 // Format
	buf.Write([]byte("fmt "))                 // Subchunk1ID
	buf.Write([]byte{0x10, 0x00, 0x00, 0x00}) // Subchunk1Size 16 for PCM
	buf.Write([]byte{0x01, 0x00})             // AudioFormat PCM = 1
	buf.Write([]byte{0x02, 0x00})             // NumChannels      Mono = 1, Stereo = 2

	var temp = make([]byte, 4)

	binary.LittleEndian.PutUint32(temp, DefaultWavSampleRate)
	buf.Write(temp) // SampleRate

	binary.LittleEndian.PutUint32(temp, DefaultWavSampleRate*2*16/8)
	buf.Write(temp) // ByteRate

	buf.Write([]byte{0x04, 0x00}) // BlockAlign
	buf.Write([]byte{0x10, 0x00}) // BitsPerSample
	buf.Write([]byte("data"))     // Subchunk2ID
	buf.Write([]byte{0, 0, 0, 0}) // Subchunk2Size

	infiniteWavHeader = buf.Bytes()

	if len(infiniteWavHeader) != 44 {
		panic("generate infinite wav header failed!")
	}

}
