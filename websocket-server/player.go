package main

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"
	"websocket_server_rock_paper_scissors/gopool"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/mailru/easygo/netpoll"
)

type Player struct {
	Uuid           string
	io             sync.Mutex
	conn           io.ReadWriteCloser
	out            chan []byte
	pool           *gopool.Pool
	mu             sync.RWMutex
	MasterPlayer   bool
	Choice         int8
	Score          int
	RoundScore     int
	Result         uint8
	Results        map[uint]Object
	Nickname       string
	connDescriptor *netpoll.Desc
}

// readRequests reads json-rpc request from connection.
// It takes io mutex.
func (p *Player) readRequest() (*Request, error) {
	p.io.Lock()
	defer p.io.Unlock()
	h, r, err := wsutil.NextReader(p.conn, ws.StateServerSide)
	if err != nil {
		return nil, err
	}
	if h.OpCode.IsControl() {
		//DETECT REFRESHED OR CLOSED BROWSER,
		//SO PLAYER MUST BE UNREGISTERED
		return nil, wsutil.ControlFrameHandler(p.conn, ws.StateServerSide)(h, r)

	}
	req := &Request{}
	decoder := json.NewDecoder(r)

	if err := decoder.Decode(req); err != nil {
		return req, nil
	}

	return req, nil
}

func (p *Player) writeErrorTo(req *Request, err string) error {
	return p.write(Error{
		ID:    req.ID,
		Error: err,
	})
}

func (p *Player) write(x interface{}) error {
	w := wsutil.NewWriter(p.conn, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(w)

	p.io.Lock()
	defer p.io.Unlock()

	if err := encoder.Encode(x); err != nil {
		return err
	}

	return w.Flush()
}

// emit sends message to each player.
func (p *Player) emit(method string, params Object) error {
	var buf bytes.Buffer

	w := wsutil.NewWriter(&buf, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(w)

	r := Request{Method: method, Params: params}
	if err := encoder.Encode(r); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}

	p.out <- buf.Bytes()

	return nil
}

// writer writes messages from each player.out channel.
func (p *Player) writer() {
	for bts := range p.out {
		p.pool.Schedule(func() {
			p.writeRaw(bts)
		})
	}
}

func (p *Player) writeRaw(bts []byte) error {
	p.io.Lock()
	defer p.io.Unlock()

	_, err := p.conn.Write(bts)

	return err
}
