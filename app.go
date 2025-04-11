package fast

import (
	"bytes"
	"io"
	"log"
	"log/slog"
	"net"
	"time"
)

type Config struct {
	ConnectionTimeout int // seconds
}

type App struct {
	config      Config
	addr        string
	ln          net.Listener
	middlewares []Middleware
	routes      map[string]map[string][]Handler // "method" -> "path"
}

type Handler func(*Ctx) error

type Middleware func(Handler) Handler

func New(c Config) *App {
	if c.ConnectionTimeout == 0 {
		c.ConnectionTimeout = 5
	}

	return &App{
		config: c,
		routes: make(map[string]map[string][]Handler),
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
		conn, err := app.ln.Accept()
		if err != nil {
			slog.Error("failed to create a new connection, stoping the listener...", "error", err)
			app.ln.Close()
			return
		}

		go app.handleConnection(conn)
	}
}

func (app *App) handleConnection(conn net.Conn) {
	err := conn.SetDeadline(time.Now().Add(time.Second * time.Duration(app.config.ConnectionTimeout)))
	if err != nil {
		slog.Error("failed to set deadline for the connection", "error", err)
	}
	defer conn.Close()

	for {
		requestBytes := app.readConnection(conn)
		if len(requestBytes) == 0 {
			slog.Debug("received an empty request. skipping the read")
			continue
		}

		request, err := NewRequest(requestBytes)
		if err != nil {
			slog.Debug("failed to parse the request", "error", err)
			return
		}

		response := app.handleRequest(&request)
		_, err = conn.Write(response)
		if err != nil {
			slog.Error("failed to write response in the connection", "error", err)
			return
		}

		// TODO: Figure out how to actually handle this.
		// if !shouldKeepAlive {
		// 	return
		// }
	}
}

func (app *App) handleRequest(request *Request) (response []byte) {
	if method, ok := app.routes[request.Method]; ok {
		if handlers, ok := method[request.Path]; ok {
			ctx := &Ctx{Request: request}

			for _, handler := range handlers {
				if err := handler(ctx); err != nil {
					return NewResponse(StatusInternalServerError, nil, []byte{}).ToBytes()
				}
			}

			return ctx.Response.ToBytes()
		}
	}

	return NewResponse(StatusNotFound, nil, []byte{}).ToBytes()
}

func (app *App) readConnection(conn net.Conn) []byte {
	var buf bytes.Buffer
	readBuf := make([]byte, 4096)
	for {
		n, err := conn.Read(readBuf)
		if err != nil && err == io.EOF {
			slog.Info("reached the EOF of the reading connection, stoping the reads...")
			break
		}
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			slog.Warn("read deadline exceeded", "remote", conn.RemoteAddr())
			break
		}

		buf.Write(readBuf[:n])

		if bytes.Contains(buf.Bytes(), []byte("\r\n\r\n")) {
			break
		}
	}

	return buf.Bytes()
}

func (app *App) Get(path string, handlers ...Handler) Router {
	return app.Add(methodGet, path, handlers...)
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

func (app *App) Use(middleware Middleware) {
	app.middlewares = append(app.middlewares, middleware)
}

func (app *App) Shutdown() error {
	return app.ln.Close()
}
