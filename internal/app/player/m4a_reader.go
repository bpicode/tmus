package player

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"time"

	"github.com/abema/go-mp4"
)

// Container field names and box relationships follow ISO/IEC 14496-12.
// The ALACSpecificConfig byte layout follows Apple's ALAC specification.
var alacBoxType = mp4.StrToBoxType("alac")

const (
	// Two million AAC access units cover roughly eleven hours at 48 kHz. The
	// limit also bounds the several per-sample tables expanded while opening a
	// file, regardless of the values declared by a constant-size stsz box.
	maxM4ASampleCount = 2_000_000
	// Encoded AAC and ALAC access units are normally much smaller. Keep enough
	// headroom for unusual files without allowing one packet to consume tens of
	// MiB in both the container reader and decoder.
	maxM4ASampleSize = 4 << 20

	m4aAudioSampleEntrySize = 28
	m4aVersion1ExtraSize    = 16
	m4aVersion2ExtraSize    = 36

	alacConfigSize       = 28
	alacBitDepthOffset   = 9
	alacChannelOffset    = 13
	alacSampleRateOffset = 24
)

type m4aCodecType int

const (
	m4aCodecUnknown m4aCodecType = iota
	m4aCodecAAC
	m4aCodecALAC
)

// m4aSampleInfo describes one encoded AAC or ALAC access unit in the
// container. It does not represent an individual decoded PCM sample.
type m4aSampleInfo struct {
	Offset   uint64
	Size     uint32
	Duration uint32
}

type m4aReader struct {
	reader io.ReadSeeker

	codec       m4aCodecType
	codecConfig []byte
	sampleRate  uint32
	channels    uint8
	sampleSize  uint8
	timescale   uint32
	duration    time.Duration
	samples     []m4aSampleInfo
	// starts contains cumulative media-time units for constant-time position
	// lookup and binary-search seeking.
	starts []uint64
}

func openM4A(r io.ReadSeeker) (*m4aReader, error) {
	info, err := parseM4AContainer(r)
	if err != nil {
		return nil, err
	}
	if info.codec == m4aCodecUnknown {
		return nil, errors.New("no supported audio track found in M4A container")
	}

	starts := make([]uint64, len(info.samples))
	var total uint64
	for i, sample := range info.samples {
		starts[i] = total
		if math.MaxUint64-total < uint64(sample.Duration) {
			return nil, errors.New("M4A sample duration table overflows")
		}
		total += uint64(sample.Duration)
	}

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("reset M4A reader: %w", err)
	}

	return &m4aReader{
		reader:      r,
		codec:       info.codec,
		codecConfig: info.codecConfig,
		sampleRate:  info.sampleRate,
		channels:    info.channels,
		sampleSize:  info.sampleSize,
		timescale:   info.timescale,
		duration:    m4aUnitsToDuration(total, info.timescale),
		samples:     info.samples,
		starts:      starts,
	}, nil
}

func (r *m4aReader) Codec() m4aCodecType {
	return r.codec
}

func (r *m4aReader) CodecConfig() []byte {
	return r.codecConfig
}

func (r *m4aReader) SampleRate() uint32 {
	return r.sampleRate
}

func (r *m4aReader) Channels() uint8 {
	return r.channels
}

func (r *m4aReader) SampleSize() uint8 {
	return r.sampleSize
}

func (r *m4aReader) Duration() time.Duration {
	return r.duration
}

func (r *m4aReader) SampleCount() int {
	return len(r.samples)
}

