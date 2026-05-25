package player

import "github.com/bpicode/tmus/internal/app/library"

func sourceFromAudioSource(source library.AudioSource) library.Source {
	return library.Source{Reader: source.Reader, Ext: extFromFormat(source.Format)}
}

func extFromFormat(format library.FormatType) string {
	switch format {
	case library.FormatMP3:
		return ".mp3"
	case library.FormatFLAC:
		return ".flac"
	case library.FormatOGG:
		return ".ogg"
	case library.FormatOPUS:
		return ".opus"
	case library.FormatOGA:
		return ".oga"
	case library.FormatM4A:
		return ".m4a"
	case library.FormatMP4:
		return ".mp4"
	case library.FormatWAV:
		return ".wav"
	default:
		return ""
	}
}
