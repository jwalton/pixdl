package cmd

import (
	"fmt"
	"os"

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
		if err != nil {
			log.LogFatal(err)
		}

		toFolder, err := cmd.Flags().GetString("out")
		if err != nil {
			log.LogFatal(err)
		}

		maxPages, err := cmd.Flags().GetInt("max-pages")
		if err != nil {
			log.LogFatal(err)
		}

		reporter := getReporter(verbose)

		if toFolder == "" {
			toFolder, err = os.Getwd()
			if err != nil {
				log.LogFatalf("Unable to determine working directory: %v", err)
			}
		}

		// TODO: Add option for this.
		maxConcurrency := uint(4)

		options := pixdl.DownloadOptions{
			ToFolder: toFolder,
			MaxPages: maxPages,
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
	getCmd.Flags().Int("max-pages", 0, "Maximum number of pages to download from album (0 for all)")
}
