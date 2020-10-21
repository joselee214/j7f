package etcd_listen

import (
	"context"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
)

var (
	timeOut = time.Duration(3) * time.Second // 超时
)

// Listener 对外通知
type Listener interface {
	Set(string,[]byte, []byte)
	SetOk(string)
	Create(string,[]byte, []byte)
	Modify(string,[]byte, []byte)
	Delete(string,[]byte)
	Empty(string)
}

// EtcdWatcher ETCD key监视器
type EtcdWatcher struct {
	cli          *clientv3.Client // etcd client
	wg           sync.WaitGroup
	listener     Listener
	mu           sync.Mutex
	closeHandler map[string]func()
}

// NewEtcdWatcher 构造
func NewEtcdWatcher(servers []string) (*EtcdWatcher, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   servers,
		DialTimeout: timeOut,
	})
	if err != nil {
		return nil, err
	}

	ew := &EtcdWatcher{
		cli:          cli,
		closeHandler: make(map[string]func()),
	}

	return ew, nil
}

// AddWatch 添加监视
func (mgr *EtcdWatcher) AddWatch(key string, prefix bool, listener Listener) bool {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	if _, ok := mgr.closeHandler[key]; ok {
		return false
	}
	ctx, cancel := context.WithCancel(context.Background())
	mgr.closeHandler[key] = cancel

	mgr.wg.Add(1)
	go mgr.watch(ctx, key, prefix, listener)

	return true
}

// RemoveWatch 删除监视
func (mgr *EtcdWatcher) RemoveWatch(key string) bool {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	cancel, ok := mgr.closeHandler[key]
	if !ok {
		return false
	}
	cancel()
	delete(mgr.closeHandler, key)

	return true
}

// ClearWatch 清除所有监视
func (mgr *EtcdWatcher) ClearWatch() {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	for k := range mgr.closeHandler {
		mgr.closeHandler[k]()
	}
	mgr.closeHandler = make(map[string]func())
}

// Close 关闭
func (mgr *EtcdWatcher) Close(wait bool) {
	mgr.ClearWatch()

	if wait {
		mgr.wg.Wait()
	}

	mgr.cli.Close()
	mgr.cli = nil
}

func (mgr *EtcdWatcher) watch(ctx context.Context, key string, prefix bool, listener Listener) error {
	defer mgr.wg.Done()

	ctx1, cancel := context.WithTimeout(context.Background(), timeOut)
	defer cancel()
	var getResp *clientv3.GetResponse
	var err error


	if prefix {
		getResp, err = mgr.cli.Get(ctx1, key, clientv3.WithPrefix())
	} else {
		getResp, err = mgr.cli.Get(ctx1, key)
	}

	if err != nil {
		return err
	}

	if len(getResp.Kvs)>0 {
		for _, ev := range getResp.Kvs {
			listener.Set(key, ev.Key, ev.Value)
		}
		listener.SetOk(key)
	} else {
		listener.Empty(key)
	}

	var watchChan clientv3.WatchChan
	if prefix {
		watchChan = mgr.cli.Watch(context.Background(), key, clientv3.WithPrefix(), clientv3.WithRev(getResp.Header.Revision+1))
	} else {
		watchChan = mgr.cli.Watch(context.Background(), key, clientv3.WithRev(getResp.Header.Revision+1))
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case resp := <-watchChan:
			err := resp.Err()
			if err != nil {
				return err
			}
			for _, ev := range resp.Events {
				if ev.IsCreate() {
					listener.Create(key, ev.Kv.Key, ev.Kv.Value)
				} else if ev.IsModify() {
					listener.Modify(key, ev.Kv.Key, ev.Kv.Value)
				} else if ev.Type == mvccpb.DELETE {
					listener.Delete(key, ev.Kv.Key)
				} else {
				}
			}
		}
	}
}
