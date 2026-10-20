package oscquery

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/grandcat/zeroconf"
	"go.uber.org/zap"
)

const (
	oscJSONService = "_oscjson._tcp"
	oscUDPService  = "_osc._udp"
	mdnsDomain     = "local."
)

// Discovery finds the VRChat client's OSCQuery endpoint via mDNS and
// advertises our own services.
type Discovery struct {
	logger *zap.Logger

	mu       sync.Mutex
	endpoint *net.TCPAddr
	servers  []*zeroconf.Server
	cancel   context.CancelFunc

	// OnClientDiscovered fires the first time a VRChat client is found.
	OnClientDiscovered func(endpoint *net.TCPAddr)
}

func NewDiscovery(logger *zap.Logger) *Discovery {
	return &Discovery{logger: logger.Named("mDNS")}
}

// Start begins browsing for VRChat.
func (d *Discovery) Start(ctx context.Context) error {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return fmt.Errorf("create mdns resolver: %w", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	d.mu.Lock()
	d.cancel = cancel
	d.mu.Unlock()

	entries := make(chan *zeroconf.ServiceEntry, 16)
	go d.consume(ctx, entries)
	if err := resolver.Browse(ctx, oscJSONService, mdnsDomain, entries); err != nil {
		cancel()
		return fmt.Errorf("browse %s: %w", oscJSONService, err)
	}
	return nil
}

func isGameClient(instance string) bool {
	return strings.HasPrefix(instance, "VRChat-Client") || strings.HasPrefix(instance, "ChilloutVR-GameClient")
}

func (d *Discovery) consume(ctx context.Context, entries <-chan *zeroconf.ServiceEntry) {
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-entries:
			if !ok {
				return
			}
			if !isGameClient(e.Instance) || len(e.AddrIPv4) == 0 || e.Port == 0 {
				continue
			}
			// VRChat always advertises 127.0.0.1; prefer a non-loopback
			// address if one is present, otherwise use loopback.
			ip := net.IPv4(127, 0, 0, 1)
			for _, a := range e.AddrIPv4 {
				if !a.IsLoopback() {
					ip = a
					break
				}
			}
			d.setEndpoint(&net.TCPAddr{IP: ip, Port: e.Port})
		}
	}
}

func (d *Discovery) setEndpoint(ep *net.TCPAddr) {
	d.mu.Lock()
	first := d.endpoint == nil
	d.endpoint = ep
	fn := d.OnClientDiscovered
	d.mu.Unlock()
	if first && fn != nil {
		fn(ep)
	}
}

// Endpoint returns the VRChat OSCQuery endpoint, or nil if not found yet.
func (d *Discovery) Endpoint() *net.TCPAddr {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.endpoint
}

// Advertise registers instance for both the OSCQuery (HTTP) and OSC (UDP)
// services so VRChat shows us in its OSC debug panel and sends data.
func (d *Discovery) Advertise(instance string, httpPort, oscPort int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.shutdownServersLocked()

	for _, svc := range []struct {
		name string
		port int
	}{{oscJSONService, httpPort}, {oscUDPService, oscPort}} {
		d.logger.Debug("Advertising service", zap.String("service", svc.name), zap.String("instance", instance), zap.Int("port", svc.port))
		srv, err := zeroconf.RegisterProxy(instance, svc.name, mdnsDomain, svc.port, instance, []string{"127.0.0.1"}, []string{"txtvers=1"}, nil)
		if err != nil {
			return fmt.Errorf("advertise %s: %w", svc.name, err)
		}
		d.servers = append(d.servers, srv)
	}
	return nil
}

func (d *Discovery) shutdownServersLocked() {
	for _, s := range d.servers {
		s.Shutdown()
	}
	d.servers = nil
}

func (d *Discovery) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}
	d.shutdownServersLocked()
	d.endpoint = nil
	return nil
}
