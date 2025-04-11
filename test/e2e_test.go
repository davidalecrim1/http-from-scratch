package tests

import (
	"io"
	"log"
	"log/slog"
	"net/http"
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

func TestMain(m *testing.M) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	code := m.Run()
	os.Exit(code)
}

func TestHandler(t *testing.T) {
	app := fast.New(
		fast.Config{},
	)

	app.Get("/", func(c *fast.Ctx) error {
		return c.SendString("OK")
	})

	customMiddleware := func(c *fast.Ctx) error {
		err := c.Next()
		c.Set("x-middleware-header", "true")
		return err
	}

	app.Get("/custom-middleware", customMiddleware, func(c *fast.Ctx) error {
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

	t.Run("should return 200 for configured handler", func(t *testing.T) {
		t.Parallel()

		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		resp, err := client.Get("http://localhost:8097")
		require.NoError(t, err)
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, []byte("OK"), respBody)
		assert.NotEmpty(t, resp.Header.Get("Content-Length"))
	})

	t.Run("should support wrapped handlers WITHOUT global middlewares", func(t *testing.T) {
		t.Parallel()

		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		resp, err := client.Get("http://localhost:8097/custom-middleware")
		require.NoError(t, err)
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, []byte("OK"), respBody)
		assert.NotEmpty(t, resp.Header.Get("Content-Length"))
		assert.Equal(t, "true", resp.Header.Get("x-middleware-header"))
	})

	t.Run("should handle concurrent requests with 200 response for each", func(t *testing.T) {
		t.Parallel()

		wg := sync.WaitGroup{}
		totalRequests := 20
		wg.Add(totalRequests)

		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		for range totalRequests {
			go func() {
				defer wg.Done()

				resp, err := client.Get("http://localhost:8097")
				require.NoError(t, err)
				defer resp.Body.Close()

				respBody, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				assert.Equal(t, []byte("OK"), respBody)
				assert.NotEmpty(t, resp.Header.Get("Content-Length"))
			}()
		}

		wg.Wait()
		require.True(t, true)
	})
}

func TestMiddlewares(t *testing.T) {
	app := fast.New(
		fast.Config{},
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

	go func() {
		err := app.Listen(":8098")
		if err != nil {
			log.Fatal("failed to start server for tests")
		}
	}()

	t.Cleanup(func() {
		if err := app.Shutdown(); err != nil {
			slog.Error("failed to shutdown the server")
		}
	})

	t.Run("should handle middleware for CORS", func(t *testing.T) {
		t.Parallel()

		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		resp, err := client.Get("http://localhost:8098/middleware")
		require.NoError(t, err)
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.Equal(t, respBody, []byte("OK"))

		assert.Equal(t, fast.StatusOK, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET,POST,PUT,DELETE,OPTIONS", resp.Header.Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Content-Type, Authorization", resp.Header.Get("Access-Control-Allow-Headers"))
	})

	t.Run("should handle middleware for recovery of panics", func(t *testing.T) {
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		resp, err := client.Get("http://localhost:8098/recovery")
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

	go func() {
		err := app.Listen(":8099")
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
			Timeout: 360 * time.Second,
		}

		resp, err := client.Get("http://localhost:8099")
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