func (r *m4aReader) ReadSample(index int) ([]byte, error) {
	if index < 0 || index >= len(r.samples) {
		return nil, fmt.Errorf("invalid sample index %d (total samples: %d)", index, len(r.samples))
	}

	sample := r.samples[index]
	if sample.Size == 0 {
		return nil, fmt.Errorf("M4A sample %d is empty", index)
	}
	if sample.Size > maxM4ASampleSize {
		return nil, fmt.Errorf("M4A sample %d is too large: %d bytes", index, sample.Size)
	}
	if sample.Offset > math.MaxInt64 {
		return nil, fmt.Errorf("M4A sample %d offset exceeds supported file size", index)
	}
	if _, err := r.reader.Seek(int64(sample.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to M4A sample %d: %w", index, err)
	}

	data := make([]byte, int(sample.Size))
	if _, err := io.ReadFull(r.reader, data); err != nil {
		return nil, fmt.Errorf("read M4A sample %d: %w", index, err)
	}
	return data, nil
}

func (r *m4aReader) SeekToTime(position time.Duration) int {
	if position <= 0 || r.timescale == 0 || len(r.samples) == 0 {
		return 0
	}

	target := m4aDurationToUnits(position, r.timescale)
	return sort.Search(len(r.samples), func(i int) bool {
		return r.starts[i]+uint64(r.samples[i].Duration) > target
	})
}

func (r *m4aReader) SampleTime(index int) time.Duration {
	if index <= 0 || r.timescale == 0 || len(r.samples) == 0 {
		return 0
	}
	if index >= len(r.samples) {
		last := len(r.samples) - 1
		return m4aUnitsToDuration(r.starts[last]+uint64(r.samples[last].Duration), r.timescale)
	}
	return m4aUnitsToDuration(r.starts[index], r.timescale)
}

func m4aDurationToUnits(d time.Duration, timescale uint32) uint64 {
	if d <= 0 || timescale == 0 {
		return 0
	}
	seconds := uint64(d / time.Second)
	if seconds > math.MaxUint64/uint64(timescale) {
		return math.MaxUint64
	}
	whole := seconds * uint64(timescale)
	fraction := uint64(d % time.Second)
	return whole + fraction*uint64(timescale)/uint64(time.Second)
}

func m4aUnitsToDuration(units uint64, timescale uint32) time.Duration {
	if timescale == 0 {
		return 0
	}

	seconds := units / uint64(timescale)
	if seconds > uint64(math.MaxInt64/int64(time.Second)) {
		return time.Duration(math.MaxInt64)
	}
	remainder := units % uint64(timescale)
	whole := time.Duration(seconds) * time.Second
	fraction := time.Duration(remainder * uint64(time.Second) / uint64(timescale))
	if fraction > time.Duration(math.MaxInt64)-whole {
		return time.Duration(math.MaxInt64)
	}
	return whole + fraction
}

type m4aParseResult struct {
	codec       m4aCodecType
	codecConfig []byte
	sampleRate  uint32
	channels    uint8
	sampleSize  uint8
	timescale   uint32
	samples     []m4aSampleInfo
}

type m4aSampleEntry struct {
	codec       m4aCodecType
	config      []byte
	sampleRate  uint32
	channels    uint8
	sampleSize  uint8
	description uint32
}

func parseM4AContainer(r io.ReadSeeker) (*m4aParseResult, error) {
	fileSize, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, fmt.Errorf("determine M4A size: %w", err)
	}
	if fileSize < 0 {
		return nil, errors.New("invalid negative M4A size")
	}

	tracks, err := mp4.ExtractBox(r, nil, mp4.BoxPath{mp4.BoxTypeMoov(), mp4.BoxTypeTrak()})
	if err != nil {
		return nil, fmt.Errorf("read M4A track list: %w", err)
	}

	for _, track := range tracks {
		info, found, err := parseM4ATrack(r, track, uint64(fileSize))
		if err != nil {
			return nil, err
		}
		if found {
			return info, nil
		}
	}
	return &m4aParseResult{}, nil
}

