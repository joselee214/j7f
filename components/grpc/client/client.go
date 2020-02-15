package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"go.7yes.com/j7f/components/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"io/ioutil"
	"time"
)

type GrpcClient struct {
	Conn         *grpc.ClientConn
	serviceName  string
	addr         string
	timeout      time.Duration
	Log          *log.Logger
	opts         []grpc.DialOption
	Ctx          context.Context
	CtxCancelFuc context.CancelFunc
}

var connMap map[string]*grpc.ClientConn

func init() {
	connMap = make(map[string]*grpc.ClientConn)
}

func NewGrpcClient(serviceName string, timeOut time.Duration, log *log.Logger, options ...grpc.DialOption) *GrpcClient {

	client := &GrpcClient{
		serviceName: serviceName,
		timeout:     timeOut,
		Log:         log,
	}

	for _, option := range options {
		client.opts = append(client.opts, option)
	}

	return client
}

// NewClient创建一个新的gRPC客户端。它拨号到addr指定的服务器。
//如果useTLS为true，则gRPC客户端与服务器建立安全连接。
//如果启用了useTLS，则cert和certKey集启用相互身份验证。
//如果找不到其中一个，NewClient将返回entity.ErrMutualAuthParamsAreNotEnough。
//如果useTLS为false，则忽略cacert，cert和certKey。
func (c *GrpcClient) Endpoint(addr string, useTLS bool, cacert, cert, certKey string) (*grpc.ClientConn, error) {
	// 如果map 里有从map里取值，如果没有重新初始化
	conn, ok := connMap[addr]
	if ok && checkConnStatus(conn) == true{
		return conn, nil
	}

	if !useTLS {
		c.opts = append(c.opts, grpc.WithInsecure())
	} else { // Enable TLS authentication
		var tlsCfg tls.Config
		if cacert != "" {
			b, err := ioutil.ReadFile(cacert)
			if err != nil {
				return nil, err
			}
			cp := x509.NewCertPool()
			if !cp.AppendCertsFromPEM(b) {
				return nil, err
			}
			tlsCfg.RootCAs = cp
		}
		if cert != "" && certKey != "" {
			// Enable mutual authentication
			certificate, err := tls.LoadX509KeyPair(cert, certKey)
			if err != nil {
				return nil, err
			}
			tlsCfg.Certificates = append(tlsCfg.Certificates, certificate)
			c.opts = append(c.opts, grpc.WithTransportCredentials(credentials.NewTLS(&tlsCfg)))
		} else if cert != "" || certKey != "" {
			return nil, fmt.Errorf("cert and certkey are required to authenticate mutually")
		}

		c.opts = append(c.opts, grpc.WithTransportCredentials(credentials.NewTLS(&tlsCfg)))
	}
	// 超时
	var ctx context.Context
	var cancel context.CancelFunc
	if c.timeout != 0 {
		ctx, cancel = context.WithTimeout(context.Background(), c.timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	c.Ctx = ctx
	c.CtxCancelFuc = cancel
	conn, err := grpc.DialContext(ctx, addr, c.opts...)
	if err != nil {
		return nil, err
	}
	if checkConnStatus(conn) == false {
		return nil, err
	}
	c.addr = addr
	c.Conn = conn
	connMap[addr] = conn
	return conn, nil
}

// 普通调用
func (c *GrpcClient) Invoke(ctx context.Context, method string, grpcReply interface{}, request interface{}) (response interface{}, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx = context.WithValue(ctx, "method", method)

	md := &metadata.MD{}
	ctx = metadata.NewOutgoingContext(ctx, *md)

	var header, trailer metadata.MD
	// invoke parameter : example "/hello.Hello/SayHello"
	if err = c.Conn.Invoke(
		ctx, method, request, grpcReply, grpc.Header(&header),
		grpc.Trailer(&trailer),
	); err != nil {
		return nil, err
	}

	return grpcReply, nil
}

//注册流拦截器
// example grpc.WithStreamInterceptor(StreamClientInterceptor(c.Log))
func (c *GrpcClient) RegisterStreamInterceptors(f grpc.StreamClientInterceptor) {
	c.opts = append(c.opts, grpc.WithStreamInterceptor(f))
}

//注册一元RPC拦截器
// example grpc.WithUnaryInterceptor(UnaryClientInterceptor(c.Log))
func (c *GrpcClient) RegisterUnaryInterceptors(f grpc.UnaryClientInterceptor) {
	c.opts = append(c.opts, grpc.WithUnaryInterceptor(f))
}

func checkConnStatus(conn *grpc.ClientConn) bool{
	switch s := conn.GetState(); s {
	case connectivity.TransientFailure:
		return false
	case connectivity.Shutdown:
		return false
	}
	return true
}