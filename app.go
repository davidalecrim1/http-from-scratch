package fast

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	IdleTimeout time.Duration // seconds
}

type App struct {
	config      Config
	addr        string
	ln          net.Listener
	middlewares []Handler
	routes      map[string]map[string][]Handler // "method" -> "path"
	quit        chan struct{}
	wg          sync.WaitGroup
	activeConns atomic.Int64
}

type Handler func(*Ctx) error

type Middleware func() Handler

func New(c Config) *App {
	if c.IdleTimeout == 0 {
		c.IdleTimeout = time.Second * 120
	}

	return &App{
		config: c,
		routes: make(map[string]map[string][]Handler),
		quit:   make(chan struct{}),
	}
}

// Expected -> :8090
func (app *App) Listen(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("failed to bind to the port 8097")
	}

	app.addr = addr
	app.ln = ln
	app.acceptConnections()
	return nil
}

func (app *App) acceptConnections() {
	for {
		select {
		case <-app.quit:
			// TODO: I could add some control to wait the current connections to end before this.
			slog.Debug("received a quit command, shutting down new connections...")
			return
		default:
			conn, err := app.ln.Accept()
			if err != nil {
				slog.Warn("failed to create a new connection, stoping the listener...", "error", err)
				continue
			}

			app.activeConns.Add(1)
			app.wg.Add(1)
			go app.handleConnection(conn)
		}
	}
}

func (app *App) handleConnection(conn net.Conn) {
	defer func() {
		slog.Debug("closing the connection given the keep alive header is not present.")

		conn.Close()
		app.activeConns.Add(-1)
		app.wg.Done()
	}()

	err := conn.SetDeadline(time.Now().Add(app.config.IdleTimeout))
	if err != nil {
		slog.Error("failed to set deadline for the connection", "error", err)
		return
	}

	err = app.resetConnTimeout(conn)
	if err != nil {
		return
	}

	for {
		requestBytes, err := app.readConnection(conn)
		if err != nil {
			// TODO: Improve this in the future.
			if strings.Contains(err.Error(), "the read deadline was exceeded") {
				slog.Debug("the timeout of the connection was reached, closing it", "error", err)
				return
			}

			slog.Error("received an error, closing the connection...", "error", err)
			return
		}

		if len(requestBytes) == 0 {
			slog.Debug("received an empty request, skipping the reads...")
			continue
		}

		request, err := NewRequest(requestBytes)
		if err != nil {
			slog.Debug("failed to parse the request", "error", err)
			return
		}

		response := app.handleRequest(request)
		_, err = conn.Write(response)
		if err != nil {
			slog.Error("failed to write response in the connection", "error", err)
			return
		}

		if app.shouldKeepAlive(request) {
			app.resetConnTimeout(conn)
			continue
		} else {
			app.resetConnTimeout(conn)
			return
		}
	}
}

func (app *App) resetConnTimeout(conn net.Conn) error {
	err := conn.SetDeadline(time.Now().Add(app.config.IdleTimeout))
	if err != nil {
		slog.Error("failed to set deadline for the connection", "error", err)
		return err
	}
	return nil
}

func (app *App) shouldKeepAlive(req *Request) bool {
	return (req.GetHeader("connection") != "close")
}

func (app *App) handleRequest(request *Request) (response []byte) {
	if method, ok := app.routes[request.Method]; ok {
		if routeHandlers, ok := method[request.Path]; ok {
			allHandlers := append(app.middlewares, routeHandlers...)

			ctx := &Ctx{
				Request:  request,
				Response: NewResponse(200, nil, nil),
				handlers: allHandlers,
				index:    -1, // because it will be incremented in each c.Next(), therefore the first will be 0.
			}

			if err := ctx.Next(); err != nil {
				return NewResponse(StatusInternalServerError, nil, []byte{}).ToBytes()
			}

			if app.shouldKeepAlive(request) {
				ctx.Set("connection", "keep-alive")
			}

			return ctx.Response.ToBytes()
		}
	}

	return NewResponse(StatusNotFound, nil, []byte{}).ToBytes()
}

func (app *App) readConnection(conn net.Conn) ([]byte, error) {
	var buf bytes.Buffer
	readBuf := make([]byte, 4096)
	for {
		n, err := conn.Read(readBuf)
		if err != nil && err == io.EOF {
			slog.Debug("reached the EOF of the reading connection, stoping the reads...")
			break
		}
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			slog.Warn("read deadline exceeded", "remote", conn.RemoteAddr())
			return nil, fmt.Errorf("the read deadline was exceeded: %v", err)
		}

		buf.Write(readBuf[:n])

		if bytes.Contains(buf.Bytes(), []byte("\r\n\r\n")) {
			break
		}
	}

	return buf.Bytes(), nil
}

func (app *App) Get(path string, handlers ...Handler) Router {
	return app.Add(MethodGet, path, handlers...)
}

func (app *App) Add(method, path string, handlers ...Handler) Router {
	app.register(method, path, handlers...)
	return app
}

func (app *App) register(method, path string, handlers ...Handler) {
	if len(handlers) == 0 {
		log.Panic("missing handler when registering a route")
	}

	if path == "" {
		path = "/"
	}

	if path[0] != '/' {
		path = "/" + path
	}

	app.addRoute(method, path, handlers...)
}

func (app *App) addRoute(method string, path string, handlers ...Handler) {
	if app.routes[method] == nil {
		app.routes[method] = make(map[string][]Handler)
	}

	app.routes[method][path] = handlers
}

func (app *App) Use(middleware Handler) {
	app.middlewares = append(app.middlewares, middleware)
}

func (app *App) Shutdown(force bool) error {
	slog.Debug("amount of active connections BEFORE closing", "activeConns", app.activeConns.Load())

	close(app.quit)

	if !force {
		app.wg.Wait()
	}

	err := app.ln.Close()
	slog.Debug("amount of active connections AFTER closing", "activeConns", app.activeConns.Load())
	return err
}
