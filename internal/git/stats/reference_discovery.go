package stats

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"gitlab.com/gitlab-org/gitaly/internal/git/pktline"
	"gitlab.com/gitlab-org/gitaly/internal/helper/text"
)

// Reference as used by the reference discovery protocol
type Reference struct {
	// Oid is the object ID the reference points to
	Oid string
	// Name of the reference. The name will be suffixed with ^{} in case
	// the reference is the peeled commit.
	Name string
}

// ReferenceDiscovery contains information about a reference discovery session.
type ReferenceDiscovery struct {
	// firstPacket tracks the time when the first pktline was received
	firstPacket time.Time
	// lastPacket tracks the time when the last pktline was received
	lastPacket time.Time
	// payloadSize tracks the size of all pktlines' data
	payloadSize int64
	// packets tracks the total number of packets consumed
	packets int
	// refs contains all announced references
	refs []Reference
	// caps contains all supported capabilities
	caps []string
}

func (d *ReferenceDiscovery) FirstPacket() time.Time { return d.firstPacket }
func (d *ReferenceDiscovery) LastPacket() time.Time  { return d.lastPacket }
func (d *ReferenceDiscovery) PayloadSize() int64     { return d.payloadSize }
func (d *ReferenceDiscovery) Packets() int           { return d.packets }
func (d *ReferenceDiscovery) Refs() []Reference      { return d.refs }
func (d *ReferenceDiscovery) Caps() []string         { return d.caps }

type referenceDiscoveryState int

const (
	referenceDiscoveryExpectService referenceDiscoveryState = iota
	referenceDiscoveryExpectFlush
	referenceDiscoveryExpectRefWithCaps
	referenceDiscoveryExpectRef
	referenceDiscoveryExpectEnd
)

// ParseReferenceDiscovery parses a client's reference discovery stream and
// returns either information about the reference discovery or an error in case
// it couldn't make sense of the client's request.
func ParseReferenceDiscovery(body io.Reader) (ReferenceDiscovery, error) {
	d := ReferenceDiscovery{}
	return d, d.Parse(body)
}

// Parse parses a client's reference discovery stream into the given
// ReferenceDiscovery struct or returns an error in case it couldn't make sense
// of the client's request.
//
// Expected protocol:
// - "# service=git-upload-pack\n"
// - FLUSH
// - "<OID> <ref>\x00<capabilities>\n"
// - "<OID> <ref>\n"
// - ...
// - FLUSH
func (d *ReferenceDiscovery) Parse(body io.Reader) error {
	state := referenceDiscoveryExpectService
	scanner := pktline.NewScanner(body)

	for ; scanner.Scan(); d.packets++ {
		pkt := scanner.Bytes()
		data := text.ChompBytes(pktline.Data(pkt))
		d.payloadSize += int64(len(data))

		switch state {
		case referenceDiscoveryExpectService:
			d.firstPacket = time.Now()
			if data != "# service=git-upload-pack" {
				return fmt.Errorf("unexpected header %q", data)
			}

			state = referenceDiscoveryExpectFlush
		case referenceDiscoveryExpectFlush:
			if !pktline.IsFlush(pkt) {
				return errors.New("missing flush after service announcement")
			}

			state = referenceDiscoveryExpectRefWithCaps
		case referenceDiscoveryExpectRefWithCaps:
			split := strings.SplitN(data, "\000", 2)
			if len(split) != 2 {
				return errors.New("invalid first reference line")
			}

			ref := strings.SplitN(string(split[0]), " ", 2)
			if len(ref) != 2 {
				return errors.New("invalid reference line")
			}
			d.refs = append(d.refs, Reference{Oid: ref[0], Name: ref[1]})
			d.caps = strings.Split(string(split[1]), " ")

			state = referenceDiscoveryExpectRef
		case referenceDiscoveryExpectRef:
			if pktline.IsFlush(pkt) {
				state = referenceDiscoveryExpectEnd
				continue
			}

			split := strings.SplitN(data, " ", 2)
			if len(split) != 2 {
				return errors.New("invalid reference line")
			}
			d.refs = append(d.refs, Reference{Oid: split[0], Name: split[1]})
		case referenceDiscoveryExpectEnd:
			return errors.New("received packet after flush")
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	if len(d.refs) == 0 {
		return errors.New("received no references")
	}
	if len(d.caps) == 0 {
		return errors.New("received no capabilities")
	}

	d.lastPacket = time.Now()

	return nil
}
