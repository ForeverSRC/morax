# Provider

## 使用方法

### 1.实现提供的服务

与使用go的`net/rpc`一致，需要创建提供rpc服务的结构体，实现对应的方法：

```go
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

> 使用morax时需注意：提供服务的方法，入参与返回值尽可能定义为结构体

### 2.向消费者提供包

```go
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

包括：

* 服务提供者服务名
* 方法的入参和返回值的结构体
* 消费者消费服务时调用的结构体

### 3.注册提供的服务

```go
func main() {
	config.Load()

	err := provider.RegisterProvider(PROVIDER_NAME, new(HelloService))
	if err != nil {
		log.Fatal(err)
	}

	provider.ListenAndServe()
	gracefulShutdown()
}
```

morax中，要求provider的服务名唯一，在底层调用`net/rpc`包的`RegisterName()`方法时，统一传入提供给消费者的`PROVIDER_NAME`	

随后，启动provider服务即可。

## 内部实现

### 1.初始化

初始化过程：

* 读入配置文件中`provider`部分
* 寻找合适的主机ip
* 初始化Service
* 注册实例到consul

#### 寻找合适的主机ip

使用如下函数寻找合适主机ip。

```go
func GetLocalAddr() (string, error) {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, netIf := range netInterfaces {
		if (netIf.Flags & net.FlagUp) != 0 {
			addrs, _ := netIf.Addrs()
			for _, address := range addrs {
				ipnet, ok := address.(*net.IPNet)
				if ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
					return ipnet.IP.String(), nil
				}
			}
		}
	}

	return "", errors.New("no usable local addr")
}
```

#### 注册实例到consul

服务名为对外提供的包中声明的服务名，消费者通过此服务名寻找对应服务。

服务的实例ID格式为：

```go
serviceName-Host:Port
```

健康检查采用`TCP`的方式进行。

### 2.启动服务

创建两个goroutine，用于提供rpc服务和健康检查服务。

注意：由于每个服务仅在对应的goroutine中阻塞，main goroutine需要阻塞直到需要运行结束，推荐实现优雅关机逻辑。

>由于`net/rpc`包通过启动一个goroutine去调用对应的方法，因此如果实现的方法中存在panic，则会导致服务整体宕机，最好在每个方法中实现recover