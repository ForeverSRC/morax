package consul

import (
	"fmt"
	"net"
	"strings"
)

import (
	"github.com/ForeverSRC/morax/common/types"
	ck "github.com/ForeverSRC/morax/config/check"
	"github.com/ForeverSRC/morax/logger"
)

import (
	consulapi "github.com/hashicorp/consul/api"
)

type HealthCheckService struct {
	checkAddr string
	CheckInfo *consulapi.AgentServiceCheck
	types.AbstractService
}

func NewHealthCheckService(host string, ckf *ck.CheckConfig) *HealthCheckService {
	hcs := &HealthCheckService{
		checkAddr: fmt.Sprintf("%s:%d", host, ckf.CheckPort),
	}

	check := new(consulapi.AgentServiceCheck)
	check.TCP = hcs.checkAddr
	check.Timeout = ckf.Timeout
	check.Interval = ckf.Interval
	check.DeregisterCriticalServiceAfter = ckf.DeregisterAfter // 故障检查失败30s后 consul自动将注册服务删除

	hcs.CheckInfo = check

	hcs.InShutdown.SetFalse()
	return hcs
}

func (hcs *HealthCheckService) Shutdown() error {

	hcs.InShutdown.SetTrue()
	hcs.Mu.Lock()
	defer hcs.Mu.Unlock()
	return hcs.CloseListenersLocked()
}

func (hcs *HealthCheckService) ListenAndServe() {
	go hcs.serveCheck()
}
func (hcs *HealthCheckService) serveCheck() {
	if hcs.InShuttingDown() {
		return
	}

	listener, err := net.Listen("tcp", hcs.checkAddr)
	if err != nil {
		logger.Fatal("listen tcp error", err)
	}
	logger.Info("check:start listening on %s", hcs.checkAddr)
	hcs.TrackListener(&listener, true)
	defer hcs.TrackListener(&listener, false)

	for {
		if hcs.InShuttingDown() {
			return
		}
		conn, cErr := listener.Accept()
		if cErr != nil {
			if strings.Contains(cErr.Error(), "use of closed network connection") {
				break
			}

			logger.Error("accept error: %s", cErr)
			continue
		}
		go hcs.handleCheck(conn)
	}
}

func (hcs *HealthCheckService) handleCheck(conn net.Conn) {
	_ = conn.Close()
}
