package main

import (
	"log"
	"os"
	"os/exec"
)

var FFMPEG_LOGGER *os.File

func init() {
	var err error
	FFMPEG_LOGGER, err = os.Create("ffmpeg.log")

	if err != nil {
		log.Fatal(err)
	}
}

func CreateFFMPEGProcess(media string) *exec.Cmd {
	ffmpeg := exec.Command("ffmpeg", "-i", "pipe:0", "-i", media, "-filter_complex", "amerge=inputs=2", "-ac", "2", "-f", "mp3", "pipe:1")
	ffmpeg.Stderr = FFMPEG_LOGGER
	return ffmpeg
}