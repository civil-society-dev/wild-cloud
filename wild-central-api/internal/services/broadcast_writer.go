package services

import (
	"bytes"
	"os"

	"github.com/wild-cloud/wild-central/daemon/internal/operations"
)

// broadcastWriter writes output to both a file and broadcasts to SSE clients
type broadcastWriter struct {
	file        *os.File
	broadcaster *operations.Broadcaster
	opID        string
	buffer      *bytes.Buffer
}

// newBroadcastWriter creates a writer that writes to file and broadcasts
func newBroadcastWriter(file *os.File, broadcaster *operations.Broadcaster, opID string) *broadcastWriter {
	return &broadcastWriter{
		file:        file,
		broadcaster: broadcaster,
		opID:        opID,
		buffer:      &bytes.Buffer{},
	}
}

// Write implements io.Writer interface
func (w *broadcastWriter) Write(p []byte) (n int, err error) {
	// Write to file first
	n, err = w.file.Write(p)
	if err != nil {
		return n, err
	}

	// Buffer the data and broadcast complete lines
	if w.broadcaster != nil {
		w.buffer.Write(p)

		// Extract and broadcast complete lines
		for {
			line, err := w.buffer.ReadBytes('\n')
			if err != nil {
				// No complete line, put back what we read and break
				w.buffer.Write(line)
				break
			}
			// Broadcast the line without the trailing newline
			if len(line) > 0 && line[len(line)-1] == '\n' {
				line = line[:len(line)-1]
			}
			w.broadcaster.Publish(w.opID, line)
		}
	}

	return n, nil
}

// Flush broadcasts any remaining buffered data
func (w *broadcastWriter) Flush() {
	if w.broadcaster != nil && w.buffer.Len() > 0 {
		// Broadcast the remaining incomplete line
		w.broadcaster.Publish(w.opID, w.buffer.Bytes())
		w.buffer.Reset()
	}
}
