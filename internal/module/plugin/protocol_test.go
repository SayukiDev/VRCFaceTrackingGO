package plugin

import (
	"bufio"
	"bytes"
	"errors"
	"testing"

	"VRCFaceTrackingGo/internal/module/adapter"
	"VRCFaceTrackingGo/pkg/tracking"
)

func TestDataRoundTrip(t *testing.T) {
	var in tracking.TrackingData
	// A distinct value in every transmitted field.
	for i, v := range fields(&in) {
		*v = float32(i) + 0.5
	}
	buf := appendData(nil, &in)
	if len(buf) != dataSize {
		t.Fatalf("encoded %d bytes, want %d", len(buf), dataSize)
	}
	out, err := readData(bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("round trip changed the data:\n in  %+v\n out %+v", in, out)
	}
	if out.Eye.MinDilation == 0 || out.Shapes[tracking.ExpressionCount-1].Weight == 0 || out.Head.PosZ == 0 {
		t.Fatal("a field was not transmitted")
	}
}

func TestSupportRoundTrip(t *testing.T) {
	for _, s := range []adapter.Support{{}, {Eye: true}, {Expression: true}, {Eye: true, Expression: true}} {
		if got := decodeSupport(encodeSupport(s)); got != s {
			t.Errorf("%+v became %+v", s, got)
		}
	}
}

func TestResultStatus(t *testing.T) {
	ok := appendResult(nil, nil, func(b []byte) []byte { return append(b, 7) })
	r := bufio.NewReader(bytes.NewReader(ok))
	if err := readStatus(r); err != nil {
		t.Fatal(err)
	}
	if b, _ := r.ReadByte(); b != 7 {
		t.Fatalf("payload = %d", b)
	}

	failed := appendResult(nil, errors.New("no device"), nil)
	err := readStatus(bufio.NewReader(bytes.NewReader(failed)))
	var remote remoteError
	if !errors.As(err, &remote) || err.Error() != "no device" {
		t.Fatalf("err = %v", err)
	}
}
