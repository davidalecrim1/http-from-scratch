package compress

import (
	"bytes"
	"compress/gzip"
	"log/slog"

	"fast"
)

func New() fast.Handler {
	return func(c *fast.Ctx) error {
		if err := c.Next(); err != nil {
			return err
		}

		encoding := c.Get("Accept-Encoding")
		if encoding == "gzip" {
			var buffer bytes.Buffer
			w := gzip.NewWriter(&buffer)

			respBody := c.Response.GetBody()
			_, err := w.Write(respBody)
			if err != nil {
				slog.Error("failed to write to gzip writer", "error", err)
				return err
			}

			if err := w.Close(); err != nil {
				slog.Error("failed to close gzip writer", "error", err)
				return err
			}

			slog.Debug("gzip encoding applied",
				"originalSize", len(respBody),
				"compressedSize", buffer.Len(),
			)

			c.Set("Content-Encoding", "gzip")
			c.Send(buffer.Bytes())
		}

		return nil
	}
}
