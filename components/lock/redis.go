package lock

import (
	"context"
	"github.com/joselee214/j7f/components/errors"
	"github.com/joselee214/j7f/lib/gopkg.in/redsync.v1"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MAX_REDIS_LOCK_EXPIRY = 1
	MAX_TRIES             = 3
	MAX_RETRY_DELAY       = 50
	MAX_TRY_GET_LOCK      = 200
	LOCK_STAT             = 1
	UNLOCK_STAT           = 0
)

type RedisLockConfig struct {
	Pools          []redsync.Pool    //redis pool
	Name           string            //redis lock key
	Expiry         int               //expiry can be used to set the expiry of a mutex to the given value. unit is s
	Tries          int               //Tries can be used to set the number of times lock acquire is attempted.
	RetryDelay     int               //RetryDelay can be used to set the amount of time to wait between retries.  unit is s
	//RetryDelayFunc redsync.DelayFunc //RetryDelayFunc can be used to override default delay behavior.
	DriftFactor    float64           //DriftFactor can be used to set the clock drift factor.
	MaxTryLock     int               //最大尝试获取一个锁的时间
}

type RedisLock struct {
	isLock int32
	ttl    int
	rl     *redsync.Mutex
	sl     *sync.Mutex
}

func NewRedsync(c *RedisLockConfig) *RedisLock {
	l := redsync.New(c.Pools)

	opts := make([]redsync.Option, 0)

	if c.Expiry > MAX_REDIS_LOCK_EXPIRY || c.Expiry == 0 {
		c.Expiry = MAX_REDIS_LOCK_EXPIRY
	}
	opts = append(opts, redsync.SetExpiry(time.Duration(c.Expiry)*time.Second))

	if c.Tries > MAX_TRIES || c.Tries == 0 {
		c.Tries = MAX_TRIES
	}
	opts = append(opts, redsync.SetTries(c.Tries))

	if c.RetryDelay > MAX_RETRY_DELAY || c.RetryDelay == 0 {
		c.RetryDelay = MAX_RETRY_DELAY
	}
	opts = append(opts, redsync.SetRetryDelay(time.Duration(c.RetryDelay)*time.Millisecond))

	//if c.RetryDelayFunc != nil {
	//	opts = append(opts, redsync.SetRetryDelayFunc(c.RetryDelayFunc))
	//}

	if c.DriftFactor != 0 {
		opts = append(opts, redsync.SetDriftFactor(c.DriftFactor))
	}

	if c.MaxTryLock > MAX_TRY_GET_LOCK || c.MaxTryLock == 0 {
		c.MaxTryLock = MAX_RETRY_DELAY
	}

	return &RedisLock{
		ttl: c.MaxTryLock,
		rl:  l.NewMutex(c.Name, opts...),
		sl:  &sync.Mutex{},
	}
}

func (r *RedisLock) Lock() error {
	c, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(r.ttl))
	defer cancel()

	lockStat := int32(UNLOCK_STAT)

	f := func() chan int {
		r.sl.Lock()
		t := make(chan int, 1)
		atomic.StoreInt32(&r.isLock, LOCK_STAT)
		atomic.StoreInt32(&lockStat, LOCK_STAT)
		t <- 1
		return t
	}

	select {
	case <-f():
		err := r.rl.Lock()
		if err != nil {
			r.sl.Unlock()
			atomic.StoreInt32(&r.isLock, UNLOCK_STAT)
			atomic.StoreInt32(&lockStat, UNLOCK_STAT)
			return err
		}
		return nil

	case <-c.Done():
		if lockStat == LOCK_STAT {
			r.sl.Unlock()
			atomic.StoreInt32(&r.isLock, UNLOCK_STAT)
		}
		return errors.New("try get lock timeout")
	}
}

func (r *RedisLock) Unlock() {
	if r.isLock == LOCK_STAT {
		r.sl.Unlock()
		atomic.StoreInt32(&r.isLock, UNLOCK_STAT)
		r.rl.Unlock()
	}
}
