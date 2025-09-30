package metrics

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"testing"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"k8s.io/apimachinery/pkg/version"
)

func Test_versionGet(t *testing.T) {
	tests := []struct {
		name string
		want version.Info
	}{
		{
			name: "get version",
			want: version.Info{
				GitVersion:   "v0.0.0-master",
				GitCommit:    "unknown",
				GitTreeState: "unknown",
				BuildDate:    "unknown",
				GoVersion:    runtime.Version(),
				Compiler:     runtime.Compiler,
				Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionGet(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("versionGet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultMetrics_Install(t *testing.T) {
	type args struct {
		router *mux.Router
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test metrics install",
			args: args{
				router: &mux.Router{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := DefaultMetrics{}
			m.Install(tt.args.router)
			if tt.args.router.Get("/kapis/metrics") != nil {
				t.Error("router name is not nil")
			}
		})
	}
}

func TestHandler(t *testing.T) {
	tests := []struct {
		name string
		want http.Handler
	}{
		{
			name: "test metric handler",
			want: promhttp.InstrumentMetricHandler(prometheus.NewRegistry(), promhttp.HandlerFor(defaultRegistry, promhttp.HandlerOpts{})),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Handler(); reflect.DeepEqual(got, tt.want) {
				t.Errorf("Handler() = %v, want %v", got, tt.want)
			}
		})
	}
}
