// Copied from: github.com/llehouerou/waves

package player

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/gopxl/beep/v2"
	"github.com/jfreymuth/vorbis"
	"github.com/jj11hh/opus"
)

// decodeOgg decodes an Ogg stream (Opus or Vorbis) into a beep streamer.
func decodeOgg(rc io.ReadSeekCloser) (beep.StreamSeekCloser, beep.Format, error) {
	// Read first page to get the identification packet
	hdr, err := parseOggPageHeader(rc)
	if err != nil {
		return nil, beep.Format{}, err
	}
	packets, partial, err := readOggPageBody(rc, hdr)
	if err != nil {
		return nil, beep.Format{}, err
	}
	if len(packets) == 0 {
		return nil, beep.Format{}, errors.New("ogg: no packets in first page")
	}

	// Detect codec from first packet
	codec, err := detectOggCodec(packets[0])
	if err != nil {
		return nil, beep.Format{}, err
	}

	// Feed header packets until codec is ready
	// Track partial packets that span pages
	for {
		complete, err := codec.AddHeaderPacket(nil) // Check if already complete
		if err != nil {
			return nil, beep.Format{}, err
		}
		if complete {
			break
		}

		// Read more pages for headers
		hdr, err := parseOggPageHeader(rc)
		if err != nil {
			return nil, beep.Format{}, err
		}
		pagePackets, newPartial, err := readOggPageBody(rc, hdr)
		if err != nil {
			return nil, beep.Format{}, err
		}

		// Join partial from previous page with first packet/partial of this page
		if len(partial) > 0 {
			if len(pagePackets) > 0 {
				// Previous partial + first complete packet = one header
				pagePackets[0] = append(partial, pagePackets[0]...)
			} else if newPartial != nil {
				// Previous partial + new partial (still spanning)
				newPartial = append(partial, newPartial...)
			}
		}

		// Feed complete packets to codec
		for _, pkt := range pagePackets {
			complete, err = codec.AddHeaderPacket(pkt)
			if err != nil {
				return nil, beep.Format{}, err
			}
			if complete {
				break
			}
		}

		// Track new partial for next iteration
		partial = newPartial
	}

	// Record where audio data starts (after headers)
	dataStart, err := rc.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, beep.Format{}, err
	}

	// Create OggReader
	ogg, err := NewOggReader(rc, codec.SampleRate(), codec.PreSkip())
	if err != nil {
		return nil, beep.Format{}, err
	}
	ogg.SetDataStart(dataStart)
	if err := ogg.ScanLastGranule(); err != nil {
		return nil, beep.Format{}, err
	}
	// Seek back to audio start
	if _, err := rc.Seek(dataStart, io.SeekStart); err != nil {
		return nil, beep.Format{}, err
	}

	format := beep.Format{
		SampleRate:  beep.SampleRate(codec.SampleRate()),
		NumChannels: codec.Channels(),
		Precision:   2,
	}

	decoder := &oggDecoder{
		ogg:       ogg,
		codec:     codec,
		closer:    rc,
		pcmBuffer: make([]float32, 8192*codec.Channels()),
		totalLen:  ogg.Duration(),
	}
	decoder.pcmPos = len(decoder.pcmBuffer) // empty buffer triggers refill

	return decoder, format, nil
}

// oggDecoder implements beep.StreamSeekCloser for Ogg streams.
type oggDecoder struct {
	ogg    *OggReader
	codec  OggCodec
	closer io.Closer

	currentPage *OggPage
	packetIdx   int
	pcmBuffer   []float32
	pcmPos      int
	granulePos  int64
	totalLen    int64
	err         error
}

