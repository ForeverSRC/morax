# Morax
## 概述
基于`net/rpc`的go语言rpc框架。

角色：

* provider
    * 服务提供者
* comsumer
    * 服务消费者
* registry
    * 注册中心
    * Morax基于consul注册中心

## 快速开始
### provider
#### 实现rpc服务
```golang
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
```
向消费端提供调用包
```go
import (
	"github.com/ForeverSRC/morax/error"
)

const PROVIDER_NAME = "sample-hello-service"

type HelloRequest struct {
	Target string `json:"target"`
}

type HelloResponse struct {
	Result string `json:"result"`
}

type HelloServiceConsumer struct {
	Hello func(res HelloRequest) (HelloResponse, error.RpcError)
	Bye   func(res HelloRequest) (HelloResponse, error.RpcError)
}
```
启动服务
```go
func main() {
	// 加载配置
	config.Load()

	// 注册提供的rpc服务
	err := provider.RegisterProvider(PROVIDER_NAME, new(HelloService))
	if err != nil {
		log.Fatal(err)
	}

	// 启动服务
	provider.ListenAndServe()
	gracefulShutdown()
}
```
#### consumer
消费端只需要加载对应配置，注册消费的服务，即可使用
```go
package main

import (
	"github.com/ForeverSRC/morax/config"
	"github.com/ForeverSRC/morax/consumer"
	"log"
	"net/http"
)

import (
	. "github.com/ForeverSRC/morax-sample/helloworld/contract"
)

var p = HelloServiceConsumer{}

func main() {
	// 加载配置
	config.Load()
	
	// 注册消费的服务
	err := consumer.RegistryConsumer(PROVIDER_NAME, &p)
	if err != nil {
		log.Fatal("error: ", err)
	}

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		// 调用
		res, rpcErr := p.Hello(HelloRequest{Target: "World"})
		if rpcErr.Err != nil {
			w.Write([]byte(rpcErr.Err.Error()))
		}else{
			w.Write([]byte(res.Result))
		}
	})

	http.HandleFunc("/bye", func(w http.ResponseWriter, r *http.Request) {
		// 调用
		res, rpcErr:= p.Bye(HelloRequest{Target: "World"})
		if rpcErr.Err != nil {
			w.Write([]byte(rpcErr.Err.Error()))
		}else{
			w.Write([]byte(res.Result))
		}
	})

	log.Println("consumer service begin listening....")
	log.Fatal(http.ListenAndServe(":9090", nil))

}
```