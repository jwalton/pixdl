package download

import (
	"net/http"
	"sync/atomic"
	"time"
)

const minTimeBetweenProgressReports = 0

// Returns a new progressWriter.  Anything written to the writer will cause
// progress events to be sent to the ProgressReporter.
func newProgressWriter(
	request *http.Request,
	file string,
	remoteFileInfo *RemoteFileInfo,
	reporter FileProgressCallback,
) *progressWriter {
	progress := newProgress(request, request.URL.String(), file, remoteFileInfo)
	return &progressWriter{progress, reporter, 0}
}

// progressWriter writes progress reports to `progress` periodically.
type progressWriter struct {
	progress *Progress
	reporter FileProgressCallback
	// If non-zero, progress reports should not be sent.
	blockProgress int32
}

func (pw *progressWriter) setSize(size int64) {
	pw.progress.Written = size
	pw.reportProgress()
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.progress.Written += int64(n)

	pw.reportProgress()
	return n, nil
}

func (pw *progressWriter) Warn(message string) {
	pw.progress.Warning = message
	pw.reporter(pw.progress)
	pw.progress.Warning = ""
}

func (pw *progressWriter) Close(err error) {
	pw.progress.Err = err
	pw.progress.Done = true
	pw.reporter(pw.progress)
}

func (pw *progressWriter) reportProgress() {
	blockProgress := atomic.LoadInt32(&pw.blockProgress)
	if blockProgress == 0 && pw.reporter != nil {
		// Update PercentComplete.
		var complete float64 = -1
		if pw.progress.Total > -1 {
			complete = float64(pw.progress.Written) / float64(pw.progress.Total) * 100.0
		}
		pw.progress.PercentComplete = complete

		// Send progress report
		pw.reporter(pw.progress)

		// Block progress for next minTimeBetweenProgressReports
		if blockProgress == 0 {
			atomic.StoreInt32(&pw.blockProgress, 1)
			time.AfterFunc(minTimeBetweenProgressReports, func() {
				atomic.StoreInt32(&pw.blockProgress, 0)
			})
		}
	}
}
