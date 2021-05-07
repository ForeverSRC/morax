package main

import (
	"fmt"
	"log"
)

import (
	"github.com/morax/consumer"
)

import (
	. "github.com/morax/sample/helloworld/provider/contract"
)

var p = HelloServiceConsumer{}

var c consumer.RpcConsumer

func init() {
	err := c.RegistryConsumer(PROVIDER_NAME,&p)
	if err != nil {
		log.Fatal("error: ", err)
	}
}

func main() {

	res,err:= p.Hello(HelloRequest{Target: "World"})
	if err.Err!=nil{
		fmt.Println(err)
	}else{
		fmt.Printf("result:%v\n", res)
	}

	res,err= p.Bye(HelloRequest{Target: "World"})
	if err.Err!=nil{
		fmt.Println(err)
	}else{
		fmt.Printf("result:%v\n", res)
	}

}

