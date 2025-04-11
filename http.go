package fast

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Request struct {
	Method   string
	Path     string
	Protocol string
	headers  map[string]string
	Body     []byte
}

func NewRequest(request []byte) (Request, error) {
	lines := strings.Split(string(request), "\r\n")
	if len(lines) < 1 {
		return Request{}, errors.New("invalid request")
	}

	// Parse the first line (method, path, protocol)
	headerFirstLine := strings.Split(lines[0], " ")
	if len(headerFirstLine) < 3 {
		return Request{}, errors.New("invalid request line")
	}

	req := Request{
		Method:   headerFirstLine[0],
		Path:     headerFirstLine[1],
		Protocol: headerFirstLine[2],
		headers:  make(map[string]string),
	}

	// Parse headers
	for _, line := range lines[1:] {
		if line == "" { // End of headers
			break
		}
		headerParts := strings.SplitN(line, ": ", 2)
		if len(headerParts) == 2 {
			req.headers[strings.ToLower(headerParts[0])] = headerParts[1]
		}
	}

	// Set the body (last part of the request)
	req.Body = []byte(lines[len(lines)-1])
	return req, nil
}

func (r *Request) GetHeader(key string) string {
	value, ok := r.headers[strings.ToLower(key)]
	if !ok {
		return ""
	}

	return value
}

type Response struct {
	statusCode int
	headers    map[string]string
	body       []byte
}

func NewResponse(statusCode int, headers map[string]string, body []byte) *Response {
	if headers == nil {
		headers = make(map[string]string)
	}

	resp := &Response{
		statusCode: statusCode,
		headers:    headers,
		body:       body,
	}

	return resp
}

func (r *Response) addContentLenghtHeader() {
	r.SetHeader("Content-Length", strconv.Itoa(len(r.body)))
}

func (r *Response) ToBytes() []byte {
	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", r.statusCode, StatusText[r.statusCode])

	body := ""
	if r.body != nil {
		body = string(r.body)
	}

	r.addContentLenghtHeader()

	headers := ""
	if r.headers != nil {
		for key, value := range r.headers {
			headers += fmt.Sprintf("%s: %s\r\n", key, value)
		}
	}

	return []byte(statusLine + headers + "\r\n" + body)
}

func (r *Response) GetBody() []byte {
	return r.body
}

func (r *Response) SetBody(body []byte) {
	r.body = body
}

func (r *Response) SetBodyString(body string) {
	r.LoadStatus()
	r.SetBody([]byte(body))
}

func (r *Response) SetHeader(key, value string) {
	r.headers[strings.ToLower(key)] = value
}

func (r *Response) LoadStatus() {
	if r.statusCode == 0 {
		r.statusCode = 200
	}
}

func (r *Response) SetStatus(status int) {
	r.statusCode = status
}
