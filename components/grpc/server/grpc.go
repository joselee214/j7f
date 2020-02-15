package server

import (
	"context"
	"github.com/joselee214/j7f/components/errors"
	"github.com/joselee214/j7f/components/log"
	"github.com/joselee214/j7f/proto/common"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"time"
)

const DEFAULT_RATE_LIMIT = 1000

const UTO_CONTEXT_LOG_KEY = "UTO_CONTEXT_LOG_KEY"

type GrpcStream struct {
	ctx            context.Context
	parentCtx      context.Context
	cancel         context.CancelFunc
	l              *log.Logger
	g              grpc.ServerStream
	r              ratelimit.Limiter
	processingDone chan int
	cfg            *Config
}

type Config struct {
	PerRequest        int
	ProcessingTimeout int
}

type RequestTrace interface {
	GetHeader() *common.CommonHeader
}

type ResponseStatus interface {
	GetStatus() *common.BusinessStatus
}

func New(ctx context.Context, l *log.Logger, g grpc.ServerStream, cfg *Config) *GrpcStream {
	if cfg.PerRequest == 0 {
		cfg.PerRequest = DEFAULT_RATE_LIMIT
	}

	s := &GrpcStream{
		parentCtx: ctx,
		ctx:       ctx,
		l:         l,
		g:         g,

		cfg: cfg,
	}

	s.r = ratelimit.New(s.cfg.PerRequest)

	return s
}

func (s *GrpcStream) SetHeader(m metadata.MD) error {
	return s.g.SetHeader(m)
}

func (s *GrpcStream) SendHeader(m metadata.MD) error {
	return s.g.SendHeader(m)
}

func (s *GrpcStream) SetTrailer(m metadata.MD) {
	s.g.SetTrailer(m)
}

func (s *GrpcStream) Context() context.Context {
	return s.ctx
}

func (s *GrpcStream) SendMsg(m interface{}) error {
	s.processingDone <- 1
	traceId := ""
	if r, ok := m.(ResponseStatus); ok {
		md, ok := metadata.FromIncomingContext(s.ctx)
		if ok {
			if traceIds, ok := md["trace_id"]; ok {
				status := r.GetStatus()
				status.TraceId = traceIds[0]
				traceId = traceIds[0]
			}
		}
	}
	s.ctx = s.parentCtx
	s.l.Debug(traceId, zap.Any("response", m))
	return s.g.SendMsg(m)
}

func (s *GrpcStream) RecvMsg(m interface{}) error {
	s.r.Take()

	if err := s.g.RecvMsg(m); err != nil {
		return err
	}

	s.ctx, s.cancel = context.WithTimeout(s.parentCtx, time.Duration(s.cfg.ProcessingTimeout)*time.Second)

	traceId := ""
	if r, ok := m.(RequestTrace); ok {
		header := r.GetHeader()
		traceId = header.GetTraceId()

		md, ok := metadata.FromIncomingContext(s.ctx)
		if ok {
			md["trace_id"] = []string{traceId}
			s.ctx = metadata.NewIncomingContext(s.ctx, md)
		}

	}
	l := s.l.Trace(s.ctx)
	s.ctx = context.WithValue(s.ctx, UTO_CONTEXT_LOG_KEY, l)

	s.processingDone = make(chan int, 1)

	go func() {
		select {
		case <-s.ctx.Done():
			err := errors.NewFromCode(errors.CommonError_PROCESSING_TIMEOUT)
			t := &common.TimeoutResponse{
				Status: errors.GetResHeader(err),
			}
			s.cancel()
			sendErr := s.SendMsg(t)
			if sendErr != nil {
				s.l.Error(traceId, zap.String("send", sendErr.Error()))
			}
		case <-s.processingDone:
			s.cancel()
			return
		}
	}()

	s.l.Debug(traceId, zap.Any("request", m))
	return nil
}
