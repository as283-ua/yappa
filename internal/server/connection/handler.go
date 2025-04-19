package connection

import (
	"context"
	"crypto/ecdh"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/as283-ua/yappa/api/gen"
	"github.com/as283-ua/yappa/internal/server/auth"
	"github.com/as283-ua/yappa/internal/server/logging"
	"github.com/quic-go/quic-go/http3"
	"google.golang.org/protobuf/proto"
)

var sessions map[string]*http3.Stream = make(map[string]*http3.Stream)

func upgrade(w http.ResponseWriter) (http3.Stream, error) {
	conn := w.(http3.Hijacker).Connection()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-conn.ReceivedSettings():
	case <-timer.C:
		return nil, fmt.Errorf("didn't receive the client's SETTINGS on time")
	}
	settings := conn.Settings()
	if !settings.EnableDatagrams {
		return nil, fmt.Errorf("missing datagram support")
	}

	w.WriteHeader(http.StatusOK)
	w.(http.Flusher).Flush()

	return w.(http3.HTTPStreamer).HTTPStream(), nil
}

func Connection(w http.ResponseWriter, r *http.Request) {
	logger := logging.GetLogger()
	username := r.TLS.PeerCertificates[0].Subject.CommonName
	logger.Println("Someone connected:", username)

	_, err := auth.Repo.GetUserByUsername(context.Background(), username)
	if err == nil {
		http.Error(w, "Invalid user", http.StatusBadRequest)
		return
	}

	ecdhK, ok := r.Context().Value(EcdhCtxKey).(*ecdh.PublicKey)
	if !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	err = auth.Repo.ChangeEcdhTemp(context.Background(), username, ecdhK.Bytes())
	if err != nil {
		logger.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	str, err := upgrade(w)
	if err != nil {
		logger.Println("Upgrade error:", err)
		return
	}
	defer func() {
		delete(sessions, username)
		str.Close()
	}()
	sessions[username] = &str

	for {
		var lenBuf [4]byte
		_, err := io.ReadFull(str, lenBuf[:])
		if err != nil {
			logger.Println("Failed to read length:", err)
			return
		}
		msgLen := binary.BigEndian.Uint32(lenBuf[:])
		var msg []byte = make([]byte, msgLen)

		str.Read(msg)

		protoMsg := &gen.ClientMessage{}
		err = proto.Unmarshal(msg, protoMsg)

		if err != nil {
			logger.Println("Failed to unmarshall client data:", err)
			return
		}

		switch payload := protoMsg.Payload.(type) {
		case *gen.ClientMessage_Send:
			chatSend := payload.Send
			logger.Println(chatSend)
		case *gen.ClientMessage_Hb:
		default:
			// Unknown or unset
		}
	}
}
