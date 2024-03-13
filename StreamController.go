package main

import (
	"io"
	"log"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/cassava/lackey/audio/mp3"
	"github.com/google/uuid"
)

type Client struct {
	exitSignalChannel chan bool
	stream            io.Writer
}

type StreamRegulatorAndTransformer struct {
	defaultChunkSize uint
	mutex            sync.Mutex
	modification     [1]string
	buffer           []byte
	stream           io.Reader
	clientStreams    map[string]Client
}

func (srm *StreamRegulatorAndTransformer) AppendClient(stream io.Writer, c chan bool) {
	streamId := uuid.NewString()

	srm.mutex.Lock()
	srm.clientStreams[streamId] = Client{
		exitSignalChannel: c,
		stream:            stream,
	}
	srm.mutex.Unlock()

	slog.Info("New Client Stream", "id", streamId)
}

func (srm *StreamRegulatorAndTransformer) RemoveClient(streamId string) {
	srm.mutex.Lock()
	delete(srm.clientStreams, streamId)
	srm.mutex.Unlock()
}

func (srm *StreamRegulatorAndTransformer) StartStream() {
	for {
		if len(srm.clientStreams) <= 0 {
			continue
		}

		if srm.modification[0] != "" {
			waitTime, err := srm.executeStreamTransformation(srm.modification[0], srm.stream)
			srm.deleteCurrentActiveModification()
			if err != nil {
				slog.Error("Audio Wait Time", "message", err.Error())
				// In this case, set arbitrary wait time based on all audios duration seconds, 
				// except audiostream.mp3
				time.Sleep(time.Second * 8)
			} else {
				// Here, we remove some  milliseconds from our waitTime
				// cause if we dont do this, in the stream, the client will notice
				// a certain "pause" on audio. Indeed, cause we stoped the browser from receiving stream chunks
				// until our waitime has passed.
				time.Sleep(waitTime - (time.Millisecond * 100))
			}

			continue
		}

		n, err := srm.stream.Read(srm.buffer)
		if err != nil {
			log.Println("STREAM READ ERROR", err.Error())
			break
		} else if n == 0 {
			break
		}

		srm.broadcast(srm.buffer[:n])

		// Browsers usually make many requests for the stream chunks, so it has a greater
		// buffer, even if we have reduced the chunks size. To avoid our audio fx to be played
		// too much further than requested, we'll control the delay a browser (or client in general) will
		// receive the chunks. So we add a simple sleep.
		time.Sleep(time.Second * 2)
	}
}

func (srm *StreamRegulatorAndTransformer) broadcast(buf []byte) {
	for id, client := range srm.clientStreams {
		if _, err := client.stream.Write(buf); err != nil {
			// the client disconnected
			slog.Warn("ClientStream Write", "message", err.Error())
			client.exitSignalChannel <- true
			srm.RemoveClient(id)
		}
	}
}

func (srm *StreamRegulatorAndTransformer) getAudioDuration(filename string) (time.Duration, error) {
	metadata, err := mp3.ReadMetadata(filename)
	if err != nil {
		return time.Duration(0), err
	}

	return metadata.Length(), nil
}

func (srm *StreamRegulatorAndTransformer) executeStreamTransformation(audioName string, stdin io.Reader) (time.Duration, error) {
	ffmpeg := CreateFFMPEGProcess(audioName)
	ffmpeg.Stdin = stdin

	stdout, _ := ffmpeg.StdoutPipe()

	go func() {
		tmpBuffer := make([]byte, 100_000)
		outgoingBuffer := []byte{}
		var totalRead int

		for {
			n, err := stdout.Read(tmpBuffer)
			if err != nil {
				break
			} else if n == 0 {
				break
			}

			totalRead += n
			outgoingBuffer = append(outgoingBuffer, tmpBuffer[:n]...)
		}

		srm.broadcast(outgoingBuffer[:totalRead])
	}()

	if err := ffmpeg.Start(); err != nil {
		log.Fatal("FFMPEG START ERROR", err.Error())
	}
	ffmpeg.Wait()

	// Forcing a GC causes ffmpeg spawning continuously increases memory usage of the main
	// process
	runtime.GC()

	// Getting audio duration to forward, we can delay the next client
	// requests according to the duration provided.
	return srm.getAudioDuration(audioName)
}

func (srm *StreamRegulatorAndTransformer) SetModification(audioName string) {
	srm.mutex.Lock()
	srm.modification[0] = audioName
	srm.mutex.Unlock()
}

func (srm *StreamRegulatorAndTransformer) deleteCurrentActiveModification() {
	srm.mutex.Lock()
	srm.modification[0] = ""
	srm.mutex.Unlock()
}

func NewStreamRegulatorAndTransformer(stream io.Reader, chunkSize uint) *StreamRegulatorAndTransformer {
	return &StreamRegulatorAndTransformer{
		defaultChunkSize: chunkSize,
		stream:           stream,
		buffer:           make([]byte, chunkSize),
		clientStreams:    make(map[string]Client),
	}
}
