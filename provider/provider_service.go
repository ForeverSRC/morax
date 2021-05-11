package provider

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

import (
	cp "github.com/ForeverSRC/morax/config/provider"
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
			log.Println("recover: rpc server error: ", err)
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
		log.Fatal("listen tcp error", err)
	}
	log.Printf("[%s]:start listening on %s", name, addr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("accept error", err)
			continue
		}

		go handler(conn)
	}
}
