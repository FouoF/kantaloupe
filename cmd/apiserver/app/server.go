package app

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"

	"github.com/dynamia-ai/kantaloupe/cmd/apiserver/app/options"
	"github.com/dynamia-ai/kantaloupe/pkg/version"
)

func NewAPIServerCommand(ctx context.Context) *cobra.Command {
	s := options.NewAPIServerRunOptions()
	cmd := &cobra.Command{
		Use:  "kantaloupe-apiserver",
		Long: `The kantaloupe API server.`,
		RunE: func(c *cobra.Command, _ []string) error { //nolint:contextcheck
			klog.V(2).InfoS("Running kantaloupe-apiserver")
			if errs := s.Validate(); len(errs) != 0 {
				return utilerrors.NewAggregate(errs)
			}

			return Run(c.Context(), s)
		},
		SilenceUsage: true,
	}
	cmd.SetContext(ctx)
	fs := cmd.Flags()
	namedFlagSets := s.Flags()
	for _, f := range namedFlagSets.FlagSets {
		fs.AddFlagSet(f)
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of kantaloupe-apiserver",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Println(version.Get())
		},
	}

	cmd.AddCommand(versionCmd)
	return cmd
}

func Run(ctx context.Context, opt *options.Options) error {
	// To help debugging, immediately log version
	klog.Infof("Version: %+v", version.Get())
	klog.InfoS("Golang settings",
		"GOGC", os.Getenv("GOGC"),
		"GOMAXPROCS", os.Getenv("GOMAXPROCS"),
		"GOTRACEBACK", os.Getenv("GOTRACEBACK"))

	apiserver, err := opt.NewAPIServer(ctx)
	if err != nil {
		return err
	}

	if err = apiserver.PrepareRun(ctx); err != nil {
		return err
	}
	return apiserver.Run(ctx)
}
