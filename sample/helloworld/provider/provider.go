package main

import (
	"github.com/morax/provider"
	. "github.com/morax/sample/helloworld/provider/contract"
	"log"
)

type HelloService struct {
}

func (service *HelloService) Hello(req HelloRequest, resp *HelloResponse) error {
	*resp = HelloResponse{
		Result: "Hello " + req.Target,
	}
	return nil
}

func (service *HelloService) Bye(req HelloRequest, resp *HelloResponse) error {
	*resp = HelloResponse{
		Result: "Bye " + req.Target,
	}
	return nil
}

func main() {
	s := provider.NewRpcService("localhost:1234")
	err := s.RegisterProvider(PROVIDER_NAME, new(HelloService))
	if err != nil {
		log.Fatal(err)
	}

	s.ListenAndServe()
}