func parseM4ATrack(r io.ReadSeeker, track *mp4.BoxInfo, fileSize uint64) (*m4aParseResult, bool, error) {
	// Inspect the small track header first. Sample tables can be large, so do
	// not unmarshal them until the handler confirms this is an audio track.
	header, err := mp4.ExtractBoxesWithPayload(r, track, []mp4.BoxPath{
		{mp4.BoxTypeMdia(), mp4.BoxTypeHdlr()},
		{mp4.BoxTypeMdia(), mp4.BoxTypeMdhd()},
	})
	if err != nil {
		return nil, false, fmt.Errorf("read M4A track header: %w", err)
	}

	hdlr, ok := m4aPayload[*mp4.Hdlr](header, mp4.BoxTypeHdlr())
	if !ok || hdlr.HandlerType != [4]byte{'s', 'o', 'u', 'n'} {
		return nil, false, nil
	}
	mdhd, ok := m4aPayload[*mp4.Mdhd](header, mp4.BoxTypeMdhd())
	if !ok || mdhd.Timescale == 0 {
		return nil, false, errors.New("audio track has no valid mdhd timescale")
	}

	entry, found, err := parseM4ASampleDescription(r, track)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}

	boxes, err := mp4.ExtractBoxesWithPayload(r, track, m4aSampleTableBoxPaths())
	if err != nil {
		return nil, false, fmt.Errorf("read M4A sample tables: %w", err)
	}
	sizes, err := m4aSampleSizes(boxes)
	if err != nil {
		return nil, false, err
	}
	offsets, err := m4aChunkOffsets(boxes)
	if err != nil {
		return nil, false, err
	}
	stsc, ok := m4aPayload[*mp4.Stsc](boxes, mp4.BoxTypeStsc())
	if !ok {
		return nil, false, errors.New("audio track has no stsc box")
	}
	stts, ok := m4aPayload[*mp4.Stts](boxes, mp4.BoxTypeStts())
	if !ok {
		return nil, false, errors.New("audio track has no stts box")
	}

	samples, err := buildM4ASampleTable(sizes, offsets, stsc.Entries, stts.Entries, entry.description)
	if err != nil {
		return nil, false, fmt.Errorf("build M4A sample table: %w", err)
	}
	for i, sample := range samples {
		end := sample.Offset + uint64(sample.Size)
		if end < sample.Offset || end > fileSize {
			return nil, false, fmt.Errorf("M4A sample %d points outside the file", i)
		}
	}

	return &m4aParseResult{
		codec:       entry.codec,
		codecConfig: entry.config,
		sampleRate:  entry.sampleRate,
		channels:    entry.channels,
		sampleSize:  entry.sampleSize,
		timescale:   mdhd.Timescale,
		samples:     samples,
	}, true, nil
}

func m4aSampleTableBoxPaths() []mp4.BoxPath {
	stbl := mp4.BoxPath{mp4.BoxTypeMdia(), mp4.BoxTypeMinf(), mp4.BoxTypeStbl()}
	path := func(boxType mp4.BoxType) mp4.BoxPath {
		return append(append(mp4.BoxPath(nil), stbl...), boxType)
	}
	return []mp4.BoxPath{
		path(mp4.BoxTypeStsz()),
		path(mp4.BoxTypeStco()),
		path(mp4.BoxTypeCo64()),
		path(mp4.BoxTypeStsc()),
		path(mp4.BoxTypeStts()),
	}
}

func m4aPayload[T any](boxes []*mp4.BoxInfoWithPayload, boxType mp4.BoxType) (T, bool) {
	var zero T
	for _, box := range boxes {
		if box.Info.Type != boxType {
			continue
		}
		payload, ok := box.Payload.(T)
		return payload, ok
	}
	return zero, false
}

func m4aSampleSizes(boxes []*mp4.BoxInfoWithPayload) ([]uint32, error) {
	stsz, ok := m4aPayload[*mp4.Stsz](boxes, mp4.BoxTypeStsz())
	if !ok || stsz.SampleCount == 0 {
		return nil, errors.New("audio track has no samples")
	}
	if stsz.SampleCount > maxM4ASampleCount {
		return nil, fmt.Errorf("M4A sample count %d exceeds limit %d", stsz.SampleCount, maxM4ASampleCount)
	}
	if stsz.SampleSize == 0 {
		if uint32(len(stsz.EntrySize)) != stsz.SampleCount {
			return nil, errors.New("stsz entry count does not match sample count")
		}
		for i, size := range stsz.EntrySize {
			if size == 0 {
				return nil, fmt.Errorf("M4A sample %d is empty", i)
			}
			if size > maxM4ASampleSize {
				return nil, fmt.Errorf("M4A sample %d is too large: %d bytes", i, size)
			}
		}
		return stsz.EntrySize, nil
	}
	if stsz.SampleSize > maxM4ASampleSize {
		return nil, fmt.Errorf("M4A samples are too large: %d bytes", stsz.SampleSize)
	}

	sizes := make([]uint32, int(stsz.SampleCount))
	for i := range sizes {
		sizes[i] = stsz.SampleSize
	}
	return sizes, nil
}

