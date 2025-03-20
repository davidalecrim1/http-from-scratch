# HTTP Server from Scratch

Building an HTTP server from scratch in Go to understand the fundamentals of computing and the HTTP protocol.

## Features
- Minimal HTTP server implementation in Go
- Handles basic HTTP requests and responses
- Customizable request parsing and response handling

## Getting Started

### Prerequisites
- Go 1.23+
- `make` (optional, for running and testing)

### Running the Server
You can start the server using:
```sh
make run
```
Or manually run:
```sh
go run ./cmd/main.go
```
The server will start listening on `localhost:4221`.

### Running Tests
To run the test suite:
```sh
make test
```
Or manually:
```sh
go test ./...
```

## HTTP Overview

### 200 OK Response Example
```http
HTTP/1.1 200 OK
// Headers (empty)
// Response body (empty)
```

### Example GET Request
```http
GET /index.html HTTP/1.1
Host: localhost:4221
User-Agent: curl/7.64.1
Accept: */*
// Request body (empty)
```

### Understanding CRLF
Carriage Return (CR) and Line Feed (LF) are control characters used to mark line breaks.

A CR followed by an LF (CRLF, `\r\n`, or `0x0D0A`) moves the cursor to the beginning of the line and then down to the next line.

For more details, see the [CRLF Spec](https://developer.mozilla.org/en-US/docs/Glossary/CRLF).


## Compression

An HTTP client uses the Accept-Encoding header to specify the compression schemes it supports. In the following example, the client specifies that it supports the gzip compression scheme:

```http
GET /echo/foo HTTP/1.1
Host: localhost:4221
User-Agent: curl/7.81.0
Accept: */*
Accept-Encoding: gzip  // Client specifies it supports the gzip compression scheme.
```

The server then chooses one of the compression schemes listed in Accept-Encoding and compresses the response body with it.

Then, the server sends a response with the compressed body and a Content-Encoding header. Content-Encoding specifies the compression scheme that was used.