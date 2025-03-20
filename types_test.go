package http_from_scratch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequest(t *testing.T) {
	t.Run("should create a valid request", func(t *testing.T) {
		request := []byte("GET /user-agent HTTP/1.1\r\nHost: localhost:4221\r\nUser-Agent: foobar/1.2.3\r\nAccept: */*\r\n\r\nfoobar")
		createdRequest, err := NewRequest(request)
		require.NoError(t, err)

		assert.Equal(t, createdRequest.Method, "GET")
		assert.Equal(t, createdRequest.Path, "/user-agent")
		assert.Equal(t, createdRequest.Protocol, "HTTP/1.1")
		assert.Equal(t, createdRequest.GetHeader("Host"), "localhost:4221")
		assert.Equal(t, createdRequest.GetHeader("User-Agent"), "foobar/1.2.3")
		assert.Equal(t, createdRequest.GetHeader("Accept"), "*/*")
		assert.Equal(t, string(createdRequest.Body), "foobar")
	})
}