func m4aChunkOffsets(boxes []*mp4.BoxInfoWithPayload) ([]uint64, error) {
	stco, has32 := m4aPayload[*mp4.Stco](boxes, mp4.BoxTypeStco())
	co64, has64 := m4aPayload[*mp4.Co64](boxes, mp4.BoxTypeCo64())
	if has32 && has64 {
		return nil, errors.New("audio track contains both stco and co64 boxes")
	}
	if has64 {
		return co64.ChunkOffset, nil
	}
	if !has32 {
		return nil, errors.New("audio track has no chunk offsets")
	}

	offsets := make([]uint64, len(stco.ChunkOffset))
	for i, offset := range stco.ChunkOffset {
		offsets[i] = uint64(offset)
	}
	return offsets, nil
}

func parseM4ASampleDescription(r io.ReadSeeker, track *mp4.BoxInfo) (m4aSampleEntry, bool, error) {
	entries, err := mp4.ExtractBox(r, track, mp4.BoxPath{
		mp4.BoxTypeMdia(), mp4.BoxTypeMinf(), mp4.BoxTypeStbl(), mp4.BoxTypeStsd(), mp4.BoxTypeAny(),
	})
	if err != nil {
		return m4aSampleEntry{}, false, fmt.Errorf("read M4A sample descriptions: %w", err)
	}

	for i, box := range entries {
		if box.Type != mp4.BoxTypeMp4a() && box.Type != alacBoxType {
			continue
		}
		entry, err := parseM4AAudioSampleEntry(r, box)
		if err != nil {
			return m4aSampleEntry{}, false, err
		}
		entry.description = uint32(i + 1)
		return entry, true, nil
	}
	return m4aSampleEntry{}, false, nil
}

