package provider

import (
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

// rpcConn 对给定的实现net.Conn的链接进行包装以记录conn的状态
// 参考net/http包
type rpcConn struct {
	curState struct{ atomic uint64 } // packed (unix_time<<8|uint8(ConnState))
	rwc      net.Conn
}

func NewConn(conn net.Conn) *rpcConn {
	rc := &rpcConn{rwc: conn}
	rc.setState(http.StateNew)
	return rc
}

func (rc *rpcConn) Read(b []byte) (n int, err error) {
	// 阻塞
	n, err = rc.rwc.Read(b)
	rc.setState(http.StateActive)
	return n, err
}

func (rc *rpcConn) Write(b []byte) (n int, err error) {
	n, err = rc.rwc.Write(b)
	rc.setState(http.StateIdle)
	return n, err
}

func (rc *rpcConn) Close() error {
	err := rc.rwc.Close()
	rc.setState(http.StateClosed)
	return err
}

func (rc *rpcConn) LocalAddr() net.Addr {
	return rc.rwc.LocalAddr()
}

func (rc *rpcConn) RemoteAddr() net.Addr {
	return rc.rwc.RemoteAddr()
}

func (rc *rpcConn) SetDeadline(t time.Time) error {
	return rc.rwc.SetDeadline(t)
}

func (rc *rpcConn) SetReadDeadline(t time.Time) error {
	return rc.rwc.SetReadDeadline(t)
}

func (rc *rpcConn) SetWriteDeadline(t time.Time) error {
	return rc.rwc.SetWriteDeadline(t)
}

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
