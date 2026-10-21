package plugin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"go.uber.org/zap"

	"VRCFaceTrackingGo/internal/module/adapter"
	"VRCFaceTrackingGo/pkg/tracking"
)

// exitTimeout is how long Close waits for the process to exit on its own
// before killing it.
const exitTimeout = 3 * time.Second

// Client is a running module program.
type Client struct {
	cmd    *exec.Cmd
	stdin  *os.File
	stdout *os.File
	r      *bufio.Reader
	exited chan struct{}

	// turn is held for a whole request/response exchange. A request whose
	// caller gave up keeps it until its response is read, so the stream
	// never gets out of step.
	turn chan struct{}

	closeOnce sync.Once
}

var _ Conn = (*Client)(nil)

// Start launches the module program at path. Its log is re-logged through
// logger.
func Start(path string, logger *zap.Logger) (*Client, error) {
	// Our own pipes rather than cmd.*Pipe, so cmd.Wait does not close them
	// under the reader.
	inR, inW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		closeAll(inR, inW)
		return nil, err
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		closeAll(inR, inW, outR, outW)
		return nil, err
	}

	cmd := exec.Command(path, "--"+HostFlag+"=true")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = inR, outW, errW
	prepare(cmd)
	err = cmd.Start()
	// The child owns its ends now (or failed to start).
	closeAll(inR, outW, errW)
	if err != nil {
		closeAll(inW, outR, errR)
		return nil, fmt.Errorf("start module %s: %w", path, err)
	}
	// Closing stdin already ends a module when the host goes away; this
	// also covers a module that is stuck.
	if err := adopt(cmd.Process); err != nil {
		logger.Warn("Module may outlive the host if it hangs", zap.String("path", path), zap.Error(err))
	}

	c := &Client{
		cmd:    cmd,
		stdin:  inW,
		stdout: outR,
		r:      bufio.NewReaderSize(outR, 1+dataSize),
		exited: make(chan struct{}),
		turn:   make(chan struct{}, 1),
	}
	go func() {
		defer errR.Close()
		forwardLogs(errR, logger)
	}()
	go func() {
		_ = cmd.Wait()
		close(c.exited)
	}()
	return c, nil
}

func closeAll(files ...*os.File) {
	for _, f := range files {
		_ = f.Close()
	}
}

type result[T any] struct {
	v   T
	err error
}

// exchange sends req and decodes the response with read, giving up when
// ctx is done. An I/O failure means the process is gone (ErrExited).
func exchange[T any](ctx context.Context, c *Client, req []byte, read func(*bufio.Reader) (T, error)) (T, error) {
	var zero T
	select {
	case c.turn <- struct{}{}:
	case <-ctx.Done():
		return zero, ctx.Err()
	}
	done := make(chan result[T], 1)
	go func() {
		defer func() { <-c.turn }()
		var res result[T]
		if _, err := c.stdin.Write(req); err != nil {
			res.err = ErrExited
		} else if err := readStatus(c.r); err != nil {
			res.err = err
		} else {
			res.v, res.err = read(c.r)
		}
		var remote remoteError
		if res.err != nil && !errors.As(res.err, &remote) {
			res.err = ErrExited
		}
		done <- res
	}()
	select {
	case res := <-done:
		return res.v, res.err
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

func (c *Client) Initialize(ctx context.Context, avail adapter.Support) (adapter.Support, error) {
	return exchange(ctx, c, []byte{opInitialize, encodeSupport(avail)}, func(r *bufio.Reader) (adapter.Support, error) {
		b, err := r.ReadByte()
		return decodeSupport(b), err
	})
}

func (c *Client) Update(ctx context.Context) (tracking.TrackingData, error) {
	return exchange(ctx, c, []byte{opUpdate}, func(r *bufio.Reader) (tracking.TrackingData, error) {
		return readData(r)
	})
}

func (c *Client) Teardown(ctx context.Context) error {
	_, err := exchange(ctx, c, []byte{opTeardown}, func(*bufio.Reader) (struct{}, error) {
		return struct{}{}, nil
	})
	return err
}

// Close closes the module's stdin, which makes Serve return, and kills the
// process if it does not exit in time. It is safe to call more than once.
func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		_ = c.stdin.Close()
		select {
		case <-c.exited:
		case <-time.After(exitTimeout):
			err = c.cmd.Process.Kill()
			<-c.exited
		}
		// The process is gone, so a pending read has ended with EOF.
		_ = c.stdout.Close()
	})
	return err
}
