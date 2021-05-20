package provider

// 在net/rpc/jsonrpc 包基础上进行改进

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/rpc"
	"sync"
	"time"
)

import (
	"github.com/ForeverSRC/morax/common/types"
)

var errMissingParams = errors.New("jsonrpc: request body missing params")

type JsonServerCodec struct {
	dec  *json.Decoder // for reading JSON values
	enc  *json.Encoder // for writing JSON values
	conn io.Closer

	req serverRequest

	// JSON-RPC clients can use arbitrary json values as request IDs.
	// Package rpc expects uint64 request IDs.
	// We assign uint64 sequence numbers to incoming requests
	// but save the original request ID in the pending map.
	// When rpc responds, we use the sequence number in
	// the response to find the original request ID.
	mutex   sync.Mutex // protects seq, pending
	seq     uint64
	pending map[uint64]*json.RawMessage
	isClose types.AtomicBool
	server  *Service
}

// NewJsonServerCodec returns a new rpc.ServerCodec using JSON-RPC on conn.
func NewJsonServerCodec(conn io.ReadWriteCloser, p *Service) rpc.ServerCodec {
	cd := &JsonServerCodec{
		dec:     json.NewDecoder(conn),
		enc:     json.NewEncoder(conn),
		conn:    conn,
		pending: make(map[uint64]*json.RawMessage),
		server:  p,
	}
	cd.isClose.SetFalse()
	p.TrackCodec(cd, true)
	return cd
}

type serverRequest struct {
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params"`
	Id     *json.RawMessage `json:"id"`
}

func (r *serverRequest) reset() {
	r.Method = ""
	r.Params = nil
	r.Id = nil
}

type serverResponse struct {
	Id     *json.RawMessage `json:"id"`
	Result interface{}      `json:"result"`
	Error  interface{}      `json:"error"`
}

// net/rpc 的 rpc.Server 首先调用ReadRequestHeader 将读取到的请求头部进行解码
// 如果此方法返回错误，且为 io.EOF 或 io.ErrUnexpectedEOF 则返回，且不再读取req
// 此时 rpc.Server 跳出循环，不再接受任何请求，等待其余请求结束后关闭codec

func (c *JsonServerCodec) ReadRequestHeader(r *rpc.Request) error {
	// 判断是否处于关闭状态
	if c.isClose.IsSet() {
		return io.EOF
	}

	c.req.reset()
	if err := c.dec.Decode(&c.req); err != nil {
		return err
	}
	r.ServiceMethod = c.req.Method

	// JSON request id can be any JSON value;
	// RPC package expects uint64.  Translate to
	// internal uint64 and save JSON on the side.
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.seq++
	c.pending[c.seq] = c.req.Id
	c.req.Id = nil
	r.Seq = c.seq

	return nil
}

func (c *JsonServerCodec) ReadRequestBody(x interface{}) error {
	// 判断是否处于关闭状态
	if c.isClose.IsSet() {
		return io.EOF
	}

	if x == nil {
		return nil
	}
	if c.req.Params == nil {
		return errMissingParams
	}
	// JSON params is array value.
	// RPC params is struct.
	// Unmarshal into array containing struct for now.
	// Should think about making RPC more general.
	var params [1]interface{}
	params[0] = x
	return json.Unmarshal(*c.req.Params, &params)
}

var null = json.RawMessage([]byte("null"))

// 并发调用
func (c *JsonServerCodec) WriteResponse(r *rpc.Response, x interface{}) error {
	c.mutex.Lock()
	b, ok := c.pending[r.Seq]
	if !ok {
		c.mutex.Unlock()
		return errors.New("invalid sequence number in response")
	}
	delete(c.pending, r.Seq)
	c.mutex.Unlock()

	if b == nil {
		// Invalid request so no id. Use JSON null.
		b = &null
	}
	resp := serverResponse{Id: b}
	if r.Error == "" {
		resp.Result = x
	} else {
		resp.Error = r.Error
	}
	return c.enc.Encode(resp)
}

func (c *JsonServerCodec) Close() error {
	if c.isClose.IsSet() {
		return nil
	}

	c.isClose.SetTrue()
	err := c.conn.Close()
	c.server.TrackCodec(c, false)
	return err
}

func (c *JsonServerCodec) CloseIdle() (bool, error) {
	if c.isClose.IsSet() {
		return true, nil
	}

	st, unixSec := (c.conn).(*RpcConn).GetState()
	if st == http.StateNew && unixSec < time.Now().Unix()-5 {
		st = http.StateIdle
	}
	if st != http.StateIdle || unixSec == 0 {
		return false, nil
	}
	c.isClose.SetTrue()
	err := c.conn.Close()
	c.server.TrackCodec(c, false)
	return true, err

}
