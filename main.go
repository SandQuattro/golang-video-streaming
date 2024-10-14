package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const sourceFile = "raceboat.mp4"

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintln(w, "Welcome to our video streaming platform!")
		if err != nil {
			log.Fatal(err)
		}
	})

	http.HandleFunc("/stream", handleStream)
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open(sourceFile)
	if err != nil {
		http.Error(w, "File not found.", http.StatusNotFound)
		return
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	fi, err := file.Stat()
	if err != nil {
		http.Error(w, "File stat error.", http.StatusInternalServerError)
		return
	}

	fileSize := fi.Size()
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Accept-Ranges", "bytes")

	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
		http.ServeContent(w, r, sourceFile, fi.ModTime(), file)
		return
	}

	var start, end int64
	parts := strings.Split(rangeHeader, "=")
	if len(parts) < 2 {
		http.Error(w, "Invalid range header.", http.StatusBadRequest)
		return
	}
	rangeParts := strings.Split(parts[1], "-")
	start, err = strconv.ParseInt(rangeParts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid range start.", http.StatusBadRequest)
		return
	}
	if len(rangeParts) > 1 && rangeParts[1] != "" {
		end, err = strconv.ParseInt(rangeParts[1], 10, 64)
		if err != nil {
			http.Error(w, "Invalid range end.", http.StatusBadRequest)
			return
		}
	} else {
		end = fileSize - 1
	}

	if start > end || start < 0 || end >= fileSize {
		http.Error(w, "Invalid range.", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
	w.WriteHeader(http.StatusPartialContent)

	_, err = file.Seek(start, 0)
	if err != nil {
		http.Error(w, "File positioning error. ", http.StatusInternalServerError)
		return
	}
	buffer := make([]byte, 64*1024) // 64KB buffer size
	_, err = io.CopyBuffer(w, &io.LimitedReader{R: file, N: end - start + 1}, buffer)
	if err != nil {
		log.Println("Error writing buffer:", err)
		return
	}
}
