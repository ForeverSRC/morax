package provider

import (
	"github.com/ForeverSRC/morax/logger"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

type RpcConn struct {
	curState struct{ atomic uint64 } // packed (unix_time<<8|uint8(ConnState))
	rwc      net.Conn
}

func NewConn(conn net.Conn) *RpcConn {
	rc := &RpcConn{rwc: conn}
	rc.setState(http.StateNew)
	logger.Debug("new conn StateNew")
	return rc
}

func (rc *RpcConn) Read(b []byte) (n int, err error) {
	// 阻塞
	n, err = rc.rwc.Read(b)
	rc.setState(http.StateActive)
	logger.Debug("conn StateActive")
	return n, err
}

func (rc *RpcConn) Write(b []byte) (n int, err error) {
	n, err = rc.rwc.Write(b)
	rc.setState(http.StateIdle)
	logger.Debug("conn StateIdle")
	return n, err
}

func (rc *RpcConn) Close() error {
	err := rc.rwc.Close()
	rc.setState(http.StateClosed)
	logger.Debug("conn StateClosed")
	return err
}

func (rc *RpcConn) LocalAddr() net.Addr {
	return rc.rwc.LocalAddr()
}

func (rc *RpcConn) RemoteAddr() net.Addr {
	return rc.rwc.RemoteAddr()
}

func (rc *RpcConn) SetDeadline(t time.Time) error {
	return rc.rwc.SetDeadline(t)
}

func (rc *RpcConn) SetReadDeadline(t time.Time) error {
	return rc.rwc.SetReadDeadline(t)
}

func (rc *RpcConn) SetWriteDeadline(t time.Time) error {
	return rc.rwc.SetWriteDeadline(t)
}

func (rc *RpcConn) setState(state http.ConnState) {
	if state > 0xff || state < 0 {
		panic("internal error")
	}
	packedState := uint64(time.Now().Unix()<<8) | uint64(state)
	atomic.StoreUint64(&rc.curState.atomic, packedState)
}

func (rc *RpcConn) GetState() (state http.ConnState, unixSec int64) {
	packedState := atomic.LoadUint64(&rc.curState.atomic)
	return http.ConnState(packedState & 0xff), int64(packedState >> 8)
}
