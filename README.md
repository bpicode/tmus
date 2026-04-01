# tmus

`tmus` is a minimal terminal music player written in Go.

![Demo usage](/images/demo.gif?raw=true "Demo usage")

## Features

- Supports MP3, WAV, FLAC, Ogg Vorbis, Opus, and M4A/MP4
- Plays audio directly from zip, tar, tar.gz/tgz, tar.xz/txz, rar, and 7z archives
- Keyboard-driven playback controls (play/pause/next/prev/stop)
- Multiple queue modes (Linear, Repeat All, Repeat One, Shuffle, Stop After Current)
- Lyrics overlay (supports embedded metadata, sidecar `.lrc`/`.txt`, and online fetching via LRCLIB)
- Single-instance handoff on Unix: opening files in a new `tmus` process forwards them to the running instance
- MPRIS integration on Linux for external media controls
- Configurable color palette

## Installation

### Binaries

Pre-compiled binaries for various platforms are available on the [Releases](https://github.com/bpicode/tmus/releases) page.

### Quick install from source (cross-platform)

Linux build/runtime note: you need `pkg-config` and ALSA development headers (`alsa.pc`, usually from `libasound2-dev`).

```bash
go install github.com/bpicode/tmus@latest
```

### Full install from source (Linux)

For Linux environments, a `Makefile` is provided. This installs the binary to `~/.local/bin` and adds a `.desktop` file and icons for launcher integration.

```bash
git clone https://github.com/bpicode/tmus.git
cd tmus
make install
```


## Usage

Show CLI options:

```bash
tmus --help
```

Initialize a default config file:

```bash
tmus config init
```

Print effective config:

```bash
tmus config show
```

## Limitations

- No library indexing. This is an intentional non-goal at this point, `tmus` stays lightweight and file-based.
- Archive entries are loaded fully into memory before playback. This is usually fine for typical track sizes (tens to low hundreds of MB), but very large files will use significant RAM.

## FAQ

- **Will you ever support less common audio formats?**  
  It depends. If there’s a reasonably maintained Go library—ideally a pure Go implementation—I’m open to trying it. Otherwise, probably not, though contributions are welcome.
- **What operating systems are supported?**  
  There’s no formal support matrix yet. `tmus` should work on platforms supported by Go and its audio dependencies, but it’s only been tested on my setup. If you run into OS-specific issues, please open an issue.
- **What terminals are supported?**  
  Any modern ANSI/VT-compatible terminal should work. For the best experience, use a monospaced font with good box-drawing support; non-monospaced fonts can cause alignment issues. Very small terminal sizes can also truncate the UI.
- **Was AI used in this project? Will that be cited as an excuse for bugs?**  
  Yes.  
  Maybe.

## Similar projects

- [waves](https://github.com/llehouerou/waves)
- [gomu](https://github.com/raziman18/gomu)
- [cliamp](https://github.com/bjarneo/cliamp)
- [termusic](https://github.com/tramhao/termusic)
