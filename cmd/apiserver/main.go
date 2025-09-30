package main

import (
	"fmt"
	"os"

	apiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/dynamia-ai/kantaloupe/cmd/apiserver/app"
)

func main() {
	logs.InitLogs()
	log.SetLogger(klog.NewKlogr())
	klog.EnableContextualLogging(true)
	defer logs.FlushLogs()

	ctx := apiserver.SetupSignalContext()

	cmd := app.NewAPIServerCommand(ctx)
	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
