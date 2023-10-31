package pchelper

import (
	"fmt"
	"github.com/smallnest/chanx"
	"golang.org/x/net/context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Set struct {
	Wg        sync.WaitGroup
	Signal    chan os.Signal
	Ctx       context.Context
	CtxCancel context.CancelFunc
	Exit      bool
	mux       sync.RWMutex
}

func New() *Set {
	ctx, cancel := context.WithCancel(context.Background())
	return &Set{
		Signal:    make(chan os.Signal, 1),
		Ctx:       ctx,
		CtxCancel: cancel,
		Exit:      false,
	}
}

func (s *Set) Add(i int) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.Wg.Add(i)
}

func (s *Set) Done() {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.Wg.Done()
}

func (s *Set) Wait() {
	s.Wg.Wait()
}

func (s *Set) SetSignal(n os.Signal) {
	s.Signal <- n
}

func (s *Set) IsClose() (state bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	state = s.Exit
	return
}

func (s *Set) Close() {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.Exit = true
}

func (s *Set) Background() {
	signal.Notify(s.Signal, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
sign:
	for f := range s.Signal {
		switch f {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			if !s.IsClose() {
				s.Close()
				s.CtxCancel()
				break sign
			}
		default:

		}
	}

	defer func() {
		fmt.Println("信號關閉")
		s.Done()
	}()
}

type ProduceFunc func(*Set, Ch, ...interface{})

func (s *Set) Produce(f func() ProduceFunc, c Ch, params ...interface{}) {
	s.Add(1)
	f()(s, c, params...)
	defer func() {
		fmt.Println("生產者關閉")
		s.Done()
	}()
}

//type ConsumeFunc func(interface{}, ...interface{})

type ConsumeFunc func(*Set, Ch, ...interface{})

func (s *Set) Consume(f func() ConsumeFunc, c Ch, params ...interface{}) {
	s.Add(1)
	f()(s, c, params...)
	defer func() {
		fmt.Println("消費者關閉")
		s.Done()
	}()
}

type Ch interface {
	Get() <-chan any
	Set(interface{})
	Len() int
	Close()
}

type Chx struct {
	Ch *chanx.UnboundedChan[any]
}

func NewChx(u *chanx.UnboundedChan[any]) *Chx {
	return &Chx{
		Ch: u,
	}
}

func (c *Chx) Get() <-chan any {
	return c.Ch.Out
}

func (c *Chx) Set(i interface{}) {
	c.Ch.In <- i
}

func (c *Chx) Len() int {
	return c.Ch.Len()
}

func (c *Chx) Close() {
	close(c.Ch.In)
}