// Stream reads audio samples into the provided buffer.
func (d *oggDecoder) Stream(samples [][2]float64) (n int, ok bool) {
	if d.err != nil {
		return 0, false
	}

	channels := d.codec.Channels()

	for n < len(samples) {
		// Use buffered PCM
		if d.pcmPos < len(d.pcmBuffer) {
			for n < len(samples) && d.pcmPos < len(d.pcmBuffer) {
				if channels == 2 {
					samples[n][0] = float64(d.pcmBuffer[d.pcmPos])
					samples[n][1] = float64(d.pcmBuffer[d.pcmPos+1])
					d.pcmPos += 2
				} else {
					samples[n][0] = float64(d.pcmBuffer[d.pcmPos])
					samples[n][1] = float64(d.pcmBuffer[d.pcmPos])
					d.pcmPos++
				}
				n++
				d.granulePos++
			}
			continue
		}

		// Need more packets
		if d.currentPage == nil || d.packetIdx >= len(d.currentPage.Packets) {
			page, err := d.ogg.ReadPage()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return n, n > 0
				}
				d.err = err
				return n, n > 0
			}
			d.currentPage = page
			d.packetIdx = 0
		}

		// Decode next packet
		if d.packetIdx < len(d.currentPage.Packets) {
			packet := d.currentPage.Packets[d.packetIdx]
			d.packetIdx++

			samplesPerChannel, err := d.codec.Decode(packet, d.pcmBuffer[:cap(d.pcmBuffer)])
			if err != nil {
				continue // skip invalid packets
			}
			d.pcmBuffer = d.pcmBuffer[:samplesPerChannel*channels]
			d.pcmPos = 0
		}
	}

	return n, true
}

// Err returns any error that occurred during streaming.
func (d *oggDecoder) Err() error { return d.err }

// Len returns the total number of samples.
func (d *oggDecoder) Len() int { return int(d.totalLen) }

// Position returns the current sample position.
func (d *oggDecoder) Position() int { return int(d.granulePos) }

// Seek seeks to the given sample position.
func (d *oggDecoder) Seek(p int) error {
	if p < 0 {
		p = 0
	}
	if p > d.Len() {
		p = d.Len()
	}

	if err := d.ogg.SeekToGranule(int64(p)); err != nil {
		return err
	}

	d.currentPage = nil
	d.packetIdx = 0
	d.pcmBuffer = d.pcmBuffer[:cap(d.pcmBuffer)]
	d.pcmPos = len(d.pcmBuffer)
	d.granulePos = int64(p)
	d.err = nil

	return d.codec.Reset()
}

// Close closes the decoder and underlying file.
func (d *oggDecoder) Close() error {
	return d.closer.Close()
}

const oggMagic = "OggS"

var (
	errInvalidOggMagic   = errors.New("ogg: invalid capture pattern")
	errInvalidOggVersion = errors.New("ogg: unsupported version")
)

// oggPageHeader represents the header of an Ogg page.
type oggPageHeader struct {
	GranulePos   int64
	SerialNumber uint32
	SequenceNum  uint32
	NumSegments  uint8
	SegmentTable []uint8
}

// parseOggPageHeader reads and parses an Ogg page header from the reader.
func parseOggPageHeader(r io.Reader) (*oggPageHeader, error) {
	// Read fixed header (27 bytes)
	var buf [27]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return nil, err
	}

	// Check capture pattern "OggS"
	if string(buf[0:4]) != oggMagic {
		return nil, errInvalidOggMagic
	}

	// Check version (must be 0)
	if buf[4] != 0 {
		return nil, errInvalidOggVersion
	}

	hdr := &oggPageHeader{
		GranulePos:   int64(binary.LittleEndian.Uint64(buf[6:14])), //nolint:gosec // granule position is semantically signed (-1 valid) but stored as unsigned
		SerialNumber: binary.LittleEndian.Uint32(buf[14:18]),
		SequenceNum:  binary.LittleEndian.Uint32(buf[18:22]),
		// checksum at buf[22:26] - skip validation for now
		NumSegments: buf[26],
	}

	// Read segment table
	if hdr.NumSegments > 0 {
		hdr.SegmentTable = make([]uint8, hdr.NumSegments)
		if _, err := io.ReadFull(r, hdr.SegmentTable); err != nil {
			return nil, err
		}
	}

	return hdr, nil
}

