package pixdl

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// TODO: Want options for things like login credentials, user agent, etc...

const defaultMaxConcurrency = 4

// ImageDownloader is an object that can download images.
type ImageDownloader interface {
	// DownloadAlbum will download all images in an album.
	DownloadAlbum(
		url string,
		toFolder string,
		reporter ProgressReporter,
	)

	// DownloadImage will download an individual image from an album.
	DownloadImage(
		image *ImageMetadata,
		toFolder string,
		reporter ProgressReporter,
	)

	// Wait will block until all albums/images currently being downloaded are
	// done downloading.
	Wait()

	// Close will shut down an ImageDownloader and prevent any further downloads.
	Close()

	// IsClosed will return true if this downloader has been closed.
	IsClosed() bool
}

type downloadRequest struct {
	image    *ImageMetadata
	toFolder string
	reporter ProgressReporter
}

type concurrentDownloader struct {
	ch             chan *downloadRequest
	albumWg        *sync.WaitGroup
	imageWg        *sync.WaitGroup
	closed         int32
	maxConcurrency uint
	minSize        int64
}

type Option func(*concurrentDownloader)

// SetMinSize is an option that sets the minimum size, in bytes, of files to download.
// Any file smaller than this will be skipped. 0 for any size.  If the size of a
// file cannot be determined ahead of time, this will be ignored.
func SetMinSize(size int64) Option {
	return func(dl *concurrentDownloader) {
		dl.minSize = size
	}
}

func SetMaxConcurrency(maxConcurrency uint) Option {
	return func(dl *concurrentDownloader) {
		if dl.ch == nil && maxConcurrency > 0 {
			dl.maxConcurrency = uint(maxConcurrency)
			dl.ch = make(chan *downloadRequest, maxConcurrency*10)
		}
	}
}

// NewConcurrnetDownloader returns an instance of ImageDownloader which will
// download multiple images simultaneously in goroutines.  `maxConcurrent` is
// the maximum number of concurrent downloads to allow at the same time.
func NewConcurrnetDownloader(options ...Option) ImageDownloader {
	downloader := &concurrentDownloader{
		ch:      nil,
		albumWg: &sync.WaitGroup{},
		imageWg: &sync.WaitGroup{},
	}

	for _, option := range options {
		option(downloader)
	}

	if downloader.ch == nil {
		SetMaxConcurrency(defaultMaxConcurrency)(downloader)
	}

	for i := uint(0); i < downloader.maxConcurrency; i++ {
		go downloader.startImageWorker(downloader.ch, i)
	}

	return downloader
}

// startImageWorker will start a worker that listens to the specified
// channel, and downloads any images sent to the channel.  If the channel
// closes, the image worker will finish up the images it is working on, and
// then terminate.
func (downloader *concurrentDownloader) startImageWorker(ch <-chan *downloadRequest, workerId uint) {
	done := false
	for !done {
		req := <-ch
		if req != nil {
			downloadImage(
				req.image,
				req.toFolder,
				downloader.minSize,
				req.reporter,
			)
			downloader.imageWg.Done()
		} else {
			done = true
		}
	}
}

func (downloader *concurrentDownloader) DownloadAlbum(
	url string,
	toFolder string,
	reporter ProgressReporter,
) {
	downloader.albumWg.Add(1)
	go func() {
		downloadAlbum(downloader, url, toFolder, reporter)
		downloader.albumWg.Done()
	}()
}

func (downloader *concurrentDownloader) DownloadImage(
	image *ImageMetadata,
	toFolder string,
	reporter ProgressReporter,
) {
	if downloader.IsClosed() {
		reporter.ImageSkip(image, fmt.Errorf("Downloader closed"))
	} else {
		downloader.imageWg.Add(1)
		downloader.ch <- &downloadRequest{image, toFolder, reporter}
	}
}

func (downloader *concurrentDownloader) Wait() {
	// Wait for any album threads to finish adding images...
	downloader.albumWg.Wait()
	// Wait for all images to finish downloading...
	downloader.imageWg.Wait()
}

func (downloader *concurrentDownloader) Close() {
	atomic.StoreInt32(&downloader.closed, 1)
	// Stop the workers...
	close(downloader.ch)
	// Block until everything is done.
	downloader.imageWg.Wait()
}

func (downloader *concurrentDownloader) IsClosed() bool {
	return atomic.LoadInt32(&downloader.closed) == 1
}
