package logapi

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis/response"
)

const (
	pollInterval = 500 * time.Millisecond
	heartbeat    = 15 * time.Second
)

// InstallHandler registers the real-time log stream endpoint.
func InstallHandler(group *gin.RouterGroup, logPath string) {
	group.GET("/system/logs/stream", streamLogs(logPath))
}

func streamLogs(logPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.TrimSpace(logPath) == "" {
			response.WriteError(c, http.StatusBadRequest, response.ErrSystemLogFileUnavailable())
			return
		}
		file, err := os.Open(logPath)
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrSystemLogFileUnavailable())
			return
		}
		defer func() {
			if file != nil {
				_ = file.Close()
			}
		}()
		if _, err := file.Seek(0, io.SeekEnd); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrSystemLogFileUnavailable())
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		c.Status(http.StatusOK)
		c.Writer.Flush()
		_, _ = c.Writer.WriteString("event: ready\ndata: {}\n\n")
		c.Writer.Flush()

		reader := bufio.NewReader(file)
		offset, _ := file.Seek(0, io.SeekCurrent)
		lastHeartbeat := time.Now()
		pending := ""

		for {
			select {
			case <-c.Request.Context().Done():
				return
			default:
			}

			line, readErr := reader.ReadString('\n')
			if line != "" {
				pending += line
			}
			if strings.HasSuffix(pending, "\n") {
				offset += int64(len(pending))
				if !writeLog(c, strings.TrimRight(pending, "\r\n")) {
					return
				}
				pending = ""
			}

			if readErr == nil {
				continue
			}
			if readErr != io.EOF {
				return
			}

			if time.Since(lastHeartbeat) >= heartbeat {
				if _, err := c.Writer.WriteString(": heartbeat\n\n"); err != nil {
					return
				}
				c.Writer.Flush()
				lastHeartbeat = time.Now()
			}

			changed, statErr := logFileChanged(file, logPath, offset)
			if statErr != nil || changed {
				_ = file.Close()
				file = nil
				pending = ""
				offset = 0
				for {
					select {
					case <-c.Request.Context().Done():
						return
					case <-time.After(pollInterval):
					}
					next, openErr := os.Open(logPath)
					if openErr != nil {
						continue
					}
					file = next
					reader = bufio.NewReader(file)
					break
				}
			}

			select {
			case <-c.Request.Context().Done():
				return
			case <-time.After(pollInterval):
			}
		}
	}
}

func logFileChanged(file *os.File, path string, offset int64) (bool, error) {
	current, err := file.Stat()
	if err != nil {
		return true, err
	}
	next, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return !os.SameFile(current, next) || next.Size() < offset, nil
}

func writeLog(c *gin.Context, line string) bool {
	payload, err := json.Marshal(gin.H{"line": line})
	if err != nil {
		return false
	}
	if _, err := fmt.Fprintf(c.Writer, "event: log\ndata: %s\n\n", payload); err != nil {
		return false
	}
	c.Writer.Flush()
	return true
}
