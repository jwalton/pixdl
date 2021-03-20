package cmd

import (
	"github.com/jwalton/go-supportscolor"
	"github.com/jwalton/pixdl/cmd/reporters"
	"github.com/jwalton/pixdl/pkg/pixdl"
)

func getReporter(verbose bool) pixdl.ProgressReporter {
	var result pixdl.ProgressReporter

	if verbose || !supportscolor.Stdout().SupportsColor {
		result = reporters.NewVerboseReporter()
	} else {
		var err error
		result, err = reporters.NewProgressBarReporter()
		if err != nil {
			result = reporters.NewVerboseReporter()
		}
	}

	return result
}