// readOggPageBody reads the page body and extracts packets.
// Packets are delimited by segment sizes: a segment of 255 bytes continues
// to the next segment, while a segment < 255 terminates the packet.
// Returns complete packets and any partial packet that continues to the next page.
func readOggPageBody(r io.Reader, hdr *oggPageHeader) (packets [][]byte, partial []byte, err error) {
	// Calculate total body size
	var totalSize int
	for _, seg := range hdr.SegmentTable {
		totalSize += int(seg)
	}

	// Read entire body
	body := make([]byte, totalSize)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, nil, err
	}

	// Extract packets from segments
	var currentPacket []byte
	offset := 0

	for _, segSize := range hdr.SegmentTable {
		currentPacket = append(currentPacket, body[offset:offset+int(segSize)]...)
		offset += int(segSize)

		// Segment < 255 terminates the packet
		if segSize < 255 {
			packets = append(packets, currentPacket)
			currentPacket = nil
		}
	}

	// If last segment was 255, packet continues to next page (incomplete)
	// Return it as partial for the caller to join with the next page
	if len(currentPacket) > 0 {
		partial = currentPacket
	}

	return packets, partial, nil
}

// OggReader reads Ogg streams with seeking support.
// It is codec-agnostic - the caller is responsible for parsing codec headers.
type OggReader struct {
	r           io.ReadSeeker
	fileSize    int64
	dataStart   int64 // byte offset where audio pages begin
	lastGranule int64 // cached from last page
	sampleRate  int   // for duration calculation
	preSkip     int   // samples to skip at start (Opus only, 0 for Vorbis)

	partial []byte // partial packet from previous page (continues on next page)
}

// NewOggReader creates a new OggReader from a seekable stream.
// sampleRate and preSkip are needed for duration/seeking calculations.
// The caller is responsible for parsing codec headers before calling this.
func NewOggReader(r io.ReadSeeker, sampleRate, preSkip int) (*OggReader, error) {
	size, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	return &OggReader{
		r:          r,
		fileSize:   size,
		sampleRate: sampleRate,
		preSkip:    preSkip,
	}, nil
}

// SetDataStart sets the byte offset where audio data begins.
// Called after header parsing is complete.
func (o *OggReader) SetDataStart(offset int64) {
	o.dataStart = offset
}

// SampleRate returns the sample rate used for duration calculations.
func (o *OggReader) SampleRate() int {
	return o.sampleRate
}

// PreSkip returns the number of samples to skip at the start.
func (o *OggReader) PreSkip() int {
	return o.preSkip
}

// OggPage represents a decoded Ogg page with its audio packets.
type OggPage struct {
	GranulePos int64
	Packets    [][]byte
	ByteOffset int64
}

// ReadPage reads the next Ogg page from the stream.
// Handles packets that span multiple pages by joining partial data.
// Returns io.EOF when no more pages are available.
func (o *OggReader) ReadPage() (*OggPage, error) {
	offset, err := o.r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	hdr, err := parseOggPageHeader(o.r)
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, io.EOF
		}
		return nil, err
	}

	packets, partial, err := readOggPageBody(o.r, hdr)
	if err != nil {
		return nil, err
	}

	// If we have a partial packet from the previous page, prepend it to the first packet
	if len(o.partial) > 0 {
		switch {
		case len(packets) > 0:
			// Join partial with first complete packet
			packets[0] = append(o.partial, packets[0]...)
		case partial != nil:
			// No complete packets, join partial with new partial
			partial = append(o.partial, partial...)
		default:
			// This shouldn't happen in valid Ogg streams
			packets = append(packets, o.partial)
		}
		o.partial = nil
	}

	// Store new partial for next page
	o.partial = partial

	return &OggPage{
		GranulePos: hdr.GranulePos,
		Packets:    packets,
		ByteOffset: offset,
	}, nil
}

// Reset seeks back to the start of audio data.
func (o *OggReader) Reset() error {
	o.partial = nil // Clear any partial packet
	_, err := o.r.Seek(o.dataStart, io.SeekStart)
	return err
}

