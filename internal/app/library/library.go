package library

// OpenArchiveRoot returns an archive URI for the root of the archive if the path is a supported archive file.
func OpenArchiveRoot(path string) (string, bool) {
	handler := archiveHandlers().findHandler(path)
	if handler == nil {
		return "", false
	}
	return BuildArchivePath(handler.Scheme(), path, ""), true
}
