package agent

import (
	"context"
	"io"
	"net"
	"sync"

	log "github.com/sirupsen/logrus"
)

const (
	tcpMaxSendQueue = 10000
	tcpNodeOnline   = 1 << iota
)

// NodeFunc TODO
type NodeFunc func(n *ClientNode)

// NodeOption TODO
type NodeOption func(n *ClientNode)

// ClientNode TODO
type ClientNode struct {
	conn              *net.Conn
	sendQueue         chan []byte
	recvBuf           []byte
	status            int
	wg                *sync.WaitGroup
	lock              *sync.Mutex
	onclose           []NodeFunc
	ctx               context.Context
	onMessageCallback []OnServerMessageFunc
	codec             ICodec
}

func setOnMessage(f ...OnServerMessageFunc) NodeOption {
	return func(n *ClientNode) {
		n.onMessageCallback = append(n.onMessageCallback, f...)
	}
}

func newNode(ctx context.Context, conn *net.Conn, codec ICodec, opts ...NodeOption) *ClientNode {
	node := &ClientNode{
		conn:              conn,
		sendQueue:         make(chan []byte, tcpMaxSendQueue),
		recvBuf:           make([]byte, 0),
		status:            tcpNodeOnline,
		ctx:               ctx,
		lock:              new(sync.Mutex),
		onclose:           make([]NodeFunc, 0),
		wg:                new(sync.WaitGroup),
		onMessageCallback: make([]OnServerMessageFunc, 0),
		codec:             codec,
	}
	for _, f := range opts {
		f(node)
	}
	go node.asyncSendService()
	return node
}

func setOnNodeClose(f NodeFunc) NodeOption {
	return func(n *ClientNode) {
		n.onclose = append(n.onclose, f)
	}
}

func (node *ClientNode) close() {
	node.lock.Lock()
	if node.status&tcpNodeOnline <= 0 {
		node.lock.Unlock()
		return
	}
	if node.status&tcpNodeOnline > 0 {
		node.status ^= tcpNodeOnline
		(*node.conn).Close()
		close(node.sendQueue)
	}
	log.Warnf("[W] node close")
	node.lock.Unlock()
	for _, f := range node.onclose {
		f(node)
	}
}

// Send TODO
func (node *ClientNode) Send(msgID int64, data []byte) (int, error) {
	sendData := node.codec.Encode(msgID, data)
	return (*node.conn).Write(sendData)
}

// AsyncSend TODO
func (node *ClientNode) AsyncSend(msgID int64, data []byte) {
	if node.status&tcpNodeOnline <= 0 {
		return
	}
	senddata := node.codec.Encode(msgID, data)
	node.sendQueue <- senddata
}

func (node *ClientNode) asyncSendService() {
	node.wg.Add(1)
	defer node.wg.Done()
	for {
		if node.status&tcpNodeOnline <= 0 {
			log.Info("[I] tcp node is closed, clientSendService exit.")
			return
		}
		select {
		case msg, ok := <-node.sendQueue:
			if !ok {
				log.Info("[I] tcp node sendQueue is closed, sendQueue channel closed.")
				return
			}
			size, err := (*node.conn).Write(msg)
			if err != nil {
				log.Errorf("[E] tcp send to %s error: %v", (*node.conn).RemoteAddr().String(), err)
				node.close()
				return
			}
			if size != len(msg) {
				log.Errorf("[E] %s send not complete: %v", (*node.conn).RemoteAddr().String(), msg)
			}
		case <-node.ctx.Done():
			log.Debugf("[D] context is closed, wait for exit, left: %d", len(node.sendQueue))
			if len(node.sendQueue) <= 0 {
				log.Info("[I] tcp service, clientSendService exit.")
				return
			}
		}
	}
}

func (node *ClientNode) onMessage(msg []byte) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("[E] Unpack recover##########%+v, %+v", err, node.recvBuf)
			node.recvBuf = make([]byte, 0)
		}
	}()
	node.recvBuf = append(node.recvBuf, msg...)
	for {
		bufferLen := len(node.recvBuf)
		msgID, content, pos, err := node.codec.Decode(node.recvBuf)
		if err != nil {
			node.recvBuf = make([]byte, 0)
			log.Errorf("[E] node.recvBuf error %v", err)
			return
		}
		if msgID <= 0 {
			return
		}
		if len(node.recvBuf) >= pos {
			node.recvBuf = append(node.recvBuf[:0], node.recvBuf[pos:]...)
		} else {
			node.recvBuf = make([]byte, 0)
			log.Errorf("[E] pos %v(olen=%v) error, cmd=%v, content=%v(%v) len is %v, data is: %+v", pos, bufferLen, msgID, content, string(content), len(node.recvBuf), node.recvBuf)
		}
		for _, f := range node.onMessageCallback {
			f(node, msgID, content)
		}
	}
}

func (node *ClientNode) readMessage() {
	for {
		readBuffer := make([]byte, 4096)
		size, err := (*node.conn).Read(readBuffer)
		if err != nil && err != io.EOF {
			// TODO:
			log.Warnf("[W] tcp node disconnect with error: %v, %v", (*node.conn).RemoteAddr().String(), err)
			node.close()
			return
		}
		node.onMessage(readBuffer[:size])
	}
}
