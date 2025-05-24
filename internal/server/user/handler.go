package user

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/as283-ua/yappa/api/gen/server"
	"github.com/as283-ua/yappa/internal/server/logging"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/proto"
)

// Gets list of usersnames
func GetUsernames(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()

	page := intOrDefault(r.Header.Get("page"), 0)
	size := intOrDefault(r.Header.Get("size"), 10)
	name := r.Header.Get("name")

	usernames, err := Repo.GetUsers(page, size, name)
	if err != nil {
		logger.Println("DB error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := server.Usernames{Usernames: usernames}
	respBytes, err := proto.Marshal(&resp)
	if err != nil {
		logger.Println("Protobuf marshal error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Write(respBytes)
	w.WriteHeader(http.StatusOK)
}

func GetUserData(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()

	username := r.PathValue("username")

	user, err := Repo.GetUserData(username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "User doesn't exist", http.StatusNotFound)
			return
		}
		logger.Println("DB error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := server.UserData{
		Username:       user.Username,
		Certificate:    user.Certificate,
		PubKeyExchange: user.PubKeyExchange,
	}

	respBytes, err := proto.Marshal(&resp)
	if err != nil {
		logger.Println("Protobuf marshal error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Write(respBytes)
	w.WriteHeader(http.StatusOK)
}

func intOrDefault(header string, def int) int {
	if header == "" {
		return def
	}

	val, err := strconv.Atoi(header)
	if err != nil {
		return def
	}

	return val
}
