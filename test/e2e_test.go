package tests

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"fast"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	code := m.Run()
	os.Exit(code)
}

func TestDefaultHandler(t *testing.T) {
	app := fast.New(
		fast.Config{},
	)

	app.Get("/", func(c *fast.Ctx) error {
		return c.SendString("OK")
	})

	go func() {
		err := app.Listen(":8097")
		if err != nil {
			log.Fatal("failed to start server for tests")
		}
	}()

	t.Cleanup(func() {
		if err := app.Shutdown(); err != nil {
			slog.Error("failed to shutdown the server")
		}
	})

	t.Run("should return 200", func(t *testing.T) {
		t.Parallel()

		conn, err := net.Dial("tcp", ":8097")
		require.NoError(t, err)
		require.NoError(t, conn.SetDeadline(time.Now().Add(time.Second*5)))

		writeBuf := []byte(
			"GET / HTTP/1.1\r\nHost: localhost:8097\r\nUser-Agent: curl/7.64.1\r\nAccept: */*\r\n\r\n",
		)
		n, err := conn.Write(writeBuf)
		require.NoError(t, err)
		assert.Greater(t, n, 0)

		readBuf := make([]byte, 1024)
		n, err = conn.Read(readBuf)
		require.NoError(t, err)

		assert.Equal(t, readBuf[:n], []byte("HTTP/1.1 200 OK\r\n\r\nOK"))
	})

	t.Run("should handle concurrent requests with 200 response for each", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}
		totalRequests := 20
		wg.Add(totalRequests)

		for range totalRequests {
			go func() {
				defer wg.Done()
				conn, err := net.Dial("tcp", ":8097")
				require.NoError(t, err)
				require.NoError(t, conn.SetDeadline(time.Now().Add(time.Second*5)))

				writeBuf := []byte(
					"GET / HTTP/1.1\r\nHost: localhost:8087\r\nUser-Agent: curl/7.64.1\r\nAccept: */*\r\n\r\n",
				)
				n, err := conn.Write(writeBuf)
				require.NoError(t, err)
				assert.Greater(t, n, 0)

				readBuf := make([]byte, 1024)
				n, err = conn.Read(readBuf)
				require.NoError(t, err)

				assert.Equal(t, readBuf[:n], []byte("HTTP/1.1 200 OK\r\n\r\nOK"))
			}()
		}

		wg.Wait()
		require.True(t, true)
	})
}

func TestE2E(t *testing.T) {
	t.Run("should return 404 for invalid path", func(t *testing.T) {
		t.Skip("rewrite this later")

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
		t.Skip("rewrite this later")

		testCases := []string{"one", "two", "three", "four", "five"}

		for _, testCase := range testCases {
			conn, err := net.Dial("tcp", ":8097")
			require.NoError(t, err)
			defer conn.Close()

			require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))

			var writeBuf bytes.Buffer
			fmt.Fprintf(&writeBuf,
				"GET /echo/%s HTTP/1.1\r\nHost: localhost:4221\r\nUser-Agent: curl/7.64.1\r\nAccept: */*\r\n\r\n",
				testCase,
			)
			n, err := conn.Write(writeBuf.Bytes())
			require.NoError(t, err)
			assert.Greater(t, n, 0)

			readBuf := make([]byte, 1024)
			n, err = conn.Read(readBuf)
			require.NoError(t, err)

			expectedResponse := fmt.Sprintf(
				"HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
				len(testCase), testCase,
			)
			assert.Equal(t, expectedResponse, string(readBuf[:n]))
		}
	})

	t.Run("should return 200 to a created /echo path with gzip encoding", func(t *testing.T) {
		t.Skip("rewrite this later")

		testCases := []string{"one", "two", "three", "four", "five"}

		for _, testCase := range testCases {
			conn, err := net.Dial("tcp", ":8097")
			require.NoError(t, err)
			defer conn.Close()

			require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))

			var writeBuf bytes.Buffer
			fmt.Fprintf(&writeBuf,
				"GET /echo/%s HTTP/1.1\r\nHost: localhost:4221\r\nUser-Agent: curl/7.64.1\r\nAccept: */*\r\nAccept-Encoding: gzip\r\n\r\n",
				testCase,
			)
			n, err := conn.Write(writeBuf.Bytes())
			require.NoError(t, err)
			assert.Greater(t, n, 0)

			readBuf := make([]byte, 1024)
			n, err = conn.Read(readBuf)
			require.NoError(t, err)

			response := string(readBuf[:n])

			assert.Contains(t, response, "Content-Encoding: gzip")

			bodyIndex := strings.Index(response, "\r\n\r\n")
			require.Greater(t, bodyIndex, -1)

			// bodyIndex + 4 skips \r\n\r\n (end of headers).
			encodedBody := readBuf[bodyIndex+4 : n]

			reader, err := gzip.NewReader(bytes.NewReader(encodedBody))
			require.NoError(t, err)
			defer reader.Close()

			decompressedBody, err := io.ReadAll(reader)
			require.NoError(t, err)

			assert.Equal(t, testCase, string(decompressedBody))
		}
	})

	t.Run("should return 400 to a bad request", func(t *testing.T) {
		t.Skip("rewrite this later")

		conn, err := net.Dial("tcp", ":8097")
		require.NoError(t, err)
		require.NoError(t, conn.SetDeadline(time.Now().Add(time.Second*5)))

		writeBuf := []byte(
			"INVALID",
		)
		n, err := conn.Write(writeBuf)
		require.NoError(t, err)
		assert.Greater(t, n, 0)

		readBuf := make([]byte, 1024)
		n, err = conn.Read(readBuf)
		require.NoError(t, err)

		assert.Equal(t, readBuf[:n], []byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
	})

	t.Run("should parse a sent user agent", func(t *testing.T) {
		t.Skip("rewrite this later")

		conn, err := net.Dial("tcp", ":8097")
		require.NoError(t, err)
		require.NoError(t, conn.SetDeadline(time.Now().Add(time.Second*5)))

		writeBuf := []byte(
			"GET /user-agent HTTP/1.1\r\nHost: localhost:4221\r\nUser-Agent: foobar/1.2.3\r\n\r\nAccept: */*\r\n\r\n",
		)
		n, err := conn.Write(writeBuf)
		require.NoError(t, err)
		assert.Greater(t, n, 0)

		readBuf := make([]byte, 1024)
		n, err = conn.Read(readBuf)
		require.NoError(t, err)

		expected := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 12\r\n\r\nfoobar/1.2.3")
		assert.Equal(t, expected, readBuf[:n])
	})
}
