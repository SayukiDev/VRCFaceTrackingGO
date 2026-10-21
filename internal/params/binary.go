package params

import (
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/dlclark/regexp2"

	"VRCFaceTrackingGo/pkg/tracking"
)

// Binary encodes a float in [0, 1] (or [-1, 1] with the Negative flag) into
// a set of boolean parameters named <name>1, <name>2, <name>4, ... where the
// suffix is the bit weight. The avatar decides how many bits exist.
type Binary struct {
	name     string
	get      func(*tracking.TrackingData) float32
	matcher  *regexp2.Regexp
	negative *Base[bool]

	bits           []*Base[bool]
	maxBinaryValue int
}

func NewBinary(name string, get func(*tracking.TrackingData) float32) *Binary {
	n := regexp2.Escape(name)
	return &Binary{
		name:     name,
		get:      get,
		matcher:  regexp2.MustCompile(`(?<!(v\d+))/`+n+`\d+$|^`+n+`\d+$`, regexp2.None),
		negative: NewBase(name+"Negative", func(d *tracking.TrackingData) bool { return get(d) < 0 }, true),
	}
}

func (b *Binary) Names() []string {
	names := make([]string, 0, len(b.bits)+1)
	for _, bit := range b.bits {
		names = append(names, bit.name)
	}
	return append(names, b.negative.name)
}

func (b *Binary) processBit(data *tracking.TrackingData, shift int) bool {
	value := b.get(data)
	// Without a Negative parameter on the avatar, negative values are cut.
	if !b.negative.Relevant() && value < 0 {
		return false
	}
	value = float32(math.Abs(float64(value)))
	if value > 0.99999 {
		return true
	}
	big := int(value * float32(b.maxBinaryValue))
	return (big>>shift)&1 == 1
}

// binarySteps returns the bit position of index when index is a power of
// two, otherwise ok is false.
func binarySteps(index int) (shift int, ok bool) {
	current := 1
	for i := range index {
		if current == index {
			return i, true
		}
		current *= 2
	}
	return 0, false
}

func trailingNumber(s string) (int, bool) {
	end := len(s)
	start := end
	for start > 0 && unicode.IsDigit(rune(s[start-1])) {
		start--
	}
	if start == end {
		return 0, false
	}
	n, err := strconv.Atoi(s[start:end])
	return n, err == nil
}

func (b *Binary) Reset(defs []Definition, forceRelevant bool) []Output {
	b.bits = b.bits[:0]
	outputs := b.negative.Reset(defs, forceRelevant)

	type bit struct {
		index, shift int
	}
	var found []bit
	for _, def := range defs {
		if def.Type != TypeBool {
			continue
		}
		if ok, _ := b.matcher.MatchString(def.Address); !ok {
			continue
		}
		index, ok := trailingNumber(strings.ReplaceAll(def.Name, b.name, ""))
		if !ok {
			continue
		}
		shift, ok := binarySteps(index)
		if !ok {
			continue
		}
		found = append(found, bit{index: index, shift: shift})
	}
	if len(found) == 0 {
		return outputs
	}

	b.maxBinaryValue = 1 << len(found)
	for _, f := range found {
		shift := f.shift
		p := NewBase(b.name+strconv.Itoa(f.index), func(d *tracking.TrackingData) bool { return b.processBit(d, shift) }, true)
		outputs = append(outputs, p.Reset(defs, forceRelevant)...)
		b.bits = append(b.bits, p)
	}
	return outputs
}
