package server

import (
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"go.7yes.com/j7f/components/service_register"
	"google.golang.org/grpc"
	"net"
)

type GrpcServer struct {
	addr *net.TCPAddr

	lis *net.TCPListener

	//grpc
	s *grpc.Server

	opts []grpc.ServerOption

	cb []GrpcCallback

	Config map[string]interface{}
}

type GrpcCallback func(s *GrpcServer) error

func NewGrpcServer(addr *net.TCPAddr, opts ...grpc.ServerOption) (*GrpcServer, error) {
	var err error
	g := &GrpcServer{}

	g.addr = addr

	g.lis, err = net.ListenTCP("tcp", g.addr)
	if err != nil {
		return nil, err
	}

	g.opts = opts

	return g, nil
}

//注册流拦截器
func (g *GrpcServer) RegisterStreamInterceptors(fs ...interface{}) {
	var tmp []grpc.StreamServerInterceptor
	for _, f := range fs {
		if f, ok := f.(grpc.StreamServerInterceptor); ok {
			tmp = append(tmp, f)
		}
	}
	g.opts = append(g.opts, grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(tmp...)))

}

//注册一元RPC拦截器
func (g *GrpcServer) RegisterUnaryInterceptors(fs ...interface{}) {
	var tmp []grpc.UnaryServerInterceptor
	for _, f := range fs {
		if f, ok := f.(grpc.UnaryServerInterceptor); ok {
			tmp = append(tmp, f)
		}
	}
	g.opts = append(g.opts, grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(tmp...)))
}

func (g *GrpcServer) RegisterCb(cbs ...interface{}) {
	for _, cb := range cbs {
		if cb, ok := cb.(GrpcCallback); ok {
			g.cb = append(g.cb, cb)
		}
	}
}

func (g *GrpcServer) NewServ() error {
	var err error

	g.s = grpc.NewServer(g.opts...)

	for _, f := range g.cb {
		err = f(g)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *GrpcServer) GetEngine() *grpc.Server {
	return g.s
}

func (g *GrpcServer) StartServ() error {
	return g.s.Serve(g.lis)
}

//停止服务
func (g *GrpcServer) Stop() {
	g.s.Stop()
}

//GracefulStop stops the gRPC server gracefully
func (g *GrpcServer) GracefulStop() {
	g.s.Stop()
}

func (g *GrpcServer) GetServicesInfo() map[string]service_register.ServerInfo {
	var infos = make(map[string]service_register.ServerInfo)
	serviceInfo := g.s.GetServiceInfo()
	for k, v := range serviceInfo {
		info := service_register.ServerInfo{
			Metadata: v.Metadata,
		}
		for _, method := range v.Methods  {
			tmp := service_register.MethodInfo{
				Name: method.Name,
				IsClientStream: method.IsClientStream,
				IsServerStream: method.IsServerStream,
			}
			info.Methods = append(info.Methods, tmp)
		}
		infos[k] = info
	}
	return infos
}

//获取监听地址
func (g *GrpcServer) GetAddress() *net.TCPAddr {
	return g.addr
}

//获取listener
func (g *GrpcServer) GetListener() *net.TCPListener {
	return g.lis
}
