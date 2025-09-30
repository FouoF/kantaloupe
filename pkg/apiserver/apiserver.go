package apiserver

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	kantaloupeapi "github.com/dynamia-ai/kantaloupe/api/v1"
	"github.com/dynamia-ai/kantaloupe/pkg/apiserver/bff"
	"github.com/dynamia-ai/kantaloupe/pkg/engine"
	"github.com/dynamia-ai/kantaloupe/pkg/service/credential"
	kfservice "github.com/dynamia-ai/kantaloupe/pkg/service/kantaloupeflow"
	"github.com/dynamia-ai/kantaloupe/pkg/service/pod"
	"github.com/dynamia-ai/kantaloupe/pkg/service/quota"
	"github.com/dynamia-ai/kantaloupe/pkg/utils/middleware"
)

type APIServer struct {
	// Debug indicates kairship apiserver mode is debug.
	stopCh           chan struct{}
	Debug            bool
	Server           *http.Server
	GrpcServer       *grpc.Server
	GatewayServerMux *runtime.ServeMux
	CMux             cmux.CMux
	router           *mux.Router
	PrometheusAddr   string
}

func (s *APIServer) PrepareRun(ctx context.Context) error {
	s.router = mux.NewRouter()

	clientManager := engine.NewClientManager()

	if err := s.registerGrpcServices(ctx, clientManager); err != nil {
		return err
	}

	// mux middleware
	s.router.Use(middleware.LogRequestAndResponse)

	s.registerHTTPAPIs()

	s.Server.Handler = s.router
	s.stopCh = make(chan struct{})

	client, err := clientManager.GeteClient(engine.LocalCluster)
	if err != nil {
		return err
	}
	informerFactory := informers.NewSharedInformerFactoryWithOptions(client, time.Hour*1)
	podManager := pod.NewService(informerFactory.Core().V1().Pods().Lister(), clientManager)

	informer := informerFactory.Core().V1().Pods().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    podManager.OnAddPod,
		UpdateFunc: podManager.OnUpdatePod,
		DeleteFunc: podManager.OnDelPod,
	})

	informerFactory.Start(s.stopCh)
	informerFactory.WaitForCacheSync(s.stopCh)

	return s.printkRouters()
}

func (s *APIServer) registerGrpcServices(ctx context.Context, cm engine.ClientManagerInterface) error {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	clientManager := engine.NewClientManager()
	monitoringEngine, err := engine.NewPrometheusClient(s.PrometheusAddr)
	if err != nil {
		return err
	}
	// register cluster service
	kantaloupeapi.RegisterClusterServer(s.GrpcServer, bff.NewClusterHandler(cm, monitoringEngine))
	err = kantaloupeapi.RegisterClusterHandlerFromEndpoint(ctx, s.GatewayServerMux, s.Server.Addr, opts)
	if err != nil {
		return err
	}

	// register node service
	// kantaloupeapi.RegisterNodeServer(s.GrpcServer, bff.NewNodeHandler(clientManager, monitoringEngine))
	// err = kantaloupeapi.RegisterNodeHandlerFromEndpoint(ctx, s.GatewayServerMux, s.Server.Addr, opts)
	// if err != nil {
	// 	return err
	// }

	// register core methods
	kantaloupeapi.RegisterCoreServer(s.GrpcServer, bff.NewCoreHandler(clientManager, monitoringEngine))
	err = kantaloupeapi.RegisterCoreHandlerFromEndpoint(ctx, s.GatewayServerMux, s.Server.Addr, opts)
	if err != nil {
		return err
	}

	// register storage methods
	kantaloupeapi.RegisterStorageServer(s.GrpcServer, bff.NewStorageHandler(clientManager))
	err = kantaloupeapi.RegisterStorageHandlerFromEndpoint(ctx, s.GatewayServerMux, s.Server.Addr, opts)
	if err != nil {
		return err
	}

	// register monitoring methods
	kantaloupeapi.RegisterMonitoringServer(s.GrpcServer, bff.NewMonitoringHandler(clientManager, monitoringEngine))
	err = kantaloupeapi.RegisterMonitoringHandlerFromEndpoint(ctx, s.GatewayServerMux, s.Server.Addr, opts)
	if err != nil {
		return err
	}

	// register credential service
	credentialService := credential.NewService(clientManager)
	kantaloupeapi.RegisterCredentialServer(s.GrpcServer, bff.NewCredentialHandler(credentialService))
	err = kantaloupeapi.RegisterCredentialHandlerFromEndpoint(ctx, s.GatewayServerMux, s.Server.Addr, opts)
	if err != nil {
		return err
	}

	// register quota service
	quotaService := quota.NewService(clientManager, monitoringEngine)
	workloadService := kfservice.NewService(clientManager)
	kantaloupeapi.RegisterQuotaServer(s.GrpcServer, bff.NewQuotaHandler(quotaService, workloadService))
	err = kantaloupeapi.RegisterQuotaHandlerFromEndpoint(ctx, s.GatewayServerMux, s.Server.Addr, opts)
	if err != nil {
		return err
	}

	kantaloupeapi.RegisterKantaloupeflowServer(s.GrpcServer, bff.NewKantaloupeflowHandler(clientManager, monitoringEngine))
	err = kantaloupeapi.RegisterKantaloupeflowHandlerFromEndpoint(ctx, s.GatewayServerMux, s.Server.Addr, opts)
	if err != nil {
		return err
	}

	// register acceleratorcard service
	kantaloupeapi.RegisterAcceleratorCardServer(s.GrpcServer, bff.NewAcceleratorCardHandler(clientManager, monitoringEngine))
	err = kantaloupeapi.RegisterAcceleratorCardHandlerFromEndpoint(ctx, s.GatewayServerMux, s.Server.Addr, opts)
	if err != nil {
		return err
	}

	// TODO: registry handlers for other services
	s.router.PathPrefix("/apis/kantaloupe.dynamia.ai/v1/").Handler(s.GatewayServerMux)
	return err
}

