package control

import (
	"net"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/mia0x75/copycat/g"
)

func NewControl(ctx *g.Context, opts ...ControlOption) *TcpService {
	tcp := &TcpService{
		Address:  ctx.Config.Control.Listen,
		lock:     new(sync.Mutex),
		wg:       new(sync.WaitGroup),
		listener: nil,
		ctx:      ctx,
		token:    g.GetKey(g.TOKEN_FILE),
	}
	for _, f := range opts {
		f(tcp)
	}
	return tcp
}

func ShowMember(f ShowMemberFunc) ControlOption {
	return func(tcp *TcpService) {
		tcp.showmember = f
	}
}

func Reload(f ReloadFunc) ControlOption {
	return func(tcp *TcpService) {
		tcp.reload = f
	}
}

func Stop(f StopFunc) ControlOption {
	return func(tcp *TcpService) {
		tcp.stop = f
	}
}

func (tcp *TcpService) Start() {
	go func() {
		listen, err := net.Listen("tcp", tcp.Address)
		if err != nil {
			log.Errorf("[E] tcp service listen with error: %+v", err)
			return
		}
		tcp.listener = &listen
		for {
			conn, err := listen.Accept()
			select {
			case <-tcp.ctx.Ctx.Done():
				return
			default:
			}
			if err != nil {
				log.Warnf("[W] tcp service accept with error: %+v", err)
				continue
			}
			node := newNode(tcp.ctx, &conn, nodeStop(tcp.stop), nodeReload(tcp.reload), nodeShowMembers(tcp.showmember))
			go node.readMessage()
		}
	}()
}

func (tcp *TcpService) Close() {
	log.Debugf("[D] tcp service closing, waiting for buffer send complete.")
	tcp.lock.Lock()
	defer tcp.lock.Unlock()
	if tcp.listener != nil {
		(*tcp.listener).Close()
	}
	log.Debugf("[D] tcp service closed.")
}
