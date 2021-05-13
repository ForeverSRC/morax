package provider

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"
	"sync/atomic"
	"time"
)

import (
	cp "github.com/ForeverSRC/morax/config/provider"
	"github.com/ForeverSRC/morax/logger"
)

type atomicBool int32

func (b *atomicBool) isSet() bool { return atomic.LoadInt32((*int32)(b)) != 0 }
func (b *atomicBool) setTrue()    { atomic.StoreInt32((*int32)(b), 1) }
func (b *atomicBool) setFalse()   { atomic.StoreInt32((*int32)(b), 0) }

type Service struct {
	RpcAddr    string
	CheckAddr  string
	server     *rpc.Server
	inShutdown atomicBool
	mu         sync.Mutex
	listeners  map[*net.Listener]struct{}
}

func (p *Service) closeListenersLocked() error {
	var err error
	for ln := range p.listeners {
		if cerr := (*ln).Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}

func (p *Service) trackListener(ln *net.Listener, add bool) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.listeners == nil {
		p.listeners = make(map[*net.Listener]struct{})
	}
	if add {
		if p.shuttingDown() {
			return false
		}
		p.listeners[ln] = struct{}{}
	} else {
		delete(p.listeners, ln)
	}
	return true
}

func (p *Service) numListeners() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.listeners)
}

func (p *Service) shuttingDown() bool {
	return p.inShutdown.isSet()
}

var providerService *Service

func InitRpcService(c *cp.ProviderConfig) {
	providerService = &Service{
		RpcAddr:   fmt.Sprintf("%s:%d", c.Service.Host, c.Service.Port),
		CheckAddr: fmt.Sprintf("%s:%d", c.Service.Host, c.Service.Check.CheckPort),
		server:    rpc.NewServer(),
	}
	providerService.inShutdown.setFalse()
}

// provider是一个结构体指针
func RegisterProvider(name string, provider interface{}) error {
	return providerService.RegisterProvider(name, provider)
}

func (p *Service) RegisterProvider(name string, provider interface{}) error {
	return p.server.RegisterName(name, provider)
}

func ListenAndServe() {
	providerService.ListenAndServe()
}

func (p *Service) ListenAndServe() {
	go p.serve("rpc", p.RpcAddr, handleRpc)
	go p.serve("check", p.CheckAddr, handleCheck)
}

func handleRpc(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error("recover: rpc server error: %s", err)
			conn.Close()
		}
	}()
	providerService.server.ServeCodec(jsonrpc.NewServerCodec(conn))
}

func handleCheck(conn net.Conn) {
	conn.Close()
}

func (p *Service) serve(name string, addr string, handler func(conn net.Conn)) {
	if p.shuttingDown() {
		return
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatal("listen tcp error", err)
	}
	logger.Info("%s:start listening on %s", name, addr)
	if !p.trackListener(&listener, true) {
		return
	}
	defer p.trackListener(&listener, false)

	for {
		if p.shuttingDown() {
			return
		}

		conn, cErr := listener.Accept()
		if cErr != nil {
			logger.Error("accept error: %s", cErr)
			continue
		}
		handler(conn)
	}
}

const shutdownPollIntervalMax = 500 * time.Millisecond

func Shutdown(ctx context.Context) error{
	return providerService.Shutdown(ctx)
}

func (p *Service) Shutdown(ctx context.Context) error {
	// 修改关闭标识
	p.inShutdown.setTrue()
	// 关闭所有打开的listener
	p.mu.Lock()
	lnerr := p.closeListenersLocked()
	p.mu.Unlock()

	pollIntervalBase := time.Millisecond

	//计算下次等待时间
	nextPollInterval := func() time.Duration {
		// Add 10% jitter.
		interval := pollIntervalBase + time.Duration(rand.Intn(int(pollIntervalBase/10)))
		// Double and clamp for next time.
		pollIntervalBase *= 2
		if pollIntervalBase > shutdownPollIntervalMax {
			pollIntervalBase = shutdownPollIntervalMax
		}
		return interval
	}

	timer := time.NewTimer(nextPollInterval())
	defer timer.Stop()
	for {
		// 没有打开的listener
		if p.numListeners() == 0 {
			return lnerr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			timer.Reset(nextPollInterval())
		}
	}
}
