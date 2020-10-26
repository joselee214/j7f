package server

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/joselee214/j7f/components/log"
	"github.com/joselee214/j7f/components/service_register"
	"go.uber.org/zap"
	"net"
	"net/http"
	//"os"
	//"os/signal"
	"time"
)

type HttpServer struct {
	addr *net.TCPAddr

	lis *net.TCPListener

	r *gin.Engine

	s *http.Server

	l *log.Logger

	cb []HttpCallback

	Config map[string]interface{}
}

type HttpCallback func(r *HttpServer) error

func NewHttpServer(addr *net.TCPAddr, log *log.Logger, env string) (*HttpServer, error) {
	var err error
	g := &HttpServer{}

	g.addr = addr

	g.lis, err = net.ListenTCP("tcp", g.addr)
	if err != nil {
		return nil, err
	}

	gin.DefaultWriter = log
	gin.DefaultErrorWriter = log

	if env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	g.r = gin.New()
	g.l = log

	return g, nil
}

func (g *HttpServer) RegisterStreamInterceptors(fs ...interface{}) {
	for _, f := range fs {
		if f, ok := f.(gin.HandlerFunc); ok {
			g.r.Use(f)
		}
	}
}

//当成RegisterMiddleware来用
func (g *HttpServer) RegisterUnaryInterceptors(fs ...interface{}) {
	for _, f := range fs {
		if f, ok := f.(gin.HandlerFunc); ok {
			g.r.Use(f)
		}
	}
}

func (g *HttpServer) RegisterCb(cbs ...interface{}) {
	for _, cb := range cbs {
		if cb, ok := cb.(HttpCallback); ok {
			g.cb = append(g.cb, cb)
		}
	}
}

func (g *HttpServer) NewServ() error {
	for _, f := range g.cb {
		err := f(g)
		if err != nil {
			return err
		}
	}

	PingInit(g.r)

	g.s = &http.Server{
		Addr:    g.addr.String(),
		Handler: g.r,
	}
	return nil
}

func (g *HttpServer) GetEngine() *gin.Engine {
	return g.r
}

func (g *HttpServer) StartServ() error {
	return g.s.Serve(g.lis)
}

func (g *HttpServer) Stop() {
	g.shutdown(g.s)
}

func (g *HttpServer) GracefulStop() {
	g.shutdown(g.s)
}

func (g *HttpServer) GetServicesInfo() map[string]service_register.ServerInfo {
	var ServerInfo = make(map[string]service_register.ServerInfo)
	routes := g.r.Routes()

	for _, route := range routes {
		if _,ok := ServerInfo[route.Path];ok==false{
			ServerInfo[route.Path] = service_register.ServerInfo{
				Methods:  make([]service_register.MethodInfo,0),	//	创建空...
			}
		}
		methods := ServerInfo[route.Path].Methods
		methods = append(methods,service_register.MethodInfo{
			Name:           route.Method,
			IsClientStream: false,
			IsServerStream: false,
		})
		ServerInfo[route.Path] = service_register.ServerInfo{
			Methods:  methods,
		}
	}

	return ServerInfo
}

func (g *HttpServer) GetAddress() *net.TCPAddr {
	return g.addr
}

func (g *HttpServer) GetListener() *net.TCPListener {
	return g.lis
}

func (g *HttpServer) shutdown(srv *http.Server) {
	//quit := make(chan os.Signal)
	//signal.Notify(quit, os.Interrupt)
	//<-quit

	g.l.Logger.Debug("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		g.l.Logger.Panic("server", zap.String("shutdown", err.Error()))
	}

	g.l.Logger.Debug("Server exiting")
}
