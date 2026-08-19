package handlers

import (
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/container"
)

const (
	clientErrorMaxBodyBytes = 16 << 10
	clientErrorMaxPerMinute = 20
	clientErrorMaxClients   = 2048
)

type clientErrorRateWindow struct {
	startedAt time.Time
	count     int
}

var clientErrorRateLimiter = struct {
	sync.Mutex
	clients map[string]clientErrorRateWindow
}{clients: make(map[string]clientErrorRateWindow)}

func allowClientError(clientIP string, now time.Time) bool {
	clientErrorRateLimiter.Lock()
	defer clientErrorRateLimiter.Unlock()
	for ip, entry := range clientErrorRateLimiter.clients {
		if now.Sub(entry.startedAt) >= time.Minute {
			delete(clientErrorRateLimiter.clients, ip)
		}
	}

	window := clientErrorRateLimiter.clients[clientIP]
	if window.startedAt.IsZero() && len(clientErrorRateLimiter.clients) >= clientErrorMaxClients {
		return false
	}
	if window.startedAt.IsZero() || now.Sub(window.startedAt) >= time.Minute {
		window = clientErrorRateWindow{startedAt: now}
	}
	if window.count >= clientErrorMaxPerMinute {
		clientErrorRateLimiter.clients[clientIP] = window
		return false
	}
	window.count++
	clientErrorRateLimiter.clients[clientIP] = window
	return true
}

func PingHandler(c *container.AppContainer) gin.HandlerFunc {
	return func(g *gin.Context) {
		res := map[string]any{
			"ping": "pong",
		}
		g.JSON(http.StatusOK, res)
	}
}

// ClientErrorHandler logs JavaScript errors sent from the frontend.
// This is a best-effort, fire-and-forget endpoint - no auth required
// since it's meant to capture errors even before login completes.
func ClientErrorHandler(c *container.AppContainer) gin.HandlerFunc {
	return func(g *gin.Context) {
		if !allowClientError(g.ClientIP(), time.Now()) {
			g.Status(http.StatusTooManyRequests)
			return
		}

		g.Request.Body = http.MaxBytesReader(g.Writer, g.Request.Body, clientErrorMaxBodyBytes)
		body, err := io.ReadAll(g.Request.Body)
		if err != nil {
			g.Status(http.StatusRequestEntityTooLarge)
			return
		}

		log.Printf("[CLIENT-ERROR] ip=%s bytes=%d payload=%q", g.ClientIP(), len(body), body)
		g.Status(http.StatusNoContent)
	}
}
