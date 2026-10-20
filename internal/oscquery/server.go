package oscquery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Server answers VRChat's OSCQuery HTTP requests. Only HOST_INFO and the
// root node (advertising /avatar/change) are implemented, which is all
// VRChat needs to start sending OSC to us.
type Server struct {
	logger *zap.Logger

	mu      sync.Mutex
	appName string
	oscIP   string
	oscPort int
	srv     *http.Server
	port    int

	// OnHostInfoQueried fires whenever VRChat asks for HOST_INFO, which
	// indicates it (re)connected and the avatar should be re-resolved.
	OnHostInfoQueried func()
}

func NewServer(logger *zap.Logger) *Server {
	return &Server{logger: logger.Named("OscQueryHttp"), appName: "VRCFT"}
}

func (s *Server) SetAppName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.appName = name
}

// Bind starts listening on a random loopback port and remembers the OSC
// endpoint to advertise. It returns the HTTP port.
func (s *Server) Bind(oscIP string, oscPort int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("listen oscquery http: %w", err)
	}
	s.oscIP, s.oscPort = oscIP, oscPort
	s.port = ln.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	s.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func(srv *http.Server) {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("OSCQuery HTTP server stopped", zap.Error(err))
		}
	}(s.srv)
	return s.port, nil
}

// Port returns the bound HTTP port, or 0.
func (s *Server) Port() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.port
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	var body []byte
	var err error

	if strings.Contains(r.URL.RawQuery, "HOST_INFO") {
		s.mu.Lock()
		info := newHostInfo(s.appName, s.oscIP, s.oscPort)
		s.mu.Unlock()
		body, err = json.Marshal(info)
		if fn := s.OnHostInfoQueried; fn != nil {
			fn()
		}
		s.logger.Debug("Responding to oscquery host info request", zap.ByteString("response", body))
	} else {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		root := NewRoot()
		root.Add(&Node{FullPath: "/avatar/change", Access: AccessWriteOnly, OSCType: "s"})
		body, err = root.JSON()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (s *Server) stopLocked() {
	if s.srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = s.srv.Shutdown(ctx)
	s.srv = nil
	s.port = 0
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
	return nil
}