// ScanLastGranule finds the granule position of the last page.
// This should be called after header parsing to enable duration calculation.
func (o *OggReader) ScanLastGranule() error {
	// Seek near end of file and scan for last page
	searchSize := min(int64(65536), o.fileSize) // Search last 64KB

	if _, err := o.r.Seek(o.fileSize-searchSize, io.SeekStart); err != nil {
		return err
	}

	// Read and scan for "OggS" magic
	buf := make([]byte, searchSize)
	n, err := io.ReadFull(o.r, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return err
	}
	buf = buf[:n]

	// Find last "OggS" occurrence
	lastOggS := -1
	for i := len(buf) - 4; i >= 0; i-- {
		if string(buf[i:i+4]) == oggMagic {
			lastOggS = i
			break
		}
	}

	if lastOggS == -1 {
		return errors.New("ogg: no page found at end of file")
	}

	// Parse granule position from that header
	if lastOggS+14 > len(buf) {
		return errors.New("ogg: incomplete last page header")
	}
	o.lastGranule = int64(binary.LittleEndian.Uint64(buf[lastOggS+6 : lastOggS+14])) //nolint:gosec // granule position is defined as unsigned but used as signed for duration calculations

	return nil
}

// Duration returns the total number of audio samples (excluding pre-skip).
func (o *OggReader) Duration() int64 {
	return o.lastGranule - int64(o.preSkip)
}

// SeekToGranule seeks to the page containing or just before the target granule position.
// Uses bisection search for efficiency on large files.
func (o *OggReader) SeekToGranule(target int64) error {
	// Handle seek to start
	if target <= 0 {
		return o.Reset()
	}

	// Bisection search
	low := o.dataStart
	high := o.fileSize

	bestOffset := o.dataStart
	var bestGranule int64

	for high-low > 4096 { // Stop when range is small enough
		mid := (low + high) / 2

		offset, granule, err := o.findPageNear(mid)
		if err != nil {
			// No page found in upper half, search lower
			high = mid
			continue
		}

		if granule <= target {
			// This page is valid, remember it and search higher
			bestOffset = offset
			bestGranule = granule
			low = offset + 1
		} else {
			// This page is past target, search lower
			high = mid
		}
	}

	// Linear scan from best known position to find exact page
	if _, err := o.r.Seek(bestOffset, io.SeekStart); err != nil {
		return err
	}

	// Scan forward to find the last page with granule ≤ target
	for {
		offset, err := o.r.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}

		hdr, err := parseOggPageHeader(o.r)
		if err != nil {
			break
		}

		// Skip page body
		var bodySize int
		for _, seg := range hdr.SegmentTable {
			bodySize += int(seg)
		}
		if _, err := o.r.Seek(int64(bodySize), io.SeekCurrent); err != nil {
			break
		}

		if hdr.GranulePos > target {
			// Went past target, seek back to previous page
			if _, err := o.r.Seek(bestOffset, io.SeekStart); err != nil {
				return err
			}
			break
		}

		if hdr.GranulePos >= 0 { // -1 means no granule
			bestOffset = offset
			bestGranule = hdr.GranulePos
		}
	}

	// Seek to best page found
	_, err := o.r.Seek(bestOffset, io.SeekStart)
	_ = bestGranule // Used for debugging if needed
	return err
}

// findPageNear finds an Ogg page starting at or after the given offset.
// Returns the page's byte offset and granule position.
func (o *OggReader) findPageNear(offset int64) (pageOffset, granule int64, err error) {
	if _, err := o.r.Seek(offset, io.SeekStart); err != nil {
		return 0, 0, err
	}

	// Read a chunk and scan for "OggS"
	buf := make([]byte, 4096)
	n, readErr := o.r.Read(buf)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return 0, 0, readErr
	}
	buf = buf[:n]

	if len(buf) < 27 {
		return 0, 0, errors.New("ogg: buffer too small to contain page header")
	}
	for i := range len(buf) - 27 {
		if string(buf[i:i+4]) == oggMagic && buf[i+4] == 0 { // version must be 0
			pageOffset = offset + int64(i)
			granule = int64(binary.LittleEndian.Uint64(buf[i+6 : i+14])) //nolint:gosec // granule position is semantically signed
			return pageOffset, granule, nil
		}
	}

	return 0, 0, errors.New("ogg: no page found")
}

const (
	opusHeadMagic = "OpusHead"
	oggFlacMagic  = "\x7fFLAC" // Ogg FLAC identification header
)

