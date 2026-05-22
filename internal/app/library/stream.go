package library

import (
	"bufio"
	"io"
	"os"
	"strings"
)

func parseURLShortcut(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	inSection := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.ToLower(line)
			if section == "[internetshortcut]" {
				inSection = true
			} else {
				inSection = false
			}
			continue
		}
		if inSection || !strings.Contains(line, "[") {
			if strings.HasPrefix(strings.ToLower(line), "url=") {
				return strings.TrimSpace(line[4:]), nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", os.ErrNotExist
}

func parseStreamShortcut(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		return line, nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", os.ErrNotExist
}
