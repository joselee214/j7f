package mq

import (
	"crypto/tls"
	"github.com/nsqio/go-nsq"
	"github.com/joselee214/j7f/util"
	"net"
	"time"
)

type Config struct {
	DialTimeout time.Duration `json:"dial_timeout" default:"1s"`

	// Deadlines for network reads and writes
	ReadTimeout  time.Duration `json:"read_timeout" min:"100ms" max:"5m" default:"60s"`
	WriteTimeout time.Duration `json:"write_timeout" min:"100ms" max:"5m" default:"1s"`

	// LocalAddr is the local address to use when dialing an nsqd.
	// If empty, a local address is automatically chosen.
	LocalAddr net.Addr `json:"local_addr"`

	// Duration between polling lookupd for new producers, and fractional jitter to add to
	// the lookupd pool loop. this helps evenly distribute requests even if multiple consumers
	// restart at the same time
	//
	// NOTE: when not using nsqlookupd, LookupdPollInterval represents the duration of time between
	// reconnection attempts
	LookupdPollInterval time.Duration `json:"lookupd_poll_interval" min:"10ms" max:"5m" default:"60s"`
	LookupdPollJitter   float64       `json:"lookupd_poll_jitter" min:"0" max:"1" default:"0.3"`

	// Maximum duration when REQueueing (for doubling of deferred requeue)
	MaxRequeueDelay     time.Duration `json:"max_requeue_delay" min:"0" max:"60m" default:"15m"`
	DefaultRequeueDelay time.Duration `json:"default_requeue_delay" min:"0" max:"60m" default:"90s"`

	// Maximum number of times this consumer will attempt to process a message before giving up
	MaxAttempts uint16 `json:"max_attempts" min:"0" max:"65535" default:"5"`

	// Duration of time between heartbeats. This must be less than ReadTimeout
	HeartbeatInterval time.Duration `json:"heartbeat_interval" default:"30s"`

	// To set TLS config, use the following options:
	//
	// tls_v1 - Bool enable TLS negotiation
	// tls_root_ca_file - String path to file containing root CA
	// tls_insecure_skip_verify - Bool indicates whether this client should verify server certificates
	// tls_cert - String path to file containing public key for certificate
	// tls_key - String path to file containing private key for certificate
	// tls_min_version - String indicating the minimum version of tls acceptable ('ssl3.0', 'tls1.0', 'tls1.1', 'tls1.2')
	//
	TlsV1     bool        `json:"tls_v1"`
	TlsConfig *tls.Config `json:"tls_config"`

	// Compression Settings
	Deflate      bool `json:"deflate"`
	DeflateLevel int  `json:"deflate_level" min:"1" max:"9" default:"6"`
	Snappy       bool `json:"snappy"`

	// Size of the buffer (in bytes) used by nsqd for buffering writes to this connection
	OutputBufferSize int64 `json:"output_buffer_size" default:"16384"`
	// Timeout used by nsqd before flushing buffered writes (set to 0 to disable).
	//
	// WARNING: configuring clients with an extremely low
	// (< 25ms) output_buffer_timeout has a significant effect
	// on nsqd CPU usage (particularly with > 50 clients connected).
	OutputBufferTimeout time.Duration `json:"output_buffer_timeout" default:"250ms"`

	// Maximum number of messages to allow in flight (concurrency knob)
	MaxInFlight int `json:"max_in_flight" min:"0" default:"1"`

	// The server-side message timeout for messages delivered to this client
	MsgTimeout time.Duration `json:"msg_timeout" min:"0"`

	// producer connect pool max number, set to 0 that means Unlimited
	PoolCap int `json:"pool_cap"`
	// consumer handler concurrency number
	Concurrency int `json:"concurrency"`
}

func NewConfig(cfg *Config) (*nsq.Config, error) {
	mapConfig := util.Struct2Map(cfg)
	config := nsq.NewConfig()
	for k, v := range mapConfig {
		if !isEmpty(v) {
			err := config.Set(k, v)
			if err != nil {
				continue
			}
		}
	}

	return config, nil
}

//return true when the value is 0 or "" or false
func isEmpty(value interface{}) bool {
	switch value.(type) {
	case int:
		if value == 0 {
			return true
		}
	case string:
		if value == "" {

		}
	case time.Duration:
		if value == 0 {
			return true
		}
	case bool:
		if value == false {
			return true
		}
	case nil:
		return true
	}
	return false
}
