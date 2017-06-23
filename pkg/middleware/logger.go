package middleware

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/hellofresh/janus/pkg/response"
	log "github.com/sirupsen/logrus"
)

// Logger struct contains data and logic required for middleware functionality
type Logger struct{}

// NewLogger builds and returns new Logger middleware instance
func NewLogger() *Logger {
	return &Logger{}
}

// Handler implementation
func (m *Logger) Handler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		log.WithFields(log.Fields{"method": r.Method, "path": r.URL.Path}).Debug("Started request")

		// reverse proxy replaces original request with target request, so keep original one
		originalURL := &url.URL{}
		*originalURL = *r.URL

		var (
			lock         sync.Mutex
			responseCode int
		)
		hooks := response.Hooks{
			WriteHeader: func(next response.WriteHeaderFunc) response.WriteHeaderFunc {
				return func(code int) {
					next(code)
					lock.Lock()
					defer lock.Unlock()

					responseCode = code
				}
			},
		}

		fields := log.Fields{
			"method":      r.Method,
			"host":        r.Host,
			"request":     r.RequestURI,
			"remote-addr": r.RemoteAddr,
			"referer":     r.Referer(),
			"user-agent":  r.UserAgent(),
		}

		handler.ServeHTTP(response.Wrap(w, hooks), r)

		fields["code"] = responseCode
		fields["duration"] = int(time.Now().Sub(start) / time.Millisecond)
		if originalURL.String() != r.URL.String() {
			fields["upstream-host"] = r.URL.Host
			fields["upstream-request"] = r.URL.RequestURI()
		}

		log.WithFields(fields).Info("Completed handling request")
	})
}
