package provider

import (
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

type Service struct {
	Addr string
	server *rpc.Server
}

func NewRpcService(addr string) *Service{
	return &Service{
		Addr: addr,
		server: rpc.NewServer(),
	}
}

// provider是一个结构体指针
func (s *Service) RegisterProvider(name string,provider interface{}) error {
	return s.server.RegisterName(name,provider)
}

func (s *Service) ListenAndServe() {
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		log.Fatal("listen tcp error", err)
	}
	log.Println("service start listening")
	for{
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("accept error", err)
		}
		log.Println(conn.RemoteAddr().String())

		go s.server.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
	// todo: graceful shutdown
}