func (s *APIServer) registerHTTPAPIs() {
	healthRouter := s.router.PathPrefix("/").Subrouter()
	healthRouter.HandleFunc("/healthz", livenessProbe)
	healthRouter.HandleFunc("/readyz", readinessProbe)
}

func (s *APIServer) printkRouters() error {
	return s.router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			klog.V(1).InfoS("ROUTE:", "template", pathTemplate)
		}
		pathRegexp, err := route.GetPathRegexp()
		if err == nil {
			klog.V(1).InfoS("Path regexp:", "regexp", pathRegexp)
		}
		queriesTemplates, err := route.GetQueriesTemplates()
		if err == nil {
			klog.V(1).InfoS("Queries templates:", "templates", strings.Join(queriesTemplates, ","))
		}
		queriesRegexps, err := route.GetQueriesRegexp()
		if err == nil {
			klog.V(1).InfoS("Queries regexps:", "regexps", strings.Join(queriesRegexps, ","))
		}
		methods, err := route.GetMethods()
		if err == nil {
			klog.V(1).InfoS("Methods:", "methods", strings.Join(methods, ","))
		}
		return nil
	})
}

func (s *APIServer) Run(ctx context.Context) error {
	s.waitForResourceSync(ctx)

	// Match connections in order:
	// First grpc, then HTTP, and otherwise Go RPC/TCP.
	grpcListener := s.CMux.Match(cmux.HTTP2())
	httpListener := s.CMux.Match(cmux.HTTP1Fast("PATCH"))

	// Use the muxed listeners for your servers.
	go func() {
		err := s.GrpcServer.Serve(grpcListener)
		if err != nil {
			klog.ErrorS(err, "Failed to start grpc server")
		}
	}()

	go func() {
		err := s.Server.Serve(httpListener)
		if err != nil {
			klog.ErrorS(err, "Failed to start http server")
		}
	}()

	// Start serving!
	klog.V(4).InfoS("Serving...")
	return s.CMux.Serve()
}

func (s *APIServer) waitForResourceSync(_ context.Context) {
	klog.V(4).InfoS("Start cache objects")

	// TODO:

	klog.V(4).InfoS("Finished caching objects")
}

func livenessProbe(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func readinessProbe(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
