package mq

import (
	"errors"
	"github.com/nsqio/go-nsq"
	"math/rand"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Producer struct {
	mtx     sync.RWMutex
	addrMtx sync.RWMutex
	connMtx sync.Mutex
	wg      sync.WaitGroup

	lookupdQueryIndex int
	lookupdHTTPAddrs  []string
	nsqdTCPAddrs      []string

	config      *Config
	connections map[string]*producerPool

	exitChan chan int
}

type producerPool struct {
	closed bool
	conns  chan *nsq.Producer
}

type NodesResp struct {
	Producers []*peerInfo `json:"producers"`
}

type peerInfo struct {
	RemoteAddress    string `json:"remote_address"`
	Hostname         string `json:"hostname"`
	BroadcastAddress string `json:"broadcast_address"`
	TCPPort          int    `json:"tcp_port"`
	HTTPPort         int    `json:"http_port"`
	Version          string `json:"version"`
}

func NewProducer(cfg *Config) (*Producer, error) {
	return &Producer{
		mtx:         sync.RWMutex{},
		addrMtx:     sync.RWMutex{},
		connMtx:     sync.Mutex{},
		config:      cfg,
		connections: make(map[string]*producerPool, 0),
		exitChan:    make(chan int, 1),
	}, nil
}

func (p *Producer) ConnectToNSQLookupd(addr string, poolCap int) error {
	p.mtx.Lock()
	for _, x := range p.lookupdHTTPAddrs {
		if x == addr {
			p.mtx.Unlock()
			return nil
		}
	}
	p.lookupdHTTPAddrs = append(p.lookupdHTTPAddrs, addr)
	numLookupd := len(p.lookupdHTTPAddrs)
	p.mtx.Unlock()
	// if this is the first one, kick off the go loop
	if numLookupd == 1 {
		p.queryLookupd(poolCap)
		p.wg.Add(1)
		go p.lookupdLoop(poolCap)
	}

	return nil
}

func (p *Producer) ConnectToNSQD(addr string, poolCap int) error {
	if _, ok := p.connections[addr]; ok {
		return ErrAlreadyConnected
	}
	p.connMtx.Lock()
	defer p.connMtx.Unlock()
	p.connections[addr] = &producerPool{
		conns:  make(chan *nsq.Producer, poolCap),
		closed: false,
	}
	config, err := NewConfig(p.config)
	if err != nil {
		return err
	}
	conn, err := nsq.NewProducer(addr, config)
	if err != nil {
		return err
	}
	for i := 0; i < poolCap; i++ {
		p.connections[addr].conns <- conn
	}
	return nil
}

// return the next lookupd endpoint to query
// keeping track of which one was last used
func (p *Producer) nextLookupdEndpoint() string {
	p.mtx.RLock()
	if p.lookupdQueryIndex >= len(p.lookupdHTTPAddrs) {
		p.lookupdQueryIndex = 0
	}
	addr := p.lookupdHTTPAddrs[p.lookupdQueryIndex]
	num := len(p.lookupdHTTPAddrs)
	p.mtx.RUnlock()
	p.lookupdQueryIndex = (p.lookupdQueryIndex + 1) % num

	urlString := addr
	if !strings.Contains(urlString, "://") {
		urlString = "http://" + addr
	}

	u, err := url.Parse(urlString)
	if err != nil {
		panic(err)
	}
	if u.Path == "/" || u.Path == "" {
		u.Path = "/nodes"
	}

	return u.String()
}

func (p *Producer) queryLookupd(poolCap int) {
	retries := 0

retry:
	endpoint := p.nextLookupdEndpoint()

	//p.log(LogLevelInfo, "querying nsqlookupd %s", endpoint)

	var data NodesResp
	err := apiRequestNegotiateV1("GET", endpoint, nil, &data)
	if err != nil {
		//p.log(LogLevelError, "error querying nsqlookupd (%s) - %s", endpoint, err)
		retries++
		if retries < 3 {
			//p.log(LogLevelInfo, "retrying with next nsqlookupd")
			goto retry
		}
		return
	}

	var nsqdAddrs []string
	for _, producer := range data.Producers {
		broadcastAddress := producer.BroadcastAddress
		port := producer.TCPPort
		joined := net.JoinHostPort(broadcastAddress, strconv.Itoa(port))
		nsqdAddrs = append(nsqdAddrs, joined)
	}
	for _, addr := range nsqdAddrs {
		err = p.ConnectToNSQD(addr, poolCap)
		if err != nil && err != ErrAlreadyConnected {
			//p.log(LogLevelError, "(%s) error connecting to nsqd - %s", addr, err)
			continue
		}
	}
	p.addrMtx.Lock()
	p.nsqdTCPAddrs = nsqdAddrs
	p.addrMtx.Unlock()
}

func (p *Producer) lookupdLoop(poolCap int) {
	var ticker *time.Ticker
	ticker = time.NewTicker(p.config.LookupdPollInterval)

	for {
		select {
		case <-ticker.C:
			p.queryLookupd(poolCap)
		case <-p.exitChan:
			goto exit
		}
	}

exit:
	if ticker != nil {
		ticker.Stop()
	}
	p.wg.Done()
}

func (p *Producer) Close() {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	for _, pool := range p.connections {
		close(pool.conns)
		pool.closed = true
		for closer := range pool.conns {
			closer.Stop()
		}
	}
	close(p.exitChan)
}

func (p *Producer) Publish(topic string, body []byte) error {
	var err error
	addr, producer, err := p.getProducerConn()
	if err != nil {
		return err
	}
	retries := 0
retry:
	err = producer.Publish(topic, body)
	if err != nil {
		retries++
		if retries < 3 {
			goto retry
		}
		err = p.putProducerConn(addr, producer)
		if err != nil {
			return err
		}
		return err
	}
	err = p.putProducerConn(addr, producer)
	if err != nil {
		return err
	}
	return nil
}

func (p *Producer) getProducerConn() (string, *nsq.Producer, error) {
	if len(p.nsqdTCPAddrs) == 0 || len(p.connections) == 0 {
		return "", nil, errors.New("producer not exist")
	}
	p.addrMtx.RLock()
	defer p.addrMtx.RUnlock()
	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(len(p.nsqdTCPAddrs))
	addr := p.nsqdTCPAddrs[index]
	if p.connections[addr].closed {
		return "", nil, errors.New("connect pool is closed")
	}
	for {
		select {
		case producer, ok := <-p.connections[addr].conns:
			if ok {
				return addr, producer, nil
			}
		}
	}
}

func (p *Producer) putProducerConn(addr string, producer *nsq.Producer) error {
	p.addrMtx.Lock()
	defer p.addrMtx.Unlock()
	if p.connections[addr].closed {
		return errors.New("connect pool is closed")
	}
	select {
	case p.connections[addr].conns <- producer:
		{
			return nil
		}
	default:
		{
			producer.Stop()
			return errors.New("connect addr(" + addr + ") pool is filled")
		}
	}
}
