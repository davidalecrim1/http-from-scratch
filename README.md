# HTTP Server from Scratch

Learning the fundamentals of computing and http through it's creation from scratch using Go.

## HTTP

### 200 Response
```bash
// Status line
HTTP/1.1  // HTTP version
200       // Status code
OK        // Optional reason phrase
\r\n      // CRLF that marks the end of the status line

// Headers (empty)
\r\n      // CRLF that marks the end of the headers

// Response body (empty)
```

#### GET Request
```bash
// Request line
GET                          // HTTP method
/index.html                  // Request target
HTTP/1.1                     // HTTP version
\r\n                         // CRLF that marks the end of the request line

// Headers
Host: localhost:4221\r\n     // Header that specifies the server's host and port
User-Agent: curl/7.64.1\r\n  // Header that describes the client's user agent
Accept: */*\r\n              // Header that specifies which media types the client can accept
\r\n                         // CRLF that marks the end of the headers

// Request body (empty)
```

### CRLF
CR and LF are control characters or bytecode that can be used to mark a line break in a text file.

A CR immediately followed by a LF (CRLF, \r\n, or 0x0D0A) moves the cursor to the beginning of the line and then down to the next line.

[CRLF Spec](https://developer.mozilla.org/en-US/docs/Glossary/CRLF).

