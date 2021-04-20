package reporters

import (
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/jwalton/gchalk"
	"github.com/jwalton/pixdl/pkg/download"
	"github.com/jwalton/pixdl/pkg/pixdl"
)

type verboseReporter struct {
	mutex sync.Mutex
}

func (p *verboseReporter) log(message string, a ...interface{}) {
	p.mutex.Lock()
	fmt.Println(fmt.Sprintf(message, a...))
	p.mutex.Unlock()
}

func (p *verboseReporter) logError(message string, a ...interface{}) {
	p.mutex.Lock()
	os.Stderr.WriteString(gchalk.Stderr.BrightRed(fmt.Sprintf(message, a...) + "\n"))
	p.mutex.Unlock()
}

func (p *verboseReporter) getItemLabel(image *pixdl.ImageMetadata) string {
	outOf := "??"
	if image.Album.TotalImageCount != -1 {
		outOf = fmt.Sprintf("%d", image.Album.TotalImageCount)
	}
	return fmt.Sprintf("%s (%d/%s)", path.Base(image.Filename), image.Index+1, outOf)
}

func (p *verboseReporter) AlbumFetch(url string) {
	p.log("Fetching album: %s", url)
}

func (p *verboseReporter) AlbumStart(album *pixdl.AlbumMetadata) {
	name := album.Name
	if name == "" {
		name = album.URL
	}
	p.log("Starting album: %s (%s)", name, album.Provider)
}

func (p *verboseReporter) AlbumEnd(album *pixdl.AlbumMetadata, err error) {
	name := album.Name
	if name == "" {
		name = album.URL
	}
	if err == nil {
		p.log("Done album: %s", name)
	} else {
		p.logError("Error downloading album: %s: %v", name, err)
	}
}

func (p *verboseReporter) ImageSkip(image *pixdl.ImageMetadata, err error) {
	if err != nil {
		p.log("Skipping:    %s: %s", p.getItemLabel(image), err)
	} else {
		p.log("Skipping:    %s", p.getItemLabel(image))
	}
}

func (p *verboseReporter) ImageStart(image *pixdl.ImageMetadata) {
	p.log("Downloading: %s", p.getItemLabel(image))
}

func (p *verboseReporter) ImageProgress(image *pixdl.ImageMetadata, progress *download.Progress) {
	if progress.Done {
		p.log("Downloaded: %s %v/%v bytes", p.getItemLabel(image), progress.Written, progress.Total)
	}
}

func (p *verboseReporter) ImageEnd(image *pixdl.ImageMetadata, err error) {
	if err != nil {
		p.logError("Error:       %s: %v", p.getItemLabel(image), err)
	}
}

// NewVerboseReporter returns a new ProgressReporter which logs all activity to stdout.
func NewVerboseReporter() pixdl.ProgressReporter {
	return &verboseReporter{}
}
