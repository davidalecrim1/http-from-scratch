package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"net/http/httptrace"
	"os"
	"sync"
	"testing"
	"time"

	"fast/middleware/compress"
	"fast/middleware/cors"
	"fast/middleware/recovery"

	"fast"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getRandomPort() int {
	return 8000 + rand.Intn(1000)
}

func TestMain(m *testing.M) {
	slog.SetLogLoggerLevel(slog.LevelWarn)

	code := m.Run()
	os.Exit(code)
}

func TestHandler(t *testing.T) {
	app := fast.New(
		fast.Config{
			IdleTimeout: 5 * time.Second,
		},
	)

	app.Get("/", func(c *fast.Ctx) error {
		return c.SendString("OK")
	})

	app.Get("/get-json", func(c *fast.Ctx) error {
		return c.JSON(fast.Map{
			"message": "Hello World",
		})
	})

	customMiddleware := func(c *fast.Ctx) error {
		err := c.Next()
		c.Set("x-middleware-header", "true")
		return err
	}

	app.Get("/custom-middleware", customMiddleware, func(c *fast.Ctx) error {
		return c.SendString("OK")
	})

	port := getRandomPort()
	go func() {
		err := app.Listen(fmt.Sprintf(":%d", port))
		if err != nil {
			log.Fatal("failed to start server for tests")
		}
	}()

	t.Cleanup(func() {
		if err := app.Shutdown(); err != nil {
			slog.Error("failed to shutdown the server")
		}
	})

	t.Run("should return 200 for configured handler", func(t *testing.T) {
		t.Parallel()

		client := &http.Client{
			Timeout: 3 * time.Second,
		}

		resp, err := client.Get(fmt.Sprintf("http://localhost:%d", port))
		require.NoError(t, err)
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, []byte("OK"), respBody)
		assert.NotEmpty(t, resp.Header.Get("Content-Length"))
		assert.Equal(t, resp.Header.Get("Connection"), "keep-alive")
		assert.Equal(t, fast.StatusOK, resp.StatusCode)
	})

	t.Run("should return 200 with a valid JSON struct", func(t *testing.T) {
		t.Parallel()

		client := &http.Client{
			Timeout: 3 * time.Second,
		}

		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/get-json", port))
		require.NoError(t, err)
		defer resp.Body.Close()

		raw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var respBody map[string]any
		err = json.Unmarshal(raw, &respBody)
		require.NoError(t, err)

		assert.Equal(t, "Hello World", respBody["message"])
		assert.NotEmpty(t, resp.Header.Get("Content-Length"))
		assert.Equal(t, fast.StatusOK, resp.StatusCode)
	})

	t.Run("should return 404 for unexisting path", func(t *testing.T) {
		t.Parallel()

		client := &http.Client{
			Timeout: 3 * time.Second,
		}

		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/invalid-path", port))
		require.NoError(t, err)
		defer resp.Body.Close()

		raw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Len(t, raw, 0)
		assert.Equal(t, fast.StatusNotFound, resp.StatusCode)
	})

	t.Run("should return 200 with support wrapped handlers WITHOUT global middlewares", func(t *testing.T) {
		t.Parallel()

		client := &http.Client{
			Timeout: 3 * time.Second,
		}

		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/custom-middleware", port))
		require.NoError(t, err)
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, []byte("OK"), respBody)
		assert.NotEmpty(t, resp.Header.Get("Content-Length"))
		assert.Equal(t, "true", resp.Header.Get("x-middleware-header"))
		assert.Equal(t, fast.StatusOK, resp.StatusCode)
	})

	t.Run("should handle concurrent requests with 200 response for each", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}
		totalRequests := 20
		wg.Add(totalRequests)

		client := &http.Client{
			Timeout: 3 * time.Second,
		}

		for range totalRequests {
			go func() {
				defer wg.Done()

				resp, err := client.Get(fmt.Sprintf("http://localhost:%d", port))
				require.NoError(t, err)
				defer resp.Body.Close()

				respBody, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				assert.Equal(t, []byte("OK"), respBody)
				assert.NotEmpty(t, resp.Header.Get("Content-Length"))
				assert.Equal(t, fast.StatusOK, resp.StatusCode)
			}()
		}

		wg.Wait()
		require.True(t, true)
	})
}

