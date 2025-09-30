package options

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/soheilhy/cmux"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"

	"github.com/dynamia-ai/kantaloupe/pkg/apiserver"
	"github.com/dynamia-ai/kantaloupe/pkg/profileflag"
)

type ServerRunOptions struct {
	// server bind address
	BindAddress string

	// insecure port number
	InsecurePort int

	// secure port number
	SecurePort int

	// tls cert file
	TLSCertFile string

	// tls private key file
	TLSPrivateKey string

	// insecure grpc port number
	InsecureGRPCPort int
}

type PrometheusOptions struct {
	TimeOut time.Duration
	// FIXME: Each member cluster should have a Prometheus address.
	Addr string // Prometheus address
}

func NewPrometheusOptions() *PrometheusOptions {
	return &PrometheusOptions{
		Addr: "http://prometheus-kube-prometheus-prometheus.monitoring.svc.cluster.local:9090",
	}
}

func (o *PrometheusOptions) AddFlags(fs *pflag.FlagSet, c *PrometheusOptions) {
	fs.StringVar(&o.Addr, "prometheus-addr", c.Addr, "Prometheus address")
}

func NewServerRunOptions() *ServerRunOptions {
	// create default server run options
	s := ServerRunOptions{
		BindAddress:      "0.0.0.0",
		InsecurePort:     8000,
		InsecureGRPCPort: 8001,
		SecurePort:       0,
		TLSCertFile:      "",
		TLSPrivateKey:    "",
	}

	return &s
}

type Options struct {
	ConfigFile        string
	ServerRunOptions  *ServerRunOptions
	PrometheusOptions *PrometheusOptions
	// Debug indicates kantaloupe apiserver mode is debug.
	Debug bool

	ProfileOpts profileflag.Options
}

func NewAPIServerRunOptions() *Options {
	return &Options{
		ServerRunOptions:  NewServerRunOptions(),
		PrometheusOptions: NewPrometheusOptions(),
	}
}

func (s *ServerRunOptions) AddFlags(fs *pflag.FlagSet, c *ServerRunOptions) {
	fs.StringVar(&s.BindAddress, "bind-address", c.BindAddress, "server bind address")
	fs.IntVar(&s.InsecurePort, "insecure-port", c.InsecurePort, "insecure port number")
	fs.IntVar(&s.SecurePort, "secure-port", s.SecurePort, "secure port number")
	fs.IntVar(&s.InsecureGRPCPort, "insecure-grpc-port", s.InsecureGRPCPort, "insecure grpc port number")
	fs.StringVar(&s.TLSCertFile, "tls-cert-file", c.TLSCertFile, "tls cert file")
	fs.StringVar(&s.TLSPrivateKey, "tls-private-key", c.TLSPrivateKey, "tls private key")
}

func (o *Options) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}
	fs := fss.FlagSet("generic")
	fs.BoolVar(&o.Debug, "debug", false, "apiserver server mode")
	o.ServerRunOptions.AddFlags(fs, o.ServerRunOptions)
	o.PrometheusOptions.AddFlags(fs, o.PrometheusOptions)
	o.ProfileOpts.AddFlags(fss.FlagSet("profile"))
	fs = fss.FlagSet("klog")
	local := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(local)
	local.VisitAll(func(fl *flag.Flag) {
		fl.Name = strings.Replace(fl.Name, "_", "-", -1)
		fs.AddGoFlag(fl)
	})

	return fss
}

func (o *Options) NewAPIServer(_ context.Context) (*apiserver.APIServer, error) {
	apiServer := &apiserver.APIServer{
		Debug:          o.Debug,
		PrometheusAddr: o.PrometheusOptions.Addr,
	}

	// Create the main listener.
	address := fmt.Sprintf("%s:%d", o.ServerRunOptions.BindAddress, o.ServerRunOptions.InsecurePort)
	l, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	// Create a cmux.
	apiServer.CMux = cmux.New(l)

	// Create your protocol servers.
	apiServer.Server = &http.Server{
		Addr:              address,
		ReadHeaderTimeout: 60 * time.Second,
	}

	apiServer.GrpcServer = grpc.NewServer(
		grpc.StreamInterceptor(grpcmiddleware.ChainStreamServer(
			grpcrecovery.StreamServerInterceptor(),
		)),
		grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(
			grpcrecovery.UnaryServerInterceptor(),
		)))

	marshaler := &runtime.JSONPb{}
	marshaler.UseProtoNames = false
	marshaler.EmitUnpopulated = true
	apiServer.GatewayServerMux = runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, marshaler),
	)

	return apiServer, nil
}
