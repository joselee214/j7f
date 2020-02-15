package client

import (
	"context"
	"github.com/rs/xid"
	"github.com/joselee214/j7f/components/grpc/server"
	"github.com/joselee214/j7f/components/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"path"
	"time"
)

// clientStream implements a client side Stream.
type clientStream struct {
	ctx    context.Context
	method string
	g      grpc.ClientStream
	l      *log.Logger
}

func NewStream(ctx context.Context, method string, l *log.Logger, g grpc.ClientStream) *clientStream {

	c := &clientStream{
		ctx:    ctx,
		l:      l,
		g:      g,
		method: method,
	}
	return c
}

func (cs *clientStream) Header() (metadata.MD, error) {
	return cs.g.Header()
}

func (cs *clientStream) Trailer() metadata.MD {
	return cs.g.Trailer()
}

func (cs *clientStream) CloseSend() error {
	return cs.g.CloseSend()
}

func (cs *clientStream) Context() context.Context {
	return cs.g.Context()
}

func (cs *clientStream) SendMsg(m interface{}) (err error) {
	traceId := GenerageTranceId()

	if r, ok := m.(server.ResponseStatus); ok {
		md, ok := metadata.FromIncomingContext(cs.ctx)
		if ok {
			if traceIds, ok := md["trace_id"]; ok {
				status := r.GetStatus()
				status.TraceId = traceIds[0]
				traceId = traceIds[0]
			}
		}
	}
	cs.l.Debug(traceId, zap.Any("response", m))
	return cs.g.SendMsg(m)
}

func (cs *clientStream) RecvMsg(m interface{}) (err error) {

	if err := cs.g.RecvMsg(m); err != nil {
		return err
	}

	traceId := GenerageTranceId()
	if r, ok := m.(server.RequestTrace); ok {
		header := r.GetHeader()
		traceId = header.GetTraceId()

		md, ok := metadata.FromIncomingContext(cs.ctx)
		if ok {
			md["trace_id"] = []string{traceId}
			cs.ctx = metadata.NewIncomingContext(cs.ctx, md)
		}
	}
	cs.l.Debug(traceId, zap.Any("request", m))
	return cs.g.RecvMsg(m)
}

// grpc client 流拦截器
func StreamClientInterceptor(l *log.Logger) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		fields := newClientLoggerFields(ctx, method)
		startTime := time.Now()
		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		logFinalClientLine(l.Logger.With(fields...), startTime, err, "finished client streaming call")

		myClientStream := NewStream(ctx, method, l, clientStream)
		return myClientStream, err
	}
}

// grpc client 普通拦截器
func UnaryClientInterceptor(l *log.Logger) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		fields := newClientLoggerFields(ctx, method)
		startTime := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		logFinalClientLine(l.Logger.With(fields...), startTime, err, "finished client streaming call")
		return err
	}
}

// 打印log
func logFinalClientLine(logger *zap.Logger, startTime time.Time, err error, msg string) {
	code := status.Code(err)
	level := ClientCodeToLevel(code)
	logger.Check(level, msg).Write(
		zap.Error(err),
		zap.String("grpc.code", code.String()),
		DurationToTimeMillisField(time.Now().Sub(startTime)),
	)
}

func newClientLoggerFields(ctx context.Context, fullMethodString string) []zapcore.Field {
	service := path.Dir(fullMethodString)[1:]
	method := path.Base(fullMethodString)

	traceId := GenerageTranceId()
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if traceIds, ok := md["trace_id"]; ok {
			traceId = traceIds[0]
		}
	}
	return []zapcore.Field{
		zap.String("grpc.service", service),
		zap.String("grpc.method", method),
		zap.String("grpc.trace_id", traceId),
	}
}

// client端，转换gRPC codes 成 log levels
func ClientCodeToLevel(code codes.Code) zapcore.Level {
	switch code {
	case codes.OK:
		return zap.DebugLevel
	case codes.Canceled:
		return zap.DebugLevel
	case codes.Unknown:
		return zap.InfoLevel
	case codes.InvalidArgument:
		return zap.DebugLevel
	case codes.DeadlineExceeded:
		return zap.InfoLevel
	case codes.NotFound:
		return zap.DebugLevel
	case codes.AlreadyExists:
		return zap.DebugLevel
	case codes.PermissionDenied:
		return zap.InfoLevel
	case codes.Unauthenticated:
		return zap.InfoLevel // unauthenticated requests can happen
	case codes.ResourceExhausted:
		return zap.DebugLevel
	case codes.FailedPrecondition:
		return zap.DebugLevel
	case codes.Aborted:
		return zap.DebugLevel
	case codes.OutOfRange:
		return zap.DebugLevel
	case codes.Unimplemented:
		return zap.WarnLevel
	case codes.Internal:
		return zap.WarnLevel
	case codes.Unavailable:
		return zap.WarnLevel
	case codes.DataLoss:
		return zap.WarnLevel
	default:
		return zap.InfoLevel
	}
}

// DurationToTimeMillisField converts the duration to milliseconds and uses the key `grpc.time_ms`.
func DurationToTimeMillisField(duration time.Duration) zapcore.Field {
	return zap.Float32("grpc.time_ms", durationToMilliseconds(duration))
}

func durationToMilliseconds(duration time.Duration) float32 {
	return float32(duration.Nanoseconds()/1000) / 1000
}

func GenerageTranceId() string {
	guid := xid.New()
	return guid.String()
}