var (
	errUnknownOggCodec     = errors.New("ogg: unknown codec (not Opus or Vorbis)")
	errOggFlacNotSupported = errors.New("ogg: FLAC in Ogg container is not yet supported")
	errInvalidVorbisHeader = errors.New("vorbis: invalid identification header")
	errInvalidOpusHead     = errors.New("opus: invalid OpusHead packet")
	errUnsupportedOpus     = errors.New("opus: unsupported version")
)

// OggCodec handles codec-specific initialization and decoding for Ogg streams.
type OggCodec interface {
	// SampleRate returns the audio sample rate.
	SampleRate() int

	// Channels returns the number of audio channels.
	Channels() int

	// PreSkip returns samples to skip at stream start (0 for Vorbis).
	PreSkip() int

	// GranuleToSamples converts granule position to sample count.
	GranuleToSamples(granule int64) int64

	// AddHeaderPacket adds a header packet for codecs that need multiple headers.
	// Returns true when all headers are received.
	// For Opus, this is a no-op (single header). For Vorbis, collects 3 headers.
	AddHeaderPacket(packet []byte) (complete bool, err error)

	// Decode decodes a packet into PCM samples.
	// Returns the number of samples per channel decoded.
	Decode(packet []byte, pcm []float32) (samplesPerChannel int, err error)

	// Reset resets decoder state (needed after seeking).
	Reset() error
}

// detectOggCodec detects the codec from the first Ogg packet and returns
// an initialized codec ready to receive further header packets.
func detectOggCodec(firstPacket []byte) (OggCodec, error) {
	// Check for Opus: starts with "OpusHead"
	if len(firstPacket) >= 8 && string(firstPacket[:8]) == opusHeadMagic {
		return newOpusCodec(firstPacket)
	}

	// Check for Vorbis: starts with 0x01 + "vorbis"
	if len(firstPacket) >= 7 && firstPacket[0] == 0x01 && string(firstPacket[1:7]) == "vorbis" {
		return newVorbisCodec(firstPacket)
	}

	// Check for FLAC in Ogg: starts with 0x7F + "FLAC"
	if len(firstPacket) >= 5 && string(firstPacket[:5]) == oggFlacMagic {
		return nil, errOggFlacNotSupported
	}

	return nil, errUnknownOggCodec
}

// opusCodec implements OggCodec for Opus streams.
type opusCodec struct {
	decoder    *opus.Decoder
	channels   int
	preSkip    int
	sampleRate int // Original sample rate from header (informational only)
}

const opusSampleRate = 48000

// newOpusCodec creates an opusCodec from an OpusHead packet.
func newOpusCodec(packet []byte) (*opusCodec, error) {
	if len(packet) < 19 {
		return nil, errInvalidOpusHead
	}

	// Check version (must be 1)
	if packet[8] != 1 {
		return nil, errUnsupportedOpus
	}

	channels := int(packet[9])

	decoder, err := opus.NewDecoder(opusSampleRate, channels)
	if err != nil {
		return nil, err
	}

	return &opusCodec{
		decoder:    decoder,
		channels:   channels,
		preSkip:    int(binary.LittleEndian.Uint16(packet[10:12])),
		sampleRate: int(binary.LittleEndian.Uint32(packet[12:16])),
	}, nil
}

// SampleRate returns 48000 (Opus always decodes to 48kHz).
func (c *opusCodec) SampleRate() int {
	return opusSampleRate // 48000
}

// Channels returns the number of audio channels.
func (c *opusCodec) Channels() int {
	return c.channels
}

// PreSkip returns samples to skip at stream start.
func (c *opusCodec) PreSkip() int {
	return c.preSkip
}

// GranuleToSamples converts granule position to sample count (subtracts pre-skip).
func (c *opusCodec) GranuleToSamples(granule int64) int64 {
	return granule - int64(c.preSkip)
}

// Decode decodes an Opus packet into PCM samples.
func (c *opusCodec) Decode(packet []byte, pcm []float32) (samplesPerChannel int, err error) {
	return c.decoder.DecodeFloat32(packet, pcm)
}

// Reset resets decoder state.
// Opus decoder recovers from packet loss automatically, so this is a no-op.
func (c *opusCodec) Reset() error {
	return nil
}

// AddHeaderPacket is a no-op for Opus (single header already parsed).
func (c *opusCodec) AddHeaderPacket(_ []byte) (bool, error) {
	return true, nil
}

