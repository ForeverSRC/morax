package provider

import (
	"context"
	"fmt"
	"github.com/ForeverSRC/morax/common/utils"
	"github.com/ForeverSRC/morax/registry/consul"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/ForeverSRC/morax/common/types"
	cp "github.com/ForeverSRC/morax/config/provider"
	"github.com/ForeverSRC/morax/logger"
)

type Service struct {
	id         string
	RpcAddr    string
	CheckAddr  string
	server     *rpc.Server
	inShutdown types.AtomicBool
	mu         sync.Mutex
	listeners  map[*net.Listener]struct{}
	rpcWg      sync.WaitGroup
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
	return p.inShutdown.IsSet()
}

var providerService *Service

func InitRpcService(c *cp.ProviderConfig) {
	providerService = &Service{
		id:        c.GenerateProviderID(),
		RpcAddr:   fmt.Sprintf("%s:%d", c.Service.Host, c.Service.Port),
		CheckAddr: fmt.Sprintf("%s:%d", c.Service.Host, c.Service.Check.CheckPort),
		server:    rpc.NewServer(),
		rpcWg:     sync.WaitGroup{},
	}
	providerService.inShutdown.SetFalse()
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
	go p.serveRpc()
	go p.serveCheck()
}

func (p *Service) handleRpc(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error("recover: rpc server error: %s", err)
		}
		conn.Close()
		p.rpcWg.Done()
	}()
	providerService.server.ServeCodec(jsonrpc.NewServerCodec(conn))
}

func (p *Service) handleCheck(conn net.Conn) {
	defer func() {
		conn.Close()
	}()
}

func (p *Service) serveRpc() {
	if p.shuttingDown() {
		return
	}

	listener, err := net.Listen("tcp", p.RpcAddr)
	if err != nil {
		logger.Fatal("listen tcp error", err)
	}
	logger.Info("rpc:start listening on %s", p.RpcAddr)
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
			if strings.Contains(cErr.Error(), "use of closed network connection") {
				break
			}

			logger.Error("accept error: %s", cErr)
			continue
		}
		p.rpcWg.Add(1)
		go p.handleRpc(conn)
	}
}

func (p *Service) serveCheck() {
	if p.shuttingDown() {
		return
	}

	listener, err := net.Listen("tcp", p.CheckAddr)
	if err != nil {
		logger.Fatal("listen tcp error", err)
	}
	logger.Info("check:start listening on %s", p.CheckAddr)
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
			if strings.Contains(cErr.Error(), "use of closed network connection") {
				break
			}

			logger.Error("accept error: %s", cErr)
			continue
		}
		go p.handleCheck(conn)
	}
}

func Shutdown(ctx context.Context) error {
	return providerService.Shutdown(ctx)
}

func (p *Service) Shutdown(ctx context.Context) error {
	// 修改关闭标识
	p.inShutdown.SetTrue()

	p.mu.Lock()

	// 向注册中心注销实例
	_ = consul.Deregister(p.id)

	// 关闭consul client的idle connections
	consul.CloseIdleConn()

	// 关闭所有打开的listener
	lnerr := p.closeListenersLocked()

	// 等待已有线程结束
	p.rpcWg.Wait()

	p.mu.Unlock()

	pollIntervalBase := time.Millisecond
	timer := time.NewTimer(utils.NextPollInterval(&pollIntervalBase))
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
			timer.Reset(utils.NextPollInterval(&pollIntervalBase))
		}
	}
}
