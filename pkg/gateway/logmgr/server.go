package logmgr

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis/response"
)

const (
	pollInterval = 500 * time.Millisecond
	heartbeat    = 15 * time.Second
)

var (
	textLevelPattern = regexp.MustCompile(`(?i)\blevel=(error|warn|info)\b`)
	jsonLevelPattern = regexp.MustCompile(`(?i)"level"\s*:\s*"(error|warn|info)"`)
)

// InstallHandler registers the real-time log stream and export endpoints.
func InstallHandler(group *gin.RouterGroup, logPath string) {
	group.GET("/system/logs/stream", streamLogs(logPath))
	group.GET("/system/logs/export", exportLogs(logPath))
}

func exportLogs(logPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawLevel := strings.TrimSpace(c.Query("level"))
		level, ok := normalizeLogLevel(rawLevel)
		if !ok {
			response.WriteError(c, http.StatusBadRequest, response.ErrSystemLogLevelInvalid(rawLevel))
			return
		}
		if strings.TrimSpace(logPath) == "" {
			response.WriteError(c, http.StatusBadRequest, response.ErrSystemLogFileUnavailable())
			return
		}
		file, err := os.Open(logPath)
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrSystemLogFileUnavailable())
			return
		}
		defer file.Close()

		filename := fmt.Sprintf("harns-gateway-%s.log", level)
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		c.Header("Cache-Control", "no-store")
		c.Status(http.StatusOK)

		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if logLineLevel(line) != level {
				continue
			}
			if _, err := c.Writer.WriteString(line + "\n"); err != nil {
				return
			}
		}
		c.Writer.Flush()
	}
}

func normalizeLogLevel(value string) (string, bool) {
	switch strings.ToLower(value) {
	case "error", "warn", "info":
		return strings.ToLower(value), true
	default:
		return "", false
	}
}

func logLineLevel(line string) string {
	if match := textLevelPattern.FindStringSubmatch(line); match != nil {
		return strings.ToLower(match[1])
	}
	if match := jsonLevelPattern.FindStringSubmatch(line); match != nil {
		return strings.ToLower(match[1])
	}
	return ""
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
