package common

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

type BiStream struct {
	io.ReadWriteCloser

	qconn  *quic.Connection
	h3Conn *http3.ClientConn
	stream *http3.RequestStream
	resp   *http.Response
}

func (s BiStream) Context() context.Context {
	return s.h3Conn.Connection.Context()
}

func (s BiStream) Read(p []byte) (n int, err error) {
	return (*s.stream).Read(p)
}

func (s BiStream) Write(p []byte) (n int, err error) {
	return (*s.stream).Write(p)
}

func (s BiStream) Close() error {
	err := s.resp.Body.Close()
	if err != nil {
		log.Println("Close error: ", err)
	}
	err = (*s.qconn).CloseWithError(0, "Client closed connection")
	if err != nil {
		log.Println("Close error: ", err)
	}
	err = (*s.stream).Close()
	if err != nil {
		log.Println("Close error: ", err)
	}

	return nil
}

func Http3Stream(ctx context.Context, url *url.URL, tr *http3.Transport, header http.Header) (*BiStream, error) {
	result := &BiStream{}
	req := &http.Request{
		Method: http.MethodConnect,
		Proto:  "HTTP/3",
		Host:   url.Host,
		URL:    url,
		Header: header,
	}
	req = req.WithContext(ctx)

	conn, err := quic.DialAddr(ctx, url.Host, tr.TLSClientConfig, tr.QUICConfig)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	clientConn := tr.NewClientConn(conn)

	select {
	case <-clientConn.ReceivedSettings():
	case <-ctx.Done():
		return result, fmt.Errorf("connection closed")
	}

	settings := clientConn.Settings()
	if !settings.EnableExtendedConnect {
		return result, fmt.Errorf("server didn't enable Extended CONNECT")
	}
	if !settings.EnableDatagrams {
		return result, fmt.Errorf("server didn't enable HTTP/3 datagram support")
	}

	requestStr, err := clientConn.OpenRequestStream(ctx)
	if err != nil {
		return result, err
	}

	if err := requestStr.SendRequestHeader(req); err != nil {
		return result, err
	}

	clientConn.Settings().EnableDatagrams = true

	rsp, err := requestStr.ReadResponse()

	if err != nil {
		return result, err
	}

	if rsp.StatusCode < 200 || rsp.StatusCode >= 300 {
		return result, fmt.Errorf("received status %v", rsp.Status)
	}

	return &BiStream{qconn: &conn, h3Conn: clientConn, stream: &requestStr, resp: rsp}, nil
}
