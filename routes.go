package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

const httpLogString = "%s %s %s\n"

func InitRoutes() {
	http.HandleFunc("GET /stream/audio", func(w http.ResponseWriter, r *http.Request) {
		log.Printf(httpLogString, r.Method, r.URL.Path, r.RemoteAddr[:strings.Index(r.RemoteAddr, ":")])

		w.Header().Add("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)

		exitCh := exitChannel()
		streamRegulator.AppendClient(w, exitCh)
		<-exitCh
	})

	http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		log.Printf(httpLogString, r.Method, r.URL.Path, r.RemoteAddr[:strings.Index(r.RemoteAddr, ":")])

		htmlBuffer, err := os.ReadFile("public/index.html")
		if err != nil {
			slog.Error("html page rendering", "message", err.Error())
			w.Header().Add("Content-Type", "text/plain")
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, "failure rendering index.html page! Please, try it again later.")
			return
		}

		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(htmlBuffer)
	})

	http.HandleFunc("POST /stream/soundeffect", func(w http.ResponseWriter, r *http.Request) {
		log.Printf(httpLogString, r.Method, r.URL.Path, r.RemoteAddr[:strings.Index(r.RemoteAddr, ":")])

		if err := r.ParseForm(); err != nil {
			slog.Error("FormData Parsing", "message", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if len(r.PostForm["soundname"]) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("form data is empty"))
			return
		}

		soundName := r.PostForm["soundname"][0]
		if soundName == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing sound effect name"))
		} else if _, err := os.Stat("public/media/" + soundName); err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("sound effect not found"))
		} else {
			streamRegulator.SetModification(fmt.Sprintf("public/media/" + soundName))
			w.WriteHeader(http.StatusOK)
		}
	})

	http.HandleFunc("GET /controlpanel", func(w http.ResponseWriter, r *http.Request) {
		buf, _ := os.ReadFile("public/controlpanel.html")
		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(buf)
	})
}
