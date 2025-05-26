package aio

import (
	"io"
	"log/slog"
)

// Close will close the given closer and log the error if required.
func Close(c io.Closer) {
	if err := c.Close(); err != nil {
		slog.Error("close failed", slog.Any("error", err))
	}
}
