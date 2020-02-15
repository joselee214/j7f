package interceptor

import (
	"go.7yes.com/j7f/components/grpc/server"
	"go.7yes.com/j7f/components/log"
	"google.golang.org/grpc"
)

func StreamServerTraceInterceptor(l *log.Logger, cfg *server.Config) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		ts := server.New(stream.Context(), l, stream, cfg)
		return handler(srv, ts)
	}
}

//TODO:: 非流式trace_id拦截器
////func UnaryServerTraceInterceptor(l *log.Logger) grpc.UnaryServerInterceptor{
////}