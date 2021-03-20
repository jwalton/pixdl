package download

import (
	"sync/atomic"

	"github.com/cavaliercoder/grab"
)

// Response is a thread-safe object for reading data about a file transfer in progress.
type Response struct {
	// URL is the URL this file is being downloaded from.
	URL string
	// File is the full path of the file on disk.
	File string
	resp *grab.Response
	done int32
}

// Size returns the total size of this file, or -1 if the size is unknown.
func (r *Response) Size() int64 {
	return r.resp.Size()
}

// Written returns the number of bytes written to disk.
func (r *Response) Written() int64 {
	return r.resp.BytesComplete()
}

// Done returns true if this file transfer is complete.
func (r *Response) Done() bool {
	return atomic.LoadInt32(&r.done) == 1
}

func (r *Response) setDone() {
	atomic.StoreInt32(&r.done, 1)
}

// Err returns an error if this transfer failed, or nil if
// successful or not complete yet.
func (r *Response) Err() error {
	if r.Done() {
		return r.resp.Err()
	} else {
		return nil
	}
}
