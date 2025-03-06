package handler

import (
	"fmt"
	"io"
	"net/http"
)

func HandleConnection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	fmt.Println(r.RemoteAddr + " connected")
	buf := make([]byte, 1024)
	for {
		n, err := r.Body.Read(buf)

		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection closed by client")
				break
			}

			fmt.Println("Error: ", err)
			continue
		}

		fmt.Fprintf(w, "%s", "Hello")
		flusher.Flush()
		fmt.Println(r.RemoteAddr + ": " + string(buf[:n]))
	}
}
