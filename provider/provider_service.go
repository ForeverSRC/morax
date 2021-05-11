package provider

import (
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

import (
	cp "github.com/ForeverSRC/morax/config/provider"
	"github.com/ForeverSRC/morax/logger"
)

type Service struct {
	RpcAddr   string
	CheckAddr string
	server    *rpc.Server
}

var providerService *Service

func InitRpcService(c *cp.ProviderConfig) {
	providerService = &Service{
		RpcAddr:   fmt.Sprintf("%s:%d", c.Service.Host, c.Service.Port),
		CheckAddr: fmt.Sprintf("%s:%d", c.Service.Host, c.Service.Check.CheckPort),
		server:    rpc.NewServer(),
	}
}

// provider是一个结构体指针
func RegisterProvider(name string, provider interface{}) error {
	return providerService.server.RegisterName(name, provider)
}

func ListenAndServe() {
	go serve("rpc", providerService.RpcAddr, handleRpc)
	go serve("check", providerService.CheckAddr, handleCheck)
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

func serve(name string, addr string, handler func(conn net.Conn)) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatal("listen tcp error", err)
	}
	logger.Info("[%s]:start listening on %s", name, addr)
	for {
		conn, cErr := listener.Accept()
		if cErr != nil {
			logger.Error("accept error: %s", cErr)
			continue
		}

		go handler(conn)
	}
}
