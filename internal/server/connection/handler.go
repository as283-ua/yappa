package connection

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/as283-ua/yappa/api/gen"
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
	username := r.TLS.PeerCertificates[0].Subject.CommonName
	logger := logging.GetLogger()
	logger.Println("Someone connected:", username)

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
		case *gen.ClientMessage_Init:
			chatInit := payload.Init
			logger.Println(chatInit)
		case *gen.ClientMessage_Hb:
		default:
			// Unknown or unset
		}
	}
}
