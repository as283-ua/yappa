package handler

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/as283-ua/yappa/api/gen"
	"github.com/as283-ua/yappa/internal/server/db"
	"github.com/as283-ua/yappa/internal/server/settings"
	"github.com/as283-ua/yappa/pkg/common"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/proto"
)

func Connection(w http.ResponseWriter, r *http.Request) {
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

// Initial registration flow.
// User requests creating an account. Checks if name is not in use. If not, generate one time token and send to ca server
// Respond to client with the generated token if the request was accepted by the ca server
func RegisterInit(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error reading registration request body:", err)
		return
	}

	request := &gen.RegistrationRequest{}
	err = proto.Unmarshal(body, request)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error unmarshalling registration request:", err)
		return
	}

	queries := db.New(db.Pool)

	_, err = queries.GetUserByUsername(r.Context(), request.User)
	if err != nil && err != pgx.ErrNoRows {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error unmarshalling registration request:", err)
		return
	}

	oneTimeToken := make([]byte, 64)
	_, err = rand.Read(oneTimeToken)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error generating token:", err)
		return
	}

	allowUser := &gen.AllowUser{
		User:  request.User,
		Token: oneTimeToken,
	}

	caReq, err := proto.Marshal(allowUser)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error marshalling allow user request:", err)
		return
	}

	caAllowUrl := fmt.Sprintf("https://%v/allow", settings.ChatSettings.CaAddr)
	caResp, err := common.HttpClient.Post(caAllowUrl, "application/x-protobuf", bytes.NewReader(caReq))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error requesting /allow user:", err)
		return
	}

	defer caResp.Body.Close()
	io.ReadAll(caResp.Body)

	if caResp.StatusCode != http.StatusOK {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Got error code from CA server:", caResp.StatusCode)
		return
	}

	err = queries.CreateUser(r.Context(), request.User)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error creating user in DB:", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(caReq)
}
