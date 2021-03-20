// Package cmd contains code for the `pixdl` CLI tool.
package cmd

import (
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/jwalton/pixdl/internal/log"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pixdl",
	Short: "Downloads image albums",
	Long: heredoc.Doc(`
		pixdl is used to download pictures and albums from the web.

		Examples:

		  # Download images from an imgur gallery
		  pixdl get https://imgur.com/gallery/88wOh
	`),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.LogError(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pixdl.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "d", false, "Use verbose output")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.LogFatal(err)
		}

		// Search config in home directory with name ".pixdl" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".pixdl")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

}
