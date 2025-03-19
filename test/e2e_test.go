package tests

import (
	"log"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"http_from_scratch"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	s := http_from_scratch.NewServer(":8097")
	go func() {
		err := s.Start()
		if err != nil {
			log.Fatal("failed to start server for tests")
		}
	}()

	code := m.Run()
	os.Exit(code)
}

func TestE2E(t *testing.T) {
	t.Run("should return 200 for valid path", func(t *testing.T) {
		conn, err := net.Dial("tcp", ":8097")
		require.NoError(t, err)
		require.NoError(t, conn.SetDeadline(time.Now().Add(time.Second*5)))

		writeBuf := []byte(
			"GET / HTTP/1.1\r\nHost: localhost:4221\r\nUser-Agent: curl/7.64.1\r\nAccept: */*\r\n\r\n",
		)
		n, err := conn.Write(writeBuf)
		require.NoError(t, err)
		assert.Greater(t, n, 0)

		readBuf := make([]byte, 1024)
		n, err = conn.Read(readBuf)
		require.NoError(t, err)

		assert.Equal(t, readBuf[:n], []byte("HTTP/1.1 200 OK\r\n\r\n"))
	})

	t.Run("should return 404 for invalid path", func(t *testing.T) {
		conn, err := net.Dial("tcp", ":8097")
		require.NoError(t, err)
		require.NoError(t, conn.SetDeadline(time.Now().Add(time.Second*5)))

		writeBuf := []byte(
			"GET /unvalid-path/index.html HTTP/1.1\r\nHost: localhost:4221\r\nUser-Agent: curl/7.64.1\r\nAccept: */*\r\n\r\n",
		)
		n, err := conn.Write(writeBuf)
		require.NoError(t, err)
		assert.Greater(t, n, 0)

		readBuf := make([]byte, 1024)
		n, err = conn.Read(readBuf)
		require.NoError(t, err)

		assert.Equal(t, readBuf[:n], []byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	})

	t.Run("should return 200 to a created /echo path", func(t *testing.T) {
		conn, err := net.Dial("tcp", ":8097")
		require.NoError(t, err)
		require.NoError(t, conn.SetDeadline(time.Now().Add(time.Second*5)))

		writeBuf := []byte(
			"GET /echo/abc HTTP/1.1\r\nHost: localhost:4221\r\nUser-Agent: curl/7.64.1\r\nAccept: */*\r\n\r\n",
		)
		n, err := conn.Write(writeBuf)
		require.NoError(t, err)
		assert.Greater(t, n, 0)

		readBuf := make([]byte, 1024)
		n, err = conn.Read(readBuf)
		require.NoError(t, err)

		assert.Equal(t, readBuf[:n], []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 3\r\n\r\nabc"))
	})
}
