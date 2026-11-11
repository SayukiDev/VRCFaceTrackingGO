package oscquery

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

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

const browseInterval = 10 * time.Second

// Start begins browsing for VRChat.
func (d *Discovery) Start(ctx context.Context) error {
	// Validate the multicast setup once so a hard failure is reported.
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return fmt.Errorf("create mdns resolver: %w", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	d.mu.Lock()
	d.cancel = cancel
	d.mu.Unlock()

	go d.browseLoop(ctx, resolver)
	return nil
}

func (d *Discovery) browseLoop(ctx context.Context, resolver *zeroconf.Resolver) {
	for ctx.Err() == nil {
		if resolver == nil {
			var err error
			resolver, err = zeroconf.NewResolver(nil)
			if err != nil {
				d.logger.Warn("Failed to create mDNS resolver", zap.Error(err))
			}
		}
		if resolver != nil {
			d.browseOnce(ctx, resolver)
			resolver = nil // a resolver's sockets are closed when its browse ends
		}
		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
		}
	}
}

func (d *Discovery) browseOnce(ctx context.Context, resolver *zeroconf.Resolver) {
	roundCtx, cancel := context.WithTimeout(ctx, browseInterval)
	defer cancel()
	entries := make(chan *zeroconf.ServiceEntry, 16)
	if err := resolver.Browse(roundCtx, oscJSONService, mdnsDomain, entries); err != nil {
		d.logger.Debug("mDNS browse failed", zap.Error(err))
		return
	}
	// consume returns once zeroconf closes entries (round over).
	d.consume(entries)
}

func isGameClient(instance string) bool {
	return strings.HasPrefix(instance, "VRChat-Client") || strings.HasPrefix(instance, "ChilloutVR-GameClient")
}

func (d *Discovery) consume(entries <-chan *zeroconf.ServiceEntry) {
	for e := range entries {
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

func (d *Discovery) setEndpoint(ep *net.TCPAddr) {
	d.mu.Lock()
	if d.cancel == nil { // closed; a late entry from a finishing round
		d.mu.Unlock()
		return
	}
	first := d.endpoint == nil
	if !first && (!d.endpoint.IP.Equal(ep.IP) || d.endpoint.Port != ep.Port) {
		d.logger.Info("VRChat OSCQuery endpoint changed", zap.String("endpoint", ep.String()))
	}
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
