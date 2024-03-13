package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
)

var (
	address         string
	streamRegulator *StreamRegulatorAndTransformer
)

func init() {
	if err := godotenv.Load(".env"); err != nil {
		log.Println(err)
	}
}

func orElse(s, defaultValue string) string {
	if len(s) == 0 {
		return defaultValue
	}

	return s
}

func main() {
	address = orElse(os.Getenv("ADDRESS"), ":8000")

	audio, err := os.Open("public/media/audiostream.mp3")
	if err != nil {
		log.Fatal(err)
	}
	defer audio.Close()

	// SIGNAL CHANNEL
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt)

	// Listen for signals
	go func(c chan os.Signal) {
		<-channel
		log.Println("SIGINT received")
		audio.Close()
		FFMPEG_LOGGER.Close()
		os.Exit(0)
	}(channel)

	InitRoutes()

	// STREAM REGULATOR AND TRANSFORMER
	streamRegulator = NewStreamRegulatorAndTransformer(audio, 32*1000)
	go streamRegulator.StartStream()

	log.Println("server started at", address)
	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatal(err)
	}
}

func CheckError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func exitChannel() chan bool {
	return make(chan bool)
}
