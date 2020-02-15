package mq

import "github.com/nsqio/go-nsq"

type Consumer struct {
	*nsq.Consumer
}

type logger interface {
	Output(calldepth int, s string) error
}

func NewConsumer(topic string, channel string, cfg *Config) (*Consumer, error) {
	config, err := NewConfig(cfg)
	if err != nil {
		return nil, err
	}
	consumer, err := nsq.NewConsumer(topic, channel, config)
	if err != nil {
		return nil, err
	}
	return &Consumer{consumer}, nil
}

// ConnectToNSQLookupd adds an nsqlookupd address to the list for this Consumer instance.
//
// If it is the first to be added, it initiates an HTTP request to discover nsqd
// producers for the configured topic.
//
// A goroutine is spawned to handle continual polling.
func (c *Consumer) ConnectToNSQLookupd(addr string) error {
	return c.Consumer.ConnectToNSQLookupd(addr)
}

// ConnectToNSQLookupds adds multiple nsqlookupd address to the list for this Consumer instance.
//
// If adding the first address it initiates an HTTP request to discover nsqd
// producers for the configured topic.
//
// A goroutine is spawned to handle continual polling.
func (c *Consumer) ConnectToNSQLookupds(addresses []string) error {
	return c.Consumer.ConnectToNSQLookupds(addresses)
}

// ConnectToNSQD takes a nsqd address to connect directly to.
//
// It is recommended to use ConnectToNSQLookupd so that topics are discovered
// automatically.  This method is useful when you want to connect to a single, local,
// instance.
func (c *Consumer) ConnectToNSQD(addr string) error {
	return c.Consumer.ConnectToNSQD(addr)
}

// ConnectToNSQDs takes multiple nsqd addresses to connect directly to.
//
// It is recommended to use ConnectToNSQLookupd so that topics are discovered
// automatically.  This method is useful when you want to connect to local instance.
func (c *Consumer) ConnectToNSQDs(addresses []string) error {
	return c.Consumer.ConnectToNSQDs(addresses)
}

// DisconnectFromNSQD closes the connection to and removes the specified
// `nsqd` address from the list
func (c *Consumer) DisconnectFromNSQD(addr string) error {
	return c.Consumer.DisconnectFromNSQD(addr)
}

// DisconnectFromNSQLookupd removes the specified `nsqlookupd` address
// from the list used for periodic discovery.
func (c *Consumer) DisconnectFromNSQLookupd(addr string) error {
	return c.Consumer.DisconnectFromNSQLookupd(addr)
}

// AddHandler sets the Handler for messages received by this Consumer. This can be called
// multiple times to add additional handlers. Handler will have a 1:1 ratio to message handling goroutines.
//
// This panics if called after connecting to NSQD or NSQ Lookupd
//
// (see Handler or HandlerFunc for details on implementing this interface)
func (c *Consumer) AddHandler(handler nsq.Handler) {
	c.Consumer.AddHandler(handler)
}

// AddConcurrentHandlers sets the Handler for messages received by this Consumer.  It
// takes a second argument which indicates the number of goroutines to spawn for
// message handling.
//
// This panics if called after connecting to NSQD or NSQ Lookupd
//
// (see Handler or HandlerFunc for details on implementing this interface)
func (c *Consumer) AddConcurrentHandlers(handler nsq.Handler, concurrency int) {
	c.Consumer.AddConcurrentHandlers(handler, concurrency)
}

// SetLogger assigns the logger to use as well as a level
//
// The logger parameter is an interface that requires the following
// method to be implemented (such as the the stdlib log.Logger):
//
//    Output(calldepth int, s string)
//
func (c *Consumer) SetLogger(l logger, lvl string) {
	var level nsq.LogLevel
	switch lvl {
	case "info":
		level = nsq.LogLevelInfo
	case "error":
		level = nsq.LogLevelError
	case "warning":
		level = nsq.LogLevelWarning
	default:
		level = nsq.LogLevelDebug
	}

	c.Consumer.SetLogger(l, level)
}
