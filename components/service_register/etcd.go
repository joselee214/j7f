package service_register

import (
	"context"
	"crypto/tls"
	"errors"
	"go.etcd.io/etcd/clientv3"
	"time"
)

type Config struct {
	Endpoints []string `json:"endpoints"`

	// AutoSyncInterval is the interval to update endpoints with its latest members.
	// 0 disables auto-sync. By default auto-sync is disabled.
	AutoSyncInterval int `json:"auto-sync-interval"`

	// DialTimeout is the timeout for failing to establish a connection.
	DialTimeout int `json:"dial-timeout"`

	// DialKeepAliveTime is the time after which client pings the server to see if
	// transport is alive.
	DialKeepAliveTime int `json:"dial-keep-alive-time"`

	// DialKeepAliveTimeout is the time that the client waits for a response for the
	// keep-alive probe. If the response is not received in this time, the connection is closed.
	DialKeepAliveTimeout int `json:"dial-keep-alive-timeout"`

	// TLS holds the client secure credentials, if any.
	TLS *tls.Config

	// Username is a user name for authentication.
	Username string `json:"username"`

	// Password is a password for authentication.
	Password string `json:"password"`
}

type EtcdCli struct {
	ctx     context.Context
	leaseID clientv3.LeaseID
	leaser  clientv3.Lease
	watcher clientv3.Watcher
	hbch    <-chan *clientv3.LeaseKeepAliveResponse
	c       *clientv3.Client
}

func NewEtcd(c *Config) (*EtcdCli, error) {
	cfg := clientv3.Config{
		Endpoints:            c.Endpoints,
		AutoSyncInterval:     time.Duration(c.AutoSyncInterval) * time.Millisecond,
		DialTimeout:          time.Duration(c.DialTimeout) * time.Millisecond,
		DialKeepAliveTime:    time.Duration(c.DialKeepAliveTime) * time.Millisecond,
		DialKeepAliveTimeout: time.Duration(c.DialKeepAliveTimeout) * time.Millisecond,
		TLS:                  c.TLS,
		Username:             c.Username,
		Password:             c.Password,
	}
	cli, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}

	return &EtcdCli{c: cli}, nil
}

func (e *EtcdCli) Register(ctx context.Context, s *Service) error {
	var err error
	if e.leaser != nil {
		err = e.leaser.Close()
		if err != nil {
			return err
		}
	}
	e.leaser = clientv3.NewLease(e.c)

	if e.watcher != nil {
		err = e.watcher.Close()
		if err != nil {
			return err
		}
	}
	e.watcher = clientv3.NewWatcher(e.c)

	if s.TTL == nil {
		s.TTL = NewTTLOption(time.Second*3, time.Second*10)
	}

	e.ctx = ctx

	grantResp, err := e.leaser.Grant(e.ctx, int64(s.TTL.ttl.Seconds()))
	if err != nil {
		return err
	}
	e.leaseID = grantResp.ID

	_, err = e.c.Put(
		e.ctx,
		s.Key,
		s.Value,
		clientv3.WithLease(e.leaseID),
	)
	if err != nil {
		return err
	}

	// this will keep the key alive 'forever' or until we revoke it or
	// the context is canceled
	e.hbch, err = e.leaser.KeepAlive(e.ctx, e.leaseID)
	if err != nil {
		return err
	}
	return nil
}

func (e *EtcdCli) DeRegister(ctx context.Context, s *Service) error {
	defer e.close()

	if s.Key == "" {
		return errors.New("no key provided")
	}
	if _, err := e.c.Delete(e.ctx, s.Key, clientv3.WithIgnoreLease()); err != nil {
		return err
	}

	return nil
}

func (e *EtcdCli) Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan {
	return e.c.Watch(ctx, key, opts...)
}

func (e *EtcdCli) close() {
	if e.leaser != nil {
		_ = e.leaser.Close()
	}
	if e.watcher != nil {
		_ = e.watcher.Close()
	}
}
