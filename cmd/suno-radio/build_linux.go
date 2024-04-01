//go:build linux

package main

// //go:generate ffmpeg -y -f lavfi -i anullsrc=r=48000:cl=stereo -t 0.1 -b:a 192k -ac 2 ../../pkg/suno/silence.mp3
//go:generate go build -trimpath -ldflags "-s -w" -o suno-radio -buildvcs=false .
//go:generate rm -f suno-radio.zip
//go:generate zip -j suno-radio.zip suno-radio ../../server.yml ../../docker-compose.yml
