package http_from_scratch

import (
	"io"
	"log"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	addr string
	ln   net.Listener
}

// e.g. addr -> ":8080"
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
	s.acceptConnections()
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
	err := conn.SetDeadline(time.Now().Add(time.Second * 5))
	if err != nil {
		slog.Error("failed to set deadline for the connection", "error", err)
	}

	readCallback := make(chan []byte)
	go s.readConnection(conn, readCallback)

	for request := range readCallback {
		response := s.handleRequest(request)
		_, err = conn.Write(response)
		if err != nil {
			slog.Debug("failed to write to the connection")
		}
	}

	err = conn.Close()
	if err != nil {
		slog.Error("failed to close the connection", "error", err)
	}
}

func (s *Server) handleRequest(requestBytes []byte) (response []byte) {
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("recovered from panic", "recover", r)
			resp := Response{
				statusCode: 400,
			}
			response = resp.ToBytes()
		}
	}()

	request, err := NewRequest(requestBytes)
	if err != nil {
		slog.Error("failed to create request", "error", err)
		resp := Response{
			statusCode: 400,
		}
		return resp.ToBytes()
	}

	if request.Path == "/" {
		resp := Response{
			statusCode: 200,
		}
		return resp.ToBytes()
	}

	if strings.HasPrefix(request.Path, "/echo/") {
		contentLengthLen := len(strings.Split(request.Path, "/")[2])
		afterEcho := strings.Split(request.Path, "/")[2]

		resp := Response{
			statusCode: 200,
			headers: map[string]string{
				"Content-Type":   "text/plain",
				"Content-Length": strconv.Itoa(contentLengthLen),
			},
			body: []byte(afterEcho),
		}

		if value := request.GetHeader("accept-encoding"); value != "" && value == "gzip" {
			resp.WithEncoding("gzip")
		}

		return resp.ToBytes()
	}

	if strings.HasPrefix(request.Path, "/user-agent") {
		userAgent := request.GetHeader("user-agent")
		if !(userAgent == "") {

			resp := Response{
				statusCode: 200,
				headers: map[string]string{
					"Content-Type":   "text/plain",
					"Content-Length": strconv.Itoa(len(userAgent)),
				},
				body: []byte(userAgent),
			}
			return resp.ToBytes()
		}
	}

	resp := Response{
		statusCode: 404,
	}
	return resp.ToBytes()
}

func (s *Server) readConnection(conn net.Conn, readCallback chan []byte) {
	readBuf := make([]byte, 1024)
	for {
		n, err := conn.Read(readBuf)
		if err != nil && err == io.EOF {
			slog.Info("reached the EOF of the reading connection, stoping the reads...")
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
