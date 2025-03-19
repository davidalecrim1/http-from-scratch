package server

import (
	"io"
	"log"
	"log/slog"
	"net"
	"strings"
	"time"
)

type Server struct {
	addr string
	ln   net.Listener
}

// addr -> ":8080"
func NewServer(addr string) *Server {
	return &Server{
		addr: addr,
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", ":8097")
	if err != nil {
		log.Fatal("failed to kind to the port 8097")
	}

	s.ln = ln
	go s.acceptConnections()
	return nil
}

func (s *Server) acceptConnections() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			slog.Error("failed to receive a new connection, stoping the listener...")
			s.ln.Close()
			return
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	conn.SetDeadline(time.Now().Add(time.Second * 5))
	readCallback := make(chan []byte)
	go s.readConnection(conn, readCallback)

	for request := range readCallback {
		if !strings.HasPrefix(string(request), "GET / HTTP/1.1") {
			conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			continue
		}

		_, err := conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
		if err != nil {
			slog.Debug("failed to write to the connection")
		}
	}

	err := conn.Close()
	if err != nil {
		slog.Error("failed to close the connection", "error", err)
	}
}

func (s *Server) readConnection(conn net.Conn, readCallback chan []byte) {
	readBuf := make([]byte, 1024)
	for {
		n, err := conn.Read(readBuf)
		if err != nil && err == io.EOF {
			slog.Error("reached the EOF of the reading connection, stoping the reads...")
			close(readCallback)
			return

		}
		if err != nil {
			slog.Error("received an error while reading the connection, stoping the reads...")
			close(readCallback)

			return
		}

		receivedData := make([]byte, n)
		copy(receivedData, readBuf[:n])

		readCallback <- receivedData
	}
}
