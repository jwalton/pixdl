package download

import "net/http"

// Progress represents progress downloading a file.
type Progress struct {
	// Request is the request for the outgoing file.  Note that in some error
	// scenarios, this may be nil.
	Request *http.Request
	// URL is the URL we are downloading from.
	URL string
	// File is the file we are writing to.
	File string
	// Total is the total size of the file, or -1 if unknown.
	Total int64
	// Written is the number of bytes written to disk (including any bytes that
	// were already on disk if the file was resumed).
	Written int64
	// PercentComplete is a value between 0 and 100 indicating completion progress.
	// If the total size is unknown, PercentComplete will be -1.
	PercentComplete float64
	// Done is true if the download is complete, false otherwise.
	Done bool
	// Err will always be nil if Done is false, otherwise if set it is an error
	// which caused the download to fail.
	Err error
	// If warning is present, it signifies a non-fatal error ocurred.
	Warning string
}

// FileProgressCallback is function that will be called with updates as a file downloads.
// Note that the exact same instance of `Progress` is updated and passed every
// time this is called to minimize allocations - if you want to write a Progress
// to a channel or send it to another thread, you should make a copy of it.
type FileProgressCallback func(progress *Progress)

func newProgress(request *http.Request, url string, filename string, remoteFileInfo *RemoteFileInfo) *Progress {
	return newErrorProgress(request, url, filename, remoteFileInfo, nil)
}

func newErrorProgress(request *http.Request, url string, filename string, remoteFileInfo *RemoteFileInfo, err error) *Progress {
	if remoteFileInfo == nil {
		remoteFileInfo = newRemoteFileInfo()
	}

	if request != nil {
		url = request.URL.String()
	}

	return &Progress{
		Request:         request,
		URL:             url,
		File:            filename,
		Total:           remoteFileInfo.Size,
		Written:         0,
		PercentComplete: 0,
		Done:            false,
		Err:             err,
		Warning:         "",
	}
}
