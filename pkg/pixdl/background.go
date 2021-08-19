package pixdl

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/jwalton/pixdl/pkg/providers/env"
)

// TODO: Want options for things like login credentials, user agent, etc...

const defaultMaxConcurrency = 4

// DownloadOptions is an object that can be passed to an ImageDownloader to
// specify options when downloading an album.
type DownloadOptions struct {
	// MaxImages is the maximum number of images to download.  0 for all.
	MaxImages int
	// MaxPages is the maximum number of pages to download.  0 for all.
	MaxPages int
	// ToFolder is the destination folder to download images to.
	ToFolder string
	// FilenameTemplate is a golang template for generating the filename to write to.
	FilenameTemplate string
	// FilterSubAlbum is the name of the subalbum to filter.  If this is non-empty,
	// then only images from the specified SubAlbum will be downloaded.
	FilterSubAlbum string
}

// ImageDownloader is an object that can download images.
type ImageDownloader interface {
	// DownloadAlbum will download all images in an album.
	DownloadAlbum(
		url string,
		options DownloadOptions,
		reporter ProgressReporter,
	)

	// DownloadImage will download an individual image from an album.
	DownloadImage(
		image *ImageMetadata,
		toFolder string,
		filenameTemplate string,
		reporter ProgressReporter,
	)

	// Wait will block until all albums/images currently being downloaded are
	// done downloading.
	Wait()

	// Close will shut down an ImageDownloader and prevent any further downloads.
	Close()

	// IsClosed will return true if this downloader has been closed.
	IsClosed() bool

	getEnv() *env.Env
}

type downloadRequest struct {
	image            *ImageMetadata
	toFolder         string
	filenameTemplate string
	reporter         ProgressReporter
}

type concurrentDownloader struct {
	env            *env.Env
	ch             chan *downloadRequest
	albumWg        *sync.WaitGroup
	imageWg        *sync.WaitGroup
	closed         int32
	maxConcurrency uint
	minSize        int64
}

// Option is an option that can be passed to NewConcurrnetDownloader().
type Option func(*concurrentDownloader)

// SetMinSize is an option that sets the minimum size, in bytes, of files to download.
// Any file smaller than this will be skipped. 0 for any size.  If the size of a
// file cannot be determined ahead of time, this will be ignored.
func SetMinSize(size int64) Option {
	return func(dl *concurrentDownloader) {
		dl.minSize = size
	}
}

// SetMaxConcurrency is an option for NewConcurrnetDownloader which sets the
// maximum number of goroutines which will be spawned to download files.
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
		env:     &env.Env{DownloadClient: client},
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
func (downloader *concurrentDownloader) startImageWorker(ch <-chan *downloadRequest, workerID uint) {
	done := false
	for !done {
		req := <-ch
		if req != nil {
			downloadImage(
				downloader.env,
				req.image,
				req.toFolder,
				req.filenameTemplate,
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
	options DownloadOptions,
	reporter ProgressReporter,
) {
	downloader.albumWg.Add(1)

	err := validateTemplate(options.FilenameTemplate)
	if err != nil {
		reporter.ImageEnd(nil, err)
	}

	go func() {
		downloadAlbum(downloader, url, options, reporter)
		downloader.albumWg.Done()
	}()
}

func (downloader *concurrentDownloader) DownloadImage(
	image *ImageMetadata,
	toFolder string,
	filenameTemplate string,
	reporter ProgressReporter,
) {
	if downloader.IsClosed() {
		reporter.ImageSkip(image, fmt.Errorf("downloader closed"))
	} else {
		downloader.imageWg.Add(1)
		downloader.ch <- &downloadRequest{image, toFolder, filenameTemplate, reporter}
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

func (downloader *concurrentDownloader) getEnv() *env.Env {
	return downloader.env
}