func parseM4AAudioSampleEntry(r io.ReadSeeker, box *mp4.BoxInfo) (m4aSampleEntry, error) {
	// An ISO audio sample entry starts with a 28-byte header. QuickTime
	// versions 1 and 2 append extra fields before the child codec boxes.
	if box.Size < box.HeaderSize+m4aAudioSampleEntrySize || box.Offset > math.MaxInt64 || box.Size > math.MaxInt64-box.Offset {
		return m4aSampleEntry{}, fmt.Errorf("invalid %s audio sample entry size", box.Type)
	}
	if _, err := box.SeekToPayload(r); err != nil {
		return m4aSampleEntry{}, fmt.Errorf("seek to %s sample entry: %w", box.Type, err)
	}

	header := make([]byte, m4aAudioSampleEntrySize)
	if _, err := io.ReadFull(r, header); err != nil {
		return m4aSampleEntry{}, fmt.Errorf("read %s sample entry: %w", box.Type, err)
	}
	version := binary.BigEndian.Uint16(header[8:10])
	extra := uint64(0)
	switch version {
	case 0:
	case 1:
		extra = m4aVersion1ExtraSize
	case 2:
		extra = m4aVersion2ExtraSize
	default:
		return m4aSampleEntry{}, fmt.Errorf("unsupported audio sample entry version %d", version)
	}

	childrenStart := box.Offset + box.HeaderSize + m4aAudioSampleEntrySize + extra
	childrenEnd := box.Offset + box.Size
	if childrenStart > childrenEnd {
		return m4aSampleEntry{}, fmt.Errorf("truncated %s audio sample entry", box.Type)
	}

	channelCount := binary.BigEndian.Uint16(header[16:18])
	if channelCount != 1 && channelCount != 2 {
		return m4aSampleEntry{}, fmt.Errorf("unsupported %s channel count %d", box.Type, channelCount)
	}
	entry := m4aSampleEntry{
		// ISO stores the sample rate as an unsigned 16.16 fixed-point value.
		sampleRate: binary.BigEndian.Uint32(header[24:28]) >> 16,
		channels:   uint8(channelCount),
		sampleSize: uint8(binary.BigEndian.Uint16(header[18:20])),
	}
	if entry.sampleRate == 0 || entry.channels == 0 {
		return m4aSampleEntry{}, fmt.Errorf("invalid %s audio format", box.Type)
	}

	switch box.Type {
	case mp4.BoxTypeMp4a():
		entry.codec = m4aCodecAAC
		entry.sampleSize = 16
		config, err := readM4AAACConfig(r, childrenStart, childrenEnd)
		if err != nil {
			return m4aSampleEntry{}, err
		}
		entry.config = config
	case alacBoxType:
		entry.codec = m4aCodecALAC
		config, err := readM4AALACConfig(r, childrenStart, childrenEnd)
		if err != nil {
			return m4aSampleEntry{}, err
		}
		entry.config = config
		entry.sampleSize = config[alacBitDepthOffset]
		entry.channels = config[alacChannelOffset]
		entry.sampleRate = binary.BigEndian.Uint32(config[alacSampleRateOffset:alacConfigSize])
		if entry.sampleSize != 16 && entry.sampleSize != 24 {
			return m4aSampleEntry{}, fmt.Errorf("unsupported ALAC sample size %d", entry.sampleSize)
		}
		if entry.channels != 1 && entry.channels != 2 {
			return m4aSampleEntry{}, fmt.Errorf("unsupported ALAC channel count %d", entry.channels)
		}
	}
	return entry, nil
}

func readM4AAACConfig(r io.ReadSeeker, start, end uint64) ([]byte, error) {
	box, err := findM4AChildBox(r, start, end, mp4.BoxTypeEsds())
	if err != nil {
		return nil, fmt.Errorf("find AAC esds box: %w", err)
	}
	if box == nil {
		return nil, errors.New("AAC sample entry has no esds box")
	}
	if _, err := box.SeekToPayload(r); err != nil {
		return nil, fmt.Errorf("seek to AAC esds box: %w", err)
	}
	payload, _, err := mp4.UnmarshalAny(r, box.Type, box.Size-box.HeaderSize, mp4.Context{})
	if err != nil {
		return nil, fmt.Errorf("parse AAC esds box: %w", err)
	}
	esds, ok := payload.(*mp4.Esds)
	if !ok {
		return nil, errors.New("unexpected AAC esds payload type")
	}
	for _, descriptor := range esds.Descriptors {
		if descriptor.Tag == mp4.DecSpecificInfoTag && len(descriptor.Data) > 0 {
			return append([]byte(nil), descriptor.Data...), nil
		}
	}
	return nil, errors.New("AAC esds box has no decoder-specific configuration")
}

func readM4AALACConfig(r io.ReadSeeker, start, end uint64) ([]byte, error) {
	box, err := findM4AChildBox(r, start, end, alacBoxType)
	if err != nil {
		return nil, fmt.Errorf("find ALAC configuration box: %w", err)
	}
	if box == nil || box.Size-box.HeaderSize < alacConfigSize {
		return nil, errors.New("ALAC sample entry has no valid configuration box")
	}
	if _, err := box.SeekToPayload(r); err != nil {
		return nil, fmt.Errorf("seek to ALAC configuration: %w", err)
	}
	config := make([]byte, alacConfigSize)
	if _, err := io.ReadFull(r, config); err != nil {
		return nil, fmt.Errorf("read ALAC configuration: %w", err)
	}
	// ALACSpecificConfig places bit depth at byte 9, channels at byte 13,
	// and the big-endian sample rate in the final four bytes.
	if config[alacBitDepthOffset] == 0 || config[alacChannelOffset] == 0 || binary.BigEndian.Uint32(config[alacSampleRateOffset:alacConfigSize]) == 0 {
		return nil, errors.New("ALAC configuration contains an invalid audio format")
	}
	return config, nil
}

