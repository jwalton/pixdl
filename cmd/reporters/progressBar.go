package reporters

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/jwalton/gchalk"
	"github.com/jwalton/pixdl/pkg/download"
	"github.com/jwalton/pixdl/pkg/pixdl"
	"golang.org/x/term"
)

const MAX_WIDTH = 100

var progressBarForeground = gchalk.WithBgCyan().Black
var progressBarBackground = gchalk.WithBgBrightBlack().BrightWhite

type downloadingEntry struct {
	progress *download.Progress
	label    string
}

type progressBarReporter struct {
	mutex  sync.Mutex
	width  int
	height int
	// This is the number of lines we want to erase at the start of the next render.
	linesToErase int
	// Map of entries that are currently downloading, indexed by URL.
	downloading map[string]*downloadingEntry
}

// moveUp moves the cursor up the specified number of lines.
func (*progressBarReporter) moveUp(lines int) {
	fmt.Printf("\u001B[%dA\r", lines)
}

func (p *progressBarReporter) getScreenSize() (width int, height int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Use the last width we had.
		width = p.width
		height = p.height
	}

	if width > MAX_WIDTH {
		width = MAX_WIDTH
	}

	p.width = width
	p.height = height

	return width, height
}

func (p *progressBarReporter) render(message string) {
	width, height := p.getScreenSize()

	// Move the cursor to the top of the area we want to overwrite.
	if p.linesToErase > 1 {
		p.moveUp(p.linesToErase - 1)
	}

	// IF there's a message, print it.
	if message != "" {
		fmt.Print("\r" + strings.Repeat(" ", width))
		fmt.Println("\r" + message)
	}

	items := make([]*downloadingEntry, 0, len(p.downloading))
	for _, entry := range p.downloading {
		items = append(items, entry)
	}

	// Sort items so most complete ones are at the top.
	sort.Slice(items, func(i int, j int) bool {
		return items[i].progress.PercentComplete > items[j].progress.PercentComplete
	})

	// Don't print more lines than will fit on the screen.
	if len(items) > (height - 1) {
		items = items[0 : height-1]
	}

	for index, item := range items {
		p.renderItem(item, width)
		if index != len(items)-1 {
			fmt.Println()
		}
	}

	p.linesToErase = len(items)

}

func (p *progressBarReporter) lineToWidth(message string, width int) string {
	if len(message) > width {
		message = message[:width]
	}

	return message + strings.Repeat(" ", width-len(message))
}

func (p *progressBarReporter) renderItem(entry *downloadingEntry, width int) {
	label := entry.label
	complete := fmt.Sprintf("%.2f%%", entry.progress.PercentComplete)

	strWidth := len(label) + len(complete) + 2 // +1 for space, +1 for left margin.
	maxWidth := width - 1
	if strWidth > maxWidth {
		over := strWidth - maxWidth
		label = label[0:len(label)-(over+1)] + "… "
	}

	strWidth = len(label) + len(complete)
	marginLeft := 1
	marginRight := maxWidth - strWidth - marginLeft
	if marginRight < 0 {
		marginRight = 0
	}

	line := strings.Repeat(" ", marginLeft) + label + " " + complete + strings.Repeat(" ", marginRight)

	completeWidth := int(float64(width) * (entry.progress.PercentComplete / 100.0))
	if completeWidth > len(line) {
		// Paranoid...
		completeWidth = len(line)
	}
	if completeWidth < 0 {
		// This will happen if we don't know the length of the file.
		completeWidth = 0
	}

	// The part that will be colored in the "done" color
	lineLeft := line[:completeWidth]
	// The part that will be colored in the "not done" coloe
	lineRight := line[completeWidth:]

	fmt.Printf("\r%s%s",
		progressBarForeground(lineLeft),
		progressBarBackground(lineRight),
	)

}

func (p *progressBarReporter) getItemLabel(image *pixdl.ImageMetadata) string {
	return fmt.Sprintf("%s (%d/%d)", path.Base(image.Filename), image.Index+1, image.Album.TotalImageCount)
}

func (p *progressBarReporter) AlbumFetch(url string) {
	message := fmt.Sprintf("%s Fetching album from %s", gchalk.BrightBlue("Info    :"), url)
	p.render(message)
}

func (p *progressBarReporter) AlbumStart(album *pixdl.AlbumMetadata) {
	// Ignore
}

func (p *progressBarReporter) AlbumEnd(album *pixdl.AlbumMetadata, err error) {
	// Ignore
}

func (p *progressBarReporter) ImageSkip(image *pixdl.ImageMetadata, err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	var message string
	if err == nil {
		message = fmt.Sprintf("%s %s", gchalk.BrightBlue("Skipped :"), p.getItemLabel(image))
	} else {
		message = fmt.Sprintf("%s %s: %v", gchalk.BrightBlue("Skipped :"), p.getItemLabel(image), err)
	}
	p.render(message)
}

func (p *progressBarReporter) ImageStart(image *pixdl.ImageMetadata) {
	// Ignore
}

func (p *progressBarReporter) ImageProgress(
	image *pixdl.ImageMetadata,
	progress *download.Progress,
) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if _, exists := p.downloading[image.URL]; exists {
		p.downloading[image.URL].progress = progress
	} else {
		p.downloading[image.URL] = &downloadingEntry{
			label:    p.getItemLabel(image),
			progress: progress,
		}
	}

	// If there's a warning, print it.
	message := ""
	if progress.Warning != "" {
		message = fmt.Sprintf("%s %s: %s",
			gchalk.BrightYellow("Warning :"),
			p.getItemLabel(image),
			progress.Warning,
		)
	}

	p.render(message)
}

func (p *progressBarReporter) ImageEnd(image *pixdl.ImageMetadata, err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// FIXME: Show errors and completed items.
	delete(p.downloading, image.URL)
	var message string
	if err == nil {
		message = gchalk.BrightGreen("Complete: ") + p.getItemLabel(image)
	} else {
		message = fmt.Sprintf("%s %s: %v", gchalk.BrightRed("Error   :"), p.getItemLabel(image), err)
	}
	p.render(message)
}

// NewProgressBarReporter returns a new ProgressReporter which shows a pretty progress bar.
func NewProgressBarReporter() (pixdl.ProgressReporter, error) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))

	if err != nil {
		return nil, err
	}

	return &progressBarReporter{
		width:        width,
		height:       height,
		linesToErase: 0,
		downloading:  map[string]*downloadingEntry{},
	}, nil
}
