package interceptor

import (
	"context"
	"go.7yes.com/j7f/components/log"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func UnaryServerErrorInterceptor(l *log.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		res, err := handler(ctx, req)
		if err != nil {
			s, _ := status.FromError(err)
			l.Error(info.FullMethod, zap.Error(err))
			return nil, s.Err()
		}

		return res, err
	}
}

// StreamServerInterceptor returns a new streaming server interceptor for panic recovery.
func StreamServerErrorInterceptor(l *log.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		err = handler(srv, stream)
		if err != nil {
			s, _ := status.FromError(err)
			l.Error(info.FullMethod, zap.Error(err))
			return s.Err()
		}

		return err
	}
}