func TestConnectionTimeout(t *testing.T) {
	app := fast.New(
		fast.Config{
			IdleTimeout: 5 * time.Second,
		},
	)

	app.Get("/", func(c *fast.Ctx) error {
		return c.SendString("OK")
	})

	app.Get("/get-json", func(c *fast.Ctx) error {
		return c.JSON(fast.Map{
			"message": "Hello World",
		})
	})

	customMiddleware := func(c *fast.Ctx) error {
		err := c.Next()
		c.Set("x-middleware-header", "true")
		return err
	}

	app.Get("/custom-middleware", customMiddleware, func(c *fast.Ctx) error {
		return c.SendString("OK")
	})

	app.Get("/handle-connection-close", func(c *fast.Ctx) error {
		c.Request.SetHeader("connection", "close") // manually setting the request header
		return c.SendString("OK")
	})

	port := getRandomPort()
	go func() {
		slog.Info("TestConnectionTimeout - Listening on", "port", port)

		err := app.Listen(fmt.Sprintf(":%d", port))
		if err != nil {
			log.Fatal("failed to start server for tests")
		}
	}()

	t.Run("should handle the connection close header", func(t *testing.T) {
		t.Parallel()
		// five times to ensure a new connection is created each time.
		for range 5 {
			client := &http.Client{
				Timeout: 3 * time.Second,
			}

			req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/handle-connection-close", port), nil)
			trace := &httptrace.ClientTrace{
				GotConn: func(gci httptrace.GotConnInfo) {
					require.Equal(t, false, gci.Reused)
				},
			}

			req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

			resp, err := client.Do(req)
			require.NoError(t, err)

			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			resp.Body.Close()

			assert.Equal(t, []byte("OK"), respBody)
			assert.Equal(t, fast.StatusOK, resp.StatusCode)
		}
	})
}

func TestMiddlewares(t *testing.T) {
	app := fast.New(
		fast.Config{
			IdleTimeout: 5 * time.Second,
		},
	)

	app.Use(cors.New())
	app.Use(recovery.New())
	app.Use(compress.New())

	app.Get("/middleware", func(c *fast.Ctx) error {
		return c.SendString("OK")
	})

	app.Get("/recovery", func(c *fast.Ctx) error {
		panic("expected panic to be recovered")
	})

	port := getRandomPort()
	go func() {
		err := app.Listen(fmt.Sprintf(":%d", port))
		if err != nil {
			log.Fatal("failed to start server for tests")
		}
	}()

	t.Cleanup(func() {
		if err := app.Shutdown(); err != nil {
			slog.Error("failed to shutdown the server")
		}
	})

	t.Run("should return 200 after handle middleware for CORS", func(t *testing.T) {
		t.Parallel()

		client := &http.Client{
			Timeout: 3 * time.Second,
		}

		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/middleware", port))
		require.NoError(t, err)
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, respBody, []byte("OK"))

		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET,POST,PUT,DELETE,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Content-Type, Authorization", resp.Header.Get("Access-Control-Allow-Headers"))

		assert.Equal(t, fast.StatusOK, resp.StatusCode)
	})

	t.Run("should handle middleware for recovery of panics", func(t *testing.T) {
		client := &http.Client{
			Timeout: 3 * time.Second,
		}

		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/recovery", port))
		require.NoError(t, err)
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, respBody, []byte("received an error while executing"))
		assert.Equal(t, fast.StatusServiceUnavailable, resp.StatusCode)
	})
}

func TestMiddleware_Compress(t *testing.T) {
	app := fast.New(
		fast.Config{},
	)

	app.Use(compress.New())

	app.Get("/", func(c *fast.Ctx) error {
		return c.SendString("OK")
	})

	port := getRandomPort()
	go func() {
		slog.Info("TestMiddleware_Compress - Listening on", "port", port)

		err := app.Listen(fmt.Sprintf(":%d", port))
		if err != nil {
			log.Fatal("failed to start server for tests")
		}
	}()

	t.Cleanup(func() {
		if err := app.Shutdown(); err != nil {
			slog.Error("failed to shutdown the server")
		}
	})

	t.Run("should handle middleware with compression", func(t *testing.T) {
		t.Parallel()

		client := &http.Client{
			Timeout: time.Second * 3,
		}

		resp, err := client.Get(fmt.Sprintf("http://localhost:%d", port))
		require.NoError(t, err)
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, respBody, []byte("OK"))

		assert.Equal(t, fast.StatusOK, resp.StatusCode)
		// TODO: Understand why the content length is removed by the client http.
		// To better understand this, I would need a playground.

		// This seems to be known by the Go team.
		// https://github.com/golang/go/blob/9f13665088012298146c573bc2a7255b1caf2750/src/net/http/fs.go#L377
	})
}
