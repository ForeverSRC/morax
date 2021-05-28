# Provider

## 使用约定

1. 一个provider的服务名唯一，消费者通过服务名确定provider实例
2. 方法的入参与返回值定义为结构体
3. 向消费者提供的包包含如下两部分：
   1. provider在注册中心上的服务名
   2. 提供方法的入参和返回值结构体
   3. 供消费者订阅时用的方法结构体

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

## 内部实现

### 1.初始化

初始化过程：

* 确定provider的ip和port
* 初始化底层`rpc.Server`
* 关闭标志位初始化为`false`

### 2.注册提供的方法

底层通过`net/rpc`包的使用方式一致。

> 注意：服务的name统一为provider name，而不是具体的结构体的名字

### 3.启动服务

在一个单独的goroutine中监听对应的ip地址和端口。

### 4.提供服务

对于每个新的消费者的链接，provider启动一个新的goroutine进行服务。

`net/rpc`包中，默认客户端和服务端之间通过单一长链接进行通信，morax的消费者和提供者之间也默认采用单一长链接。

### 5.优雅关机

rpc 服务端优雅关机原理

> - 停止时，先标记为不接收新请求，新请求过来时直接报错，让客户端重试其它机器。
> - 检测正在运行的线程，等待线程执行完毕

Provider优雅涉及到的资源：

* 每个消费者的长链接
* provider用于监听的listener

#### Provider优雅关机过程

<img src="./img/provider_shutdown.png" style="zoom: 50%;" />

> 主要参考`net/http`包中的`Shutdown()`方法。

#### 实现

`net/rpc`包中，默认客户端主动关闭链接，服务端没有对应的API主动关闭链接。morax通过自定义的编解码器(`rpc.ServerCodec`)和链接(`net.Conn`)实现链接的主动关闭。

##### 自定义链接

参考并使用`net/http`包中的链接状态(`ConnState`)，对一个`net.Conn`接口的实例进行包装：

```go
type rpcConn struct {
	curState struct{ atomic uint64 }
	rwc      net.Conn
}
```

**状态转换**

![](./img/conn_state.png)

设置与获取状态的方法参考`net/http`包的实现：

```go
func (rc *rpcConn) setState(state http.ConnState) {
	if state > 0xff || state < 0 {
		panic("internal error")
	}
	packedState := uint64(time.Now().Unix()<<8) | uint64(state)
	atomic.StoreUint64(&rc.curState.atomic, packedState)
}

func (rc *rpcConn) getState() (state http.ConnState, unixSec int64) {
	packedState := atomic.LoadUint64(&rc.curState.atomic)
	return http.ConnState(packedState & 0xff), int64(packedState >> 8)
}
```

##### 自定义编解码器

编解码器对应通信协议，目前morax基于json协议进行通信。参考`net/rpc/jsonrpc`包中的`serverCodec`实现，自定义服务端编解码器：

```go
type JsonServerCodec struct {
	dec  *json.Decoder // for reading JSON values
	enc  *json.Encoder // for writing JSON values
	conn io.Closer

	req serverRequest

	mutex   sync.Mutex // protects seq, pending
	seq     uint64
	pending map[uint64]*json.RawMessage
  // 新增
	isClose types.AtomicBool
	server  *Service
}
```

其中，`isClose`维护编解码器的关闭状态，`server`指针用于使用当前编解码器的rpc server跟踪当前编解码器。

* 调用`ReadRequestHeader()`时，如果编解码器处于关闭状态，则返回`io.EOF`
* 调用`ReadRequestBody()`时，如果编解码器处于关闭状态，则返回`io.EOF`
* 调用`Close()`时，如果编解码器已经处于关闭状态，则返回，避免重复关闭

**关闭空闲链接**

主要步骤如下：

* 判断编解码器是否处于关闭状态，避免重复关闭
* 获取当前编解码器绑定的链接的状态
* 判断是否为空闲态，或处于新建态超过了一定时间
  * 参考`net/http`包
* 关闭处于空闲态的链接
  * 关闭编解码器（`Close()`方法）的实质即为关闭编解码器绑定的链接
* 将编解码器置为关闭态
* 从server中移除编解码器