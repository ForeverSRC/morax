package main

import (
	"fmt"
	"github.com/morax/config"
	"github.com/morax/consumer"
	"log"
)

import (
	. "github.com/morax/sample/helloworld/provider/contract"
)

var p = HelloServiceConsumer{}

func init() {

}

func main() {
	config.Load()
	err := consumer.RegistryConsumer(PROVIDER_NAME, &p)
	if err != nil {
		log.Fatal("error: ", err)
	}

	res, rpcErr := p.Hello(HelloRequest{Target: "World"})
	if rpcErr.Err != nil {
		fmt.Println(rpcErr.Err)
	} else {
		fmt.Printf("result:%v\n", res)
	}

	res, rpcErr = p.Bye(HelloRequest{Target: "World"})
	if rpcErr.Err != nil {
		fmt.Println(rpcErr.Err)
	} else {
		fmt.Printf("result:%v\n", res)
	}

}
