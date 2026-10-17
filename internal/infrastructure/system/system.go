// Package system implements Clock and IDGenerator on top of the OS.
package system

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/s4bb4t/infra-layer-practice/internal/app/port"
)

var (
	_ port.Clock       = Clock{}
	_ port.IDGenerator = UUIDv7{}
)

type Clock struct{}

func (Clock) Now() time.Time { return time.Now().UTC() }

// UUIDv7 generates time-ordered UUIDs (RFC 9562). Time ordering keeps B-tree
// primary keys append-mostly, unlike random UUIDv4.
type UUIDv7 struct{}

func (UUIDv7) NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	ms := uint64(time.Now().UnixMilli())
	b[0], b[1], b[2], b[3], b[4], b[5] = byte(ms>>40), byte(ms>>32), byte(ms>>24), byte(ms>>16), byte(ms>>8), byte(ms)
	b[6] = b[6]&0x0f | 0x70 // version 7
	b[8] = b[8]&0x3f | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