func findM4AChildBox(r io.ReadSeeker, start, end uint64, boxType mp4.BoxType) (*mp4.BoxInfo, error) {
	if start > end || end > math.MaxInt64 {
		return nil, errors.New("invalid child box bounds")
	}
	if _, err := r.Seek(int64(start), io.SeekStart); err != nil {
		return nil, err
	}

	for offset := start; end-offset >= mp4.SmallHeaderSize; {
		box, err := mp4.ReadBoxInfo(r)
		if err != nil {
			return nil, err
		}
		boxEnd := box.Offset + box.Size
		if box.Size < box.HeaderSize || boxEnd < box.Offset || boxEnd > end {
			return nil, fmt.Errorf("invalid %s child box size", box.Type)
		}
		if box.Type == boxType {
			return box, nil
		}
		if _, err := box.SeekToEnd(r); err != nil {
			return nil, err
		}
		offset = boxEnd
	}
	if start != end {
		pos, err := r.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, err
		}
		if uint64(pos) != end {
			return nil, errors.New("trailing bytes in audio sample entry")
		}
	}
	return nil, nil
}

func buildM4ASampleTable(sampleSizes []uint32, chunkOffsets []uint64, stscEntries []mp4.StscEntry, sttsEntries []mp4.SttsEntry, description uint32) ([]m4aSampleInfo, error) {
	// stsz supplies encoded sample sizes, stco/co64 locates chunks, stsc maps
	// each chunk to a sample count and description, and stts supplies the
	// duration of every sample in media-timescale units.
	if len(sampleSizes) == 0 || len(chunkOffsets) == 0 {
		return nil, errors.New("sample or chunk table is empty")
	}
	if len(stscEntries) == 0 || stscEntries[0].FirstChunk != 1 {
		return nil, errors.New("stsc table must begin at chunk 1")
	}
	for i, entry := range stscEntries {
		if entry.SamplesPerChunk == 0 || entry.SampleDescriptionIndex != description {
			return nil, fmt.Errorf("unsupported stsc entry %d", i)
		}
		if i > 0 && entry.FirstChunk <= stscEntries[i-1].FirstChunk {
			return nil, errors.New("stsc chunk ranges are not increasing")
		}
	}

	durations := make([]uint32, 0, len(sampleSizes))
	for _, entry := range sttsEntries {
		if entry.SampleDelta == 0 || uint64(entry.SampleCount) > uint64(len(sampleSizes)-len(durations)) {
			return nil, errors.New("stts entries do not match sample count")
		}
		for range entry.SampleCount {
			durations = append(durations, entry.SampleDelta)
		}
	}
	if len(durations) != len(sampleSizes) {
		return nil, errors.New("stts entries do not cover every sample")
	}

	samples := make([]m4aSampleInfo, 0, len(sampleSizes))
	sampleIndex := 0
	stscIndex := 0
	for chunkIndex, chunkOffset := range chunkOffsets {
		chunkNumber := uint32(chunkIndex + 1)
		if stscIndex+1 < len(stscEntries) && chunkNumber >= stscEntries[stscIndex+1].FirstChunk {
			stscIndex++
		}
		for range stscEntries[stscIndex].SamplesPerChunk {
			if sampleIndex == len(sampleSizes) {
				return nil, errors.New("chunk table describes too many samples")
			}
			size := sampleSizes[sampleIndex]
			samples = append(samples, m4aSampleInfo{
				Offset:   chunkOffset,
				Size:     size,
				Duration: durations[sampleIndex],
			})
			if math.MaxUint64-chunkOffset < uint64(size) {
				return nil, errors.New("sample offset overflows")
			}
			chunkOffset += uint64(size)
			sampleIndex++
		}
	}
	if sampleIndex != len(sampleSizes) {
		return nil, errors.New("chunk table does not cover every sample")
	}
	return samples, nil
}
