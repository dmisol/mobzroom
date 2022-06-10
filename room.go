package mobzroom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

const (
	roomTimeout = 2 * time.Hour
	hbPeriod    = 3 * time.Second
	//reqTo       = 3 * time.Second

	rxbuflimit = 1000000
)

var (
	ErrRespTo     = errors.New("Responce Timeout")
	ErrUnexpected = errors.New("Unexpected Signal")
)

type State int32

const (
	Idle State = iota
	Initializing
	Joined
	Leaved
	Waiting
	Reinitializing
	Failed
)

type RoomClient struct {
	State
	dc DataCommon
	ws string
	mu sync.Mutex

	conn *websocket.Conn
	ctx  context.Context

	cb    func(*DataAck)
	onerr func(error)

	PeerConn *webrtc.PeerConnection
}

func NewClient(ctx context.Context, ws string, rid string, rt string, si *SessionInfo, onAck func(data *DataAck), onError func(err error)) (mr *RoomClient) {
	mr = &RoomClient{
		State: Idle,
		ctx:   ctx,
		dc: DataCommon{
			S:   "heartbeat",
			Ch:  rt,
			Rid: rid,
			Ss:  *si,
		},
		ws: ws,

		cb:    onAck,
		onerr: onError,
	}
	return
}

func (mr *RoomClient) Join(op *Op, a, v bool) {
	var err error
	if mr.conn, _, err = websocket.DefaultDialer.Dial(mr.ws, nil); err != nil {
		mr.Println("dial", err)
		mr = nil
		return
	}
	mr.conn.SetReadLimit(rxbuflimit)

	go mr.rdrun()
	go mr.wrrun()

	data := &Data{
		DataCommon: mr.dc,
		Op:         *op,
		Ve:         v,
		Ae:         a,
		Eg:         "native",
	}
	data.S = "JOIN"

	atomic.StoreInt32((*int32)(&mr.State), int32(Initializing))
	mr.Send("join", data)

}

func (mr *RoomClient) Webrtc(action string, sdp string, to interface{}, op *WrtcOp) {
	data := DataWrtc{
		DataCommon: mr.dc,
		Sdp:        sdp,
		WrtcOp:     op,
		To:         to,
	}

	switch action {
	case "offer":
		data.S = "OFFER"
	case "answer":
		data.S = "ANSWER"
	case "iceCandidate":
		data.S = "ICE"
	case "watch":
		data.S = "WATCH"
	default:
		if mr.onerr != nil {
			mr.onerr(ErrUnexpected)
			return
		}
	}
	mr.Send(action, data)
}
func (mr *RoomClient) Update(a, v bool) {}

func (mr *RoomClient) wrrun() {
	defer func() {
		data := mr.dc
		data.S = "LEAVE"
		a := ApiPayload{
			Action: "leave",
			Data:   data,
		}
		// send directly, without queueing
		b, err := json.Marshal(a)
		if err != nil {
			mr.Println("snd marshal", err)
			if mr.onerr != nil {
				go mr.onerr(err)
			}
			return
		}
		mr.send(b)

		atomic.StoreInt32((*int32)(&mr.State), int32(Failed))
		mr.conn.Close()

		mr.Println("wr stopped")
	}()

	tick := time.NewTicker(hbPeriod)
	defer tick.Stop()

	hbc := 0
	hbm := DataHb{
		DataCommon: mr.dc,
		Ack:        true,
	}
	hbm.DataCommon.S = "HEARTBEAT"
	a := ApiPayload{
		Action: "heartbeat",
	}
	for {
		select {
		case <-mr.ctx.Done():
			return
		case <-tick.C:
			if atomic.LoadInt32((*int32)(&mr.State)) != int32(Joined) {
				continue
			}

			hbm.Bid = fmt.Sprintf("%d-%X", hbc, time.Now().Unix())
			a.Data = hbm

			hbc++
			b, err := json.Marshal(a)
			if err != nil {
				mr.Println("snd marshal", err)
				if mr.onerr != nil {
					go mr.onerr(err)
				}
				continue
			}

			if mr.send(b) {
				return
			}
		}
	}
}

func (mr *RoomClient) send(b []byte) (ret bool) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	//mr.Println("tx", string(b))
	if err := mr.conn.WriteMessage(websocket.TextMessage, b); err != nil {
		//mr.Println("send", err)
		if mr.onerr != nil {
			go mr.onerr(err)
		}
		if atomic.LoadInt32((*int32)(&mr.State)) == int32(Failed) {
			ret = true
			return
		}
		atomic.StoreInt32((*int32)(&mr.State), int32(Waiting))
	}
	return
}

func (mr *RoomClient) rdrun() {
	defer func() {
		mr.Println("rd stopped")
	}()

	for {
		select {
		case <-mr.ctx.Done():
			return
		default:
			_, b, err := mr.conn.ReadMessage()
			if err != nil {
				mr.Println("read", err)
				if mr.onerr != nil {
					go mr.onerr(err)
				}
				if atomic.LoadInt32((*int32)(&mr.State)) == int32(Failed) {
					return
				}
				atomic.StoreInt32((*int32)(&mr.State), int32(Waiting))
				mr.Println("rd need to re-Join")
				return
			}
			//mr.Println("rx", string(b))
			data := &DataAck{}
			if err = json.Unmarshal(b, data); err != nil {
				mr.Println("rx", err)
				if mr.onerr != nil {
					go mr.onerr(err)
				}
				return
			}
			switch strings.ToUpper(data.S) {
			case "JOIN":
				atomic.StoreInt32((*int32)(&mr.State), int32(Joined))
			case "HEARTBEAT", "KEEP_ALIVE":
				continue
			}
			if mr.cb != nil {
				go mr.cb(data)
			}
		}
	}
}

func (mr *RoomClient) Send(action string, data interface{}) {
	a := ApiPayload{
		Action: action,
		Data:   data,
	}

	b, err := json.Marshal(a)
	if err != nil {
		mr.Println("snd marshal", err)
		if mr.onerr != nil {
			mr.onerr(err)
		}
		return
	}
	mr.send(b)
}

func (mr *RoomClient) Println(i ...interface{}) {
	log.Println("ws", i)
}
