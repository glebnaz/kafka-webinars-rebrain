package main

import (
	"context"
	"net"
	"net/http"

	"github.com/glebnaz/kafka-webinars-rebrain/internal/pkg/bindata"

	"github.com/glebnaz/go-platform/metrics"
	"github.com/glebnaz/kafka-webinars-rebrain/internal/app/services"
	pb "github.com/glebnaz/kafka-webinars-rebrain/pkg/pb/api/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/kelseyhightower/envconfig"
	"github.com/labstack/echo"
	runner "github.com/oklog/run"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	metricsURI = "/metrics"
	swaggerURI = "/swagger"
)

type app struct {
	//http servers
	debug     *echo.Echo
	muxServer *runtime.ServeMux
	//grpc server
	grpcServer *grpc.Server

	cfg cfgApp
}

//Start app with grpc and http
func (a app) Start(ctx context.Context) error {
	var serverGroup runner.Group

	//add debug
	serverGroup.Add(func() error {
		a.debug.HideBanner = true
		a.debug.HidePort = true
		log.Infof("debug server started at port: %s", a.cfg.DebugPort)
		return a.debug.Start(a.cfg.DebugPort)
	}, func(err error) {
		log.Errorf("Error start debug server: %s", err)
	})

	//grpc
	serverGroup.Add(func() error {
		lis, err := net.Listen("tcp", a.cfg.PortGRPC)
		if err != nil {
			log.Fatalf("bad start grpc port: %s", err)
			return err
		}
		log.Infof("grpc start at: %s", a.cfg.PortGRPC)
		return a.grpcServer.Serve(lis)
	}, func(err error) {
		log.Errorf("bad start grpc port: %s", err)
	})

	//http
	serverGroup.Add(func() error {
		log.Infof("http server started at port: %s", a.cfg.PortHTTP)
		return http.ListenAndServe(a.cfg.PortHTTP, a.muxServer)
	}, func(err error) {
		log.Errorf("bad start http port: %s", err)
	})

	return serverGroup.Run()
}

type cfgApp struct {
	DebugPort string `json:"debug_port" envconfig:"DEBUG_PORT" default:":8084"`
	PortHTTP  string `json:"port_http" envconfig:"PORT_HTTP" default:":8080"`
	PortGRPC  string `json:"port_grpc" envconfig:"PORT_GRPC" default:":8082"`
}

func newApp(ctx context.Context, grpcOPTS ...grpc.ServerOption) app {
	var cfg cfgApp

	err := envconfig.Process("APP", &cfg)
	if err != nil {
		log.Fatalf("cfg app is bad: %s", err)
	}

	srv := services.NewService()

	//debug port
	serverDBG := debugServer()

	//grpc server
	grpcServer := grpcServer(srv, grpcOPTS)

	muxServer := httpServer(ctx, cfg.PortGRPC)

	return app{
		cfg:        cfg,
		debug:      serverDBG,
		grpcServer: grpcServer,
		muxServer:  muxServer,
	}
}

func grpcServer(srv *services.Service, grpcOPTS []grpc.ServerOption) *grpc.Server {
	//grpc server
	grpcServer := grpc.NewServer(grpcOPTS...)
	pb.RegisterPetStoreServer(grpcServer, srv)

	return grpcServer
}

func debugServer() *echo.Echo {
	s := echo.New()

	s.GET(metricsURI, echo.WrapHandler(metrics.Handler()))
	s.GET(swaggerURI, func(c echo.Context) error {
		return c.Blob(http.StatusOK, "application/json", bindata.MustAsset("api/api.swagger.json"))
	})
	return s
}

func httpServer(ctx context.Context, endpoint string) *runtime.ServeMux {
	var jsonPb runtime.JSONPb
	jsonPb.UseProtoNames = true
	mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &jsonPb))
	mustInit(pb.RegisterPetStoreHandlerFromEndpoint(ctx, mux, endpoint, []grpc.DialOption{grpc.WithInsecure()}))
	return mux
}

func mustInit(err error) {
	if err != nil {
		log.Fatalf("Error init: %s", err)
	}
}
