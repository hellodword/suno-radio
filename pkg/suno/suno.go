package suno

import _ "embed"

const (
	PlaylistTrending = "1190bf92-10dc-4ce5-968a-7a377f37f984"
	PlaylistNew      = "cc14084a-2622-4c4b-8258-1f6b4b4f54b3"

	DefaultSampleRate = 48000
)

var MP3Header = []byte{
	73, 68, 51, 4, 0, 0, 0, 0, 0, 35, 84, 83, 83, 69, 0, 0,
	0, 15, 0, 0, 3, 76, 97, 118, 102, 53, 56, 46, 52, 53, 46,
	49, 48, 48, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
}

//go:embed silence.mp3
var MP3Silence100ms []byte
