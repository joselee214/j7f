package grace

import (
	"errors"
	"fmt"
	"github.com/joselee214/j7f/internal/log"
	"github.com/joselee214/j7f/components/service_register"
	"os"
	"os/exec"
	"os/signal"
	//"sync"
	"syscall"
	"time"
)

type Server struct {
	GraceListener graceListener
	Rr service_register.RegisterOpts
	sigChan       chan os.Signal
	SignalHooks   map[int]map[os.Signal][]func()

	//wg sync.WaitGroup

	isChild bool

	log log.Logger

	runing bool
}

func (srv *Server) ListenAndServe() (err error) {
	srv.runing = true
	go srv.handleSignals()

	srv.log.Info( " => server run as child : ",srv.isChild )

	if srv.isChild {
		process, err := os.FindProcess(os.Getppid())
		if err != nil {
			return err
		}
		err = process.Signal(syscall.SIGTERM)  //os.Interrupt
		if err != nil {
			return err
		}
	}

	return srv.Serve()
}

func (srv *Server) Serve() (err error) {
	//srv.wg.Add(1)
	err = srv.GraceListener.StartServ()
	//srv.wg.Wait()
	//fmt.Println("==================af StartServ")
	srv.runing = false
	return
}

func (srv *Server) handleSignals() {
	var sig os.Signal
	signal.Notify(
		srv.sigChan,
		hookableSignals...,
	)


	for {
		if srv.runing == false {
			break
		}
		sig = <-srv.sigChan

		srv.log.Infof(" => handleSignals Received %v",sig)

		srv.signalHooks(PreSignal, sig)

		switch sig {
		case syscall.SIGHUP:
			err := srv.fork()
			if err != nil {
				srv.log.Errorf("Fork err: %s", err)
			}
		case syscall.SIGINT:
			DefaultTimeout = 0
			srv.shutdown()
		case syscall.SIGTERM:
			srv.shutdown()
		case syscall.SIGKILL:
			DefaultTimeout = 0
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

	if srv.runing==false {
		return
	}
	srv.runing = false

	if srv.Rr.RegisterData != nil {
		srv.Rr.DeRegister()
	}

	if DefaultTimeout > 0 {
		time.AfterFunc( DefaultTimeout , func() {
			srv.shutClose()
		}  )
	} else {
		srv.shutClose()
	}
	//srv.serverTimeout(DefaultTimeout)
	//	srv.wg.Done()
	//else {
	//	srv.wg.Done()
	//}
}

func (srv *Server) shutClose(){
	err := srv.Close()
	if err != nil {
		srv.log.Errorf("%d Listener.Close() error:", syscall.Getpid(), err)
	} else {
		srv.log.Infof(" => Server %s closed / pid %d ", srv.GraceListener.GetAddress().String() , syscall.Getpid() )
	}
}

//func (srv *Server) serverTimeout(d time.Duration) {
//	defer func() {
//		if r := recover(); r != nil {
//			srv.log.Infof("WaitGroup at 0 ,%v", r)
//		}
//	}()
//
//	for {
//		if srv.runing == false {
//			break
//		}
//		srv.wg.Done()
//		time.Sleep( 200 * time.Millisecond) //交出携程控制权
//	}
//}

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
			panic(err)
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
			if arg == "--graceful" {
				continue
			}
			args = append(args, arg)
		}
	}
	args = append(args, "--graceful")

	srv.log.Info(" ==> fork run : ",path,args)

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
