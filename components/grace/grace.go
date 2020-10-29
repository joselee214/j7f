package grace

import (
	"github.com/joselee214/j7f/internal/log"
	flag "github.com/spf13/pflag"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

// Package grace use to hot reload
// Description: http://grisha.org/blog/2014/06/03/graceful-restart-in-golang/

type graceListener interface {
	GetAddress() *net.TCPAddr
	GetListener() *net.TCPListener
	GracefulStop()
	Stop()
	StartServ() error
}

const (
	// PreSignal is the position to add filter before signal
	PreSignal = iota
	// SufSignal is the position to add filter after signal
	SufSignal
)

var (
	regLock              *sync.Mutex
	runningServers       map[string]*Server
	isChild              bool
	hookableSignals      []os.Signal
	runningServersForked bool

	// DefaultTimeout is the shutdown server's timeout. default is 60s
	DefaultTimeout = 60 * time.Second
)

func init() {
	flag.BoolVar(&isChild, "graceful", false, "listen on open fd (after forking)")

	regLock = &sync.Mutex{}

	hookableSignals = []os.Signal{
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
	}
}
func NewServer(grace graceListener) (srv *Server) {
	regLock.Lock()
	defer regLock.Unlock()

	srv = &Server{
		GraceListener: grace,
		sigChan:       make(chan os.Signal),
		isChild:       isChild,
		//wg:            sync.WaitGroup{},
		log:           log.NewLoggerDefault(),
		runing:			false,
	}
	runningServers := make(map[string]*Server)
	runningServers[grace.GetAddress().String()] = srv

	return
}
