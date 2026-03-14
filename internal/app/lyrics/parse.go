package lyrics

import (
	"strconv"
	"strings"
	"time"
)

// parse turns raw lyrics into lines and detects timestamped formats.
func parse(value string) ([]Line, bool) {
	value = normalize(value)
	if strings.TrimSpace(value) == "" {
		return nil, false
	}
	raw := strings.Split(value, "\n")
	lines := make([]Line, 0, len(raw))
	timed := false
	for _, line := range raw {
		if isLrcTagLine(line) {
			continue
		}
		if times, text, ok := parseLrcLine(line); ok {
			timed = true
			for _, ts := range times {
				lines = append(lines, Line{
					Text:    text,
					Time:    ts,
					HasTime: true,
				})
			}
			continue
		}
		lines = append(lines, Line{Text: line})
	}
	return lines, timed
}

func normalize(value string) string {
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return value
}

func isLrcTagLine(line string) bool {
	if !strings.HasPrefix(line, "[") {
		return false
	}
	end := strings.Index(line, "]")
	if end == -1 {
		return false
	}
	tag := line[1:end]
	if tag == "" {
		return false
	}
	parts := strings.SplitN(tag, ":", 2)
	if len(parts) != 2 {
		return false
	}
	switch strings.ToLower(parts[0]) {
	case "ar", "ti", "al", "by", "offset", "length", "re", "ve":
		return true
	default:
		return false
	}
}

func parseLrcLine(line string) ([]time.Duration, string, bool) {
	if !strings.HasPrefix(line, "[") {
		return nil, "", false
	}
	rest := line
	times := make([]time.Duration, 0, 1)
	for strings.HasPrefix(rest, "[") {
		end := strings.Index(rest, "]")
		if end == -1 {
			break
		}
		token := rest[1:end]
		ts, ok := parseTimestamp(token)
		if !ok {
			break
		}
		times = append(times, ts)
		rest = rest[end+1:]
	}
	if len(times) == 0 {
		return nil, "", false
	}
	return times, strings.TrimSpace(rest), true
}

func parseTimestamp(raw string) (time.Duration, bool) {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return 0, false
	}
	minStr := parts[0]
	secPart := parts[1]
	if minStr == "" || secPart == "" {
		return 0, false
	}
	minutes, err := strconv.Atoi(minStr)
	if err != nil {
		return 0, false
	}
	secStr := secPart
	fracStr := ""
	if before, after, ok := strings.Cut(secPart, "."); ok {
		secStr = before
		fracStr = after
	}
	sec, err := strconv.Atoi(secStr)
	if err != nil {
		return 0, false
	}
	if sec < 0 {
		return 0, false
	}
	dur := time.Duration(minutes)*time.Minute + time.Duration(sec)*time.Second
	if fracStr != "" {
		if frac, err := strconv.Atoi(fracStr); err == nil {
			switch len(fracStr) {
			case 1:
				dur += time.Duration(frac) * 100 * time.Millisecond
			case 2:
				dur += time.Duration(frac) * 10 * time.Millisecond
			default:
				dur += time.Duration(frac) * time.Millisecond
			}
		}
	}
	return dur, true
}
