package main

import (
	"fmt"
	"os"

	apiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/dynamia-ai/kantaloupe/cmd/controller-manager/app"
)

// Controller-manager main.
func main() {
	logs.InitLogs()
	log.SetLogger(klog.NewKlogr())
	klog.EnableContextualLogging(true)
	defer logs.FlushLogs()

	ctx := apiserver.SetupSignalContext()

	if err := app.NewControllerManagerCommand(ctx).Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
