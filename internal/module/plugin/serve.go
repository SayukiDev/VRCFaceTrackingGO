package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"VRCFaceTracking/internal/module/adapter"
	"VRCFaceTracking/pkg/tracking"
)

// Serve runs mod as a module program until the host closes stdin, or
// prints its Description when started with --info. Serving the host
// requires --vfgmod=true. It must be called from main and nothing else may
// write to stdout.
func Serve(mod adapter.Module) {
	info := flag.Bool(InfoFlag, false, "print the module description as JSON and exit")
	host := flag.Bool(HostFlag, false, "serve the VRCFaceTracking host over stdio (set by the host)")
	flag.Parse()
	if *info {
		if err := json.NewEncoder(os.Stdout).Encode(describe(mod)); err != nil {
			os.Exit(1)
		}
		return
	}
	if !*host {
		fmt.Fprintf(os.Stderr, "%s is a VRCFaceTracking module. Install it from the Modules page instead of running it.\n", mod.Metadata().Name)
		os.Exit(2)
	}

	stdin, stdout := os.Stdin, os.Stdout
	// Stray prints must not corrupt the protocol stream.
	os.Stdout = os.Stderr

	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.Lock(os.Stderr),
		zapcore.DebugLevel,
	))
	defer logger.Sync()
	watchParent(logger)

	ctx, cancel := context.WithCancel(context.Background())
	s := &server{ctx: ctx, mod: mod, logger: logger, data: tracking.NewTrackingData()}
	if err := s.serve(bufio.NewReader(stdin), stdout); err != nil {
		logger.Error("Protocol error", zap.Error(err))
	}

	// The host is gone (or asked us to stop); release the device.
	cancel()
	if err := s.teardownSafe(); err != nil {
		logger.Warn("Module teardown failed", zap.Error(err))
	}
}

func describe(mod adapter.Module) Description {
	return Description{ID: mod.ID(), Metadata: mod.Metadata(), Supported: mod.Supported()}
}

// server answers the host's requests for one adapter.Module.
type server struct {
	ctx    context.Context
	mod    adapter.Module
	logger *zap.Logger
	data   *tracking.TrackingData

	initialized bool
}

// serve answers requests until r ends.
func (s *server) serve(r *bufio.Reader, w io.Writer) error {
	buf := make([]byte, 0, 1+dataSize)
	for {
		op, err := r.ReadByte()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		buf = buf[:0]
		switch op {
		case opInitialize:
			b, err := r.ReadByte()
			if err != nil {
				return err
			}
			granted, err := s.initialize(decodeSupport(b))
			buf = appendResult(buf, err, func(buf []byte) []byte { return append(buf, encodeSupport(granted)) })
		case opUpdate:
			err := s.update()
			buf = appendResult(buf, err, func(buf []byte) []byte { return appendData(buf, s.data) })
		case opTeardown:
			buf = appendResult(buf, s.teardownSafe(), nil)
		default:
			return fmt.Errorf("unknown request %q", op)
		}
		if _, err := w.Write(buf); err != nil {
			return err
		}
	}
}

// recoverPanic turns a panic in the module into an error for the host, so
// one bad call does not take the whole process down.
func (s *server) recoverPanic(method string, err *error) {
	if r := recover(); r != nil {
		s.logger.Error("Module panicked", zap.String("method", method), zap.Any("panic", r), zap.Stack("stack"))
		*err = fmt.Errorf("module panicked in %s: %v", method, r)
	}
}

func (s *server) initialize(avail adapter.Support) (granted adapter.Support, err error) {
	defer s.recoverPanic("Initialize", &err)
	granted, err = s.mod.Initialize(s.ctx, s.logger, avail)
	if err == nil {
		s.initialized = true
	}
	return granted, err
}

func (s *server) update() (err error) {
	defer s.recoverPanic("Update", &err)
	return s.mod.Update(s.ctx, s.data)
}

func (s *server) teardownSafe() (err error) {
	defer s.recoverPanic("Teardown", &err)
	return s.teardown()
}

func (s *server) teardown() error {
	if !s.initialized {
		return nil
	}
	s.initialized = false
	return s.mod.Teardown()
}
