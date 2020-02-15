package grace

import (
	"errors"
	"fmt"
	"go.7yes.com/j7f/internal/log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	GraceListener graceListener
	sigChan       chan os.Signal
	SignalHooks   map[int]map[os.Signal][]func()

	wg sync.WaitGroup

	isChild bool

	log log.Logger
}

func (srv *Server) ListenAndServe() (err error) {
	go srv.handleSignals()

	if srv.isChild {
		process, err := os.FindProcess(os.Getppid())
		if err != nil {
			return err
		}
		err = process.Signal(os.Interrupt)
		if err != nil {
			return err
		}
	}

	return srv.Serve()
}

func (srv *Server) Serve() (err error) {
	srv.wg.Add(1)
	err = srv.GraceListener.StartServ()
	srv.wg.Wait()
	return
}

func (srv *Server) handleSignals() {
	var sig os.Signal
	signal.Notify(
		srv.sigChan,
		hookableSignals...,
	)

	for {
		sig = <-srv.sigChan
		srv.signalHooks(PreSignal, sig)
		switch sig {
		case syscall.SIGHUP:
			err := srv.fork()
			if err != nil {
				srv.log.Errorf("Fork err: %s", err)
			}
		case syscall.SIGINT:
			srv.shutdown()
		case syscall.SIGTERM:
			srv.shutdown()
		default:
			srv.log.Infof("Received %v: nothing i care about...\n", sig)
		}
		srv.signalHooks(SufSignal, sig)
	}

}

func (srv *Server) signalHooks(ppFlag int, sig os.Signal) {
	if _, notSet := srv.SignalHooks[ppFlag][sig]; !notSet {
		return
	}
	for _, f := range srv.SignalHooks[ppFlag][sig] {
		f()
	}
}

func (srv *Server) shutdown() {
	if DefaultTimeout >= 0 {
		go srv.serverTimeout(DefaultTimeout)
	}
	err := srv.Close()
	if err != nil {
		srv.log.Errorf("%d Listener.Close() error:", syscall.Getpid(), err)
	} else {
		srv.log.Infof("%d Listener closed.", syscall.Getpid())
	}
	srv.wg.Done()
}

func (srv *Server) Close() (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}
		}
	}()

	srv.GraceListener.GracefulStop()

	return
}

func (srv *Server) fork() (err error) {
	regLock.Lock()
	defer regLock.Unlock()
	if runningServersForked {
		return
	}
	runningServersForked = true

	var files = make([]*os.File, len(runningServers))
	for _, srvPtr := range runningServers {
		switch srvPtr.GraceListener.(type) {
		case graceListener:
			fl, err := srvPtr.GraceListener.GetListener().File()
			if err != nil {
				srv.log.Errorf("Get listener file error: %s", err)
				continue
			}
			files = append(files, fl)
		}
	}

	path := os.Args[0]
	var args []string
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			if arg == "-graceful" {
				break
			}
			args = append(args, arg)
		}
	}
	args = append(args, "-graceful")

	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = files
	err = cmd.Start()
	if err != nil {
		srv.log.Errorf("Restart: Failed to launch, error: %s", err)
	}
	return
}

func (srv *Server) serverTimeout(d time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			srv.log.Infof("WaitGroup at 0 ,%v", r)
		}
	}()

	for {
		srv.wg.Done()
	}

}

// RegisterSignalHook registers a function to be run PreSignal or PostSignal for a given signal.
func (srv *Server) RegisterSignalHook(ppFlag int, sig os.Signal, f func()) (err error) {
	if ppFlag != PreSignal && ppFlag != SufSignal {
		err = fmt.Errorf("invalid ppFlag argument. Must be either grace.PreSignal or grace.PostSignal")
		return
	}
	for _, s := range hookableSignals {
		if s == sig {
			srv.SignalHooks[ppFlag][sig] = append(srv.SignalHooks[ppFlag][sig], f)
			return
		}
	}
	err = fmt.Errorf("signal '%+v' is not supported", sig)
	return
}
