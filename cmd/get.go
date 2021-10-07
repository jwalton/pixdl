package cmd

import (
	"fmt"
	"os"
	"regexp"

	"github.com/MakeNowJust/heredoc"
	"github.com/jwalton/pixdl/internal/log"
	"github.com/jwalton/pixdl/pkg/pixdl"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get [url]",
	Short: "Download an album",
	Example: heredoc.Doc(`
		# Download images from an imgur gallery
		pixdl get https://imgur.com/gallery/88wOh

		# Download files from gofile.io
		pixdl get --param gofile.token=xxx https://gofile.io/d/abdef
	`),
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("requires a URL to download from")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]

		verbose, err := cmd.Flags().GetBool("verbose")
		log.PixdlDieOnError(err)

		toFolder, err := cmd.Flags().GetString("out")
		log.PixdlDieOnError(err)

		filenameTemplate, err := cmd.Flags().GetString("template")
		log.PixdlDieOnError(err)

		maxImages, err := cmd.Flags().GetInt("max")
		log.PixdlDieOnError(err)

		maxPages, err := cmd.Flags().GetInt("max-pages")
		log.PixdlDieOnError(err)

		filterSubAlbum, err := cmd.Flags().GetString("subalbum")
		log.PixdlDieOnError(err)

		params, err := cmd.Flags().GetStringArray("param")
		log.PixdlDieOnError(err)

		reporter := getReporter(verbose)

		if toFolder == "" {
			toFolder, err = os.Getwd()
			if err != nil {
				log.PixdlFatalf("Unable to determine working directory: %v", err)
			}
		}

		// TODO: Add option for this.
		maxConcurrency := uint(4)

		options := pixdl.DownloadOptions{
			ToFolder:         toFolder,
			FilenameTemplate: filenameTemplate,
			MaxPages:         maxPages,
			MaxImages:        maxImages,
			FilterSubAlbum:   filterSubAlbum,
			Params:           parseParams(params),
		}

		downloader := pixdl.NewConcurrnetDownloader(pixdl.SetMaxConcurrency(maxConcurrency))
		downloader.DownloadAlbum(url, options, reporter)
		downloader.Wait()
		downloader.Close()

		fmt.Println("All done")
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().StringP("out", "o", "", "Output directory to put files in")
	getCmd.Flags().StringP("template", "t", "", `Template to use to generate filenames.
e.g. "{{.Album.Name}}/{{.Image.SubAlbum}}/{{.Filename}}"`)
	getCmd.Flags().IntP("max", "n", 0, "Maximum number of images to download from album (0 for all)")
	getCmd.Flags().Int("max-pages", 0, "Maximum number of pages to download from album (0 for all)")
	getCmd.Flags().String("subalbum", "", "Only download images from the specified sub-album or post")
	getCmd.Flags().StringArrayP("param", "p", []string{}, "Specify a parameter to pass to providers")
}

var paramRegex = regexp.MustCompile(`^([a-zA-Z\.-_]*)=(.*)$`)

func parseParams(params []string) map[string]string {
	result := map[string]string{}

	for _, param := range params {
		match := paramRegex.FindStringSubmatch(param)
		if match == nil {
			result[param] = ""
		} else {
			key := match[1]
			value := match[2]
			result[key] = value
		}
	}

	return result
}
