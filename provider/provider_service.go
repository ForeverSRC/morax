package provider

import (
	"fmt"
	"net"
	"net/rpc"
	"strings"
)

import (
	"github.com/ForeverSRC/morax/common/types"
	cp "github.com/ForeverSRC/morax/config/provider"
	"github.com/ForeverSRC/morax/logger"
)

type RpcProvider struct {
	RpcAddr string
	server  *rpc.Server
	types.AbstractService
	codecs map[*JsonServerCodec]struct{}
}

func NewRpcProvider(host string, pvf *cp.ProviderConfig) *RpcProvider {
	pro := &RpcProvider{
		RpcAddr: fmt.Sprintf("%s:%d", host, pvf.Service.Port),
		server:  rpc.NewServer(),
	}
	pro.InShutdown.SetFalse()
	return pro
}

// methods 是一个结构体指针
func (p *RpcProvider) RegisterProvider(name string, methods interface{}) error {
	return p.server.RegisterName(name, methods)
}

func (p *RpcProvider) ListenAndServe() {
	go p.serveRpc()
}

func (p *RpcProvider) serveRpc() {
	if p.InShuttingDown() {
		return
	}

	listener, err := net.Listen("tcp", p.RpcAddr)
	if err != nil {
		logger.Fatal("listen tcp error", err)
	}
	logger.Info("rpc:start listening on %s", p.RpcAddr)
	if !p.TrackListener(&listener, true) {
		return
	}
	// 关闭listener后 accept返回，goroutine退出，移除listener
	defer p.TrackListener(&listener, false)

	for {
		if p.InShuttingDown() {
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
		go p.handleRpc(conn)
	}
}

func (p *RpcProvider) handleRpc(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error("recover: rpc server error: %s", err)
		}
	}()
	rc := NewConn(conn)
	codec := NewJsonServerCodec(rc, p)

	p.server.ServeCodec(codec)
	logger.Debug("rpc serve codec return")
}

func (p *RpcProvider) Shutdown() error {
	// 修改关闭标识
	p.InShutdown.SetTrue()
	p.Mu.Lock()
	defer p.Mu.Unlock()
	// 关闭所有打开的listener
	return p.CloseListenersLocked()
}

func (p *RpcProvider) CloseIdleCodecs() bool {
	quiescent := true

	for cd := range p.codecs {
		flag, _ := cd.closeIdle()
		quiescent = quiescent && flag
	}

	return quiescent
}

func (p *RpcProvider) TrackCodec(codec *JsonServerCodec, add bool) {
	p.Mu.Lock()
	defer p.Mu.Unlock()
	if p.codecs == nil {
		p.codecs = make(map[*JsonServerCodec]struct{})
	}
	if add {
		p.codecs[codec] = struct{}{}
	} else {
		delete(p.codecs, codec)
	}
}
