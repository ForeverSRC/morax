# Check

每个Morax Service需要提供应对consul健康检查的服务。

Morax Service中，健康检查通过`tcp`进行。

## 内部实现

### 1.初始化

* 初始化健康检查的ip和端口号
* 初始化健康检查配置设定
* 关闭标志位初始化为`false`

### 2.启动

向注册中心注册当前Morax Service实例时，会将健康检查的设定信息一并传递。

在单独的goroutine中启动健康检查服务，监听对应的端口。

### 3.优雅关机

check service仅涉及到listener的释放。