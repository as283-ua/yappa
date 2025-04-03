package auth

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"

	"github.com/as283-ua/yappa/api/gen"
	"github.com/as283-ua/yappa/internal/server/logging"
	"github.com/as283-ua/yappa/internal/server/settings"
	"github.com/as283-ua/yappa/pkg/common"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/proto"
)

var log = logging.GetLogger()

// username -> token map. tokens generated by ca. value is compared when confirming a registration and assigning a certificate to a user.
var confirmationTokens map[string][]byte = make(map[string][]byte)

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
		http.Error(w, "Incorrect format", http.StatusBadRequest)
		return
	}

	_, err = Repo.GetUserByUsername(r.Context(), request.User)
	if err != nil && err != pgx.ErrNoRows {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Pgx error:", err)
		return
	}

	if err == nil {
		http.Error(w, "Username already taken", http.StatusBadRequest)
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
	confirmationBytes, err := io.ReadAll(caResp.Body)

	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error reading response:", err)
		return
	}

	if caResp.StatusCode != http.StatusOK {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Got error code from CA server:", caResp.StatusCode)
		return
	}

	confirmation := &gen.ConfirmRegistrationToken{}

	err = proto.Unmarshal(confirmationBytes, confirmation)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Couldn't unmarshall CAs response:", err)
		return
	}

	confirmationTokens[confirmation.User] = confirmation.Token

	log.Printf("Authorized user %v to get a certificate\n", confirmation.User)

	w.WriteHeader(http.StatusOK)
	w.Write(caReq)
}

func RegisterComplete(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)

	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error reading body:", err)
		return
	}

	confirmation := &gen.ConfirmRegistration{}

	err = proto.Unmarshal(b, confirmation)
	if err != nil {
		http.Error(w, "Incorrect format", http.StatusBadRequest)
		return
	}

	if token, ok := confirmationTokens[confirmation.User]; !ok || !bytes.Equal(token, confirmation.Token) {
		http.Error(w, "Incorrect confirmation token", http.StatusBadRequest)
		return
	}

	err = Repo.CreateUser(r.Context(), confirmation.User, string(confirmation.Cert))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Println("Error creating user in DB:", err)
		return
	}

	log.Printf("User %v registered\n", confirmation.User)
	w.WriteHeader(http.StatusOK)

	delete(confirmationTokens, confirmation.User)
}