// vorbisCodec implements OggCodec for Vorbis streams.
type vorbisCodec struct {
	decoder       *vorbis.Decoder
	channels      int
	sampleRate    int
	headerPackets [][]byte // collect headers before initializing decoder
}

// newVorbisCodec creates a vorbisCodec from a Vorbis identification header.
func newVorbisCodec(packet []byte) (*vorbisCodec, error) {
	// Vorbis identification header format:
	// [0]      = packet type (0x01)
	// [1:7]    = "vorbis"
	// [7:11]   = version (must be 0)
	// [11]     = channels
	// [12:16]  = sample rate (little-endian)
	if len(packet) < 16 {
		return nil, errInvalidVorbisHeader
	}

	// Check version (must be 0)
	version := binary.LittleEndian.Uint32(packet[7:11])
	if version != 0 {
		return nil, errInvalidVorbisHeader
	}

	// Store a copy of the identification header
	identHeader := make([]byte, len(packet))
	copy(identHeader, packet)

	return &vorbisCodec{
		channels:      int(packet[11]),
		sampleRate:    int(binary.LittleEndian.Uint32(packet[12:16])),
		headerPackets: [][]byte{identHeader},
	}, nil
}

// SampleRate returns the audio sample rate from the Vorbis header.
func (c *vorbisCodec) SampleRate() int {
	return c.sampleRate
}

// Channels returns the number of audio channels.
func (c *vorbisCodec) Channels() int {
	return c.channels
}

// PreSkip returns 0 (Vorbis has no pre-skip).
func (c *vorbisCodec) PreSkip() int {
	return 0
}

// GranuleToSamples converts granule position to sample count (direct mapping for Vorbis).
func (c *vorbisCodec) GranuleToSamples(granule int64) int64 {
	return granule
}

var (
	errVorbisDecoderNotInitialized = errors.New("vorbis: decoder not initialized (headers incomplete)")
	errVorbisBufferTooSmall        = errors.New("vorbis: output buffer too small")
)

// AddHeaderPacket adds a header packet for Vorbis.
// Vorbis requires 3 header packets: identification, comment, setup.
// Returns true when all headers are received and the decoder is initialized.
// If packet is nil, just checks if all headers have been received.
func (c *vorbisCodec) AddHeaderPacket(packet []byte) (bool, error) {
	// If decoder is already initialized, we're done
	if c.decoder != nil {
		return true, nil
	}

	// If packet is nil, just check if we have all headers
	if packet == nil {
		return len(c.headerPackets) >= 3, nil
	}

	// Store a copy of the header packet
	headerCopy := make([]byte, len(packet))
	copy(headerCopy, packet)
	c.headerPackets = append(c.headerPackets, headerCopy)

	// Vorbis has 3 header packets: identification, comment, setup
	// Once we have all 3, initialize the decoder
	if len(c.headerPackets) >= 3 {
		decoder := &vorbis.Decoder{}
		for _, hdr := range c.headerPackets {
			if err := decoder.ReadHeader(hdr); err != nil {
				return false, err
			}
		}
		c.decoder = decoder
		c.headerPackets = nil // free memory
		return true, nil
	}

	return false, nil
}

// Decode decodes a Vorbis packet into PCM samples.
func (c *vorbisCodec) Decode(packet []byte, pcm []float32) (samplesPerChannel int, err error) {
	if c.decoder == nil {
		return 0, errVorbisDecoderNotInitialized
	}
	// jfreymuth/vorbis Decode returns []float32 samples (interleaved)
	samples, err := c.decoder.Decode(packet)
	if err != nil {
		return 0, err
	}
	// Ensure output buffer is large enough
	if len(pcm) < len(samples) {
		return 0, errVorbisBufferTooSmall
	}
	// Copy to output buffer
	n := copy(pcm, samples)
	return n / c.channels, nil // return samples per channel
}

// Reset resets decoder state.
// For Vorbis, we need to clear any internal decoder state after seeking.
func (c *vorbisCodec) Reset() error {
	if c.decoder != nil {
		c.decoder.Clear()
	}
	return nil
}
