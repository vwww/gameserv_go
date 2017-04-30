package slime

import (
	"encoding/binary"
)

type RemotePlayer struct {
	*Player
	SendBuf chan []byte
}

var _ PlayerSender = (*RemotePlayer)(nil)

func NewRemotePlayer(name []byte, col int) *RemotePlayer {
	r := &RemotePlayer{
		nil,
		make(chan []byte, 70), // at least 2 seconds
	}
	r.Player = NewPlayer(name, col, r)
	return r
}

func (r *RemotePlayer) Recv(b []byte) {
	// For speed, process immediately instead of using chan
	if len(b) != 0 {
		b := b[len(b)-1]
		r.L = (b & 1) != 0
		r.R = (b & 2) != 0
		r.U = (b & 4) != 0
	}
}

func (r *RemotePlayer) Send(b []byte) {
	select {
	case r.SendBuf <- b:
	default:
		// queue overflow
		r.Close()
	}
}

func (r *RemotePlayer) SendWelcome() {
	b := make([]byte, len(r.Name)+4)

	binary.BigEndian.PutUint32(b, uint32(r.Color))
	b[0] = 0
	copy(b[4:], r.Name)

	r.Send(b)
}

func transformState(p1, p2 *Player, b MoveState, forP1 bool) (self, other, ball MoveState, selfKeys, otherKeys InputState) {
	// For x-coordinates, transform to the player's coordinate space:
	// Players: 0 is far from net, 1 at net
	// Ball: 0 is on our side, 2 on other side
	ball = b
	if forP1 {
		self = p1.MoveState
		other = p2.MoveState
		selfKeys = p1.InputState
		otherKeys = p2.InputState

		other.O.X = 2 - p2.O.X
	} else {
		self = p2.MoveState
		other = p1.MoveState
		selfKeys = p2.InputState
		otherKeys = p1.InputState

		self.O.X = 2 - p2.O.X
		ball.O.X = 2 - ball.O.X
	}
	return
}

func (r *RemotePlayer) SendState(self, other, ball MoveState, selfKeys, otherKeys InputState) {
	b := make([]byte, 22)

	if ball.O.Y > 0.8 {
		ball.O.Y = 0.8
	}

	const (
		DMF = 0xFFFF
		DVF = 0x3FFF
	)

	b[0] = 1
	b[1] = 0
	if selfKeys.L {
		b[1] |= 1
	}
	if selfKeys.R {
		b[1] |= 2
	}
	if selfKeys.U {
		b[1] |= 4
	}
	if otherKeys.L {
		b[1] |= 8
	}
	if otherKeys.R {
		b[1] |= 16
	}
	if otherKeys.U {
		b[1] |= 32
	}
	binary.BigEndian.PutUint16(b[2:], uint16(self.O.X*DMF))
	binary.BigEndian.PutUint16(b[4:], uint16(self.O.Y*DMF))
	binary.BigEndian.PutUint16(b[6:], uint16(int16(self.V.Y*DVF)))
	binary.BigEndian.PutUint16(b[8:], uint16((other.O.X-1)*DMF))
	binary.BigEndian.PutUint16(b[10:], uint16(other.O.Y*DMF))
	binary.BigEndian.PutUint16(b[12:], uint16(int16(other.V.Y*DVF)))
	binary.BigEndian.PutUint16(b[14:], uint16(ball.O.X*0.5*DMF))
	binary.BigEndian.PutUint16(b[16:], uint16(ball.O.Y*DMF))
	binary.BigEndian.PutUint16(b[18:], uint16(int16(ball.V.X*DVF)))
	binary.BigEndian.PutUint16(b[20:], uint16(int16(ball.V.Y*DVF)))

	r.Send(b)
}

func (r *RemotePlayer) SendEnter(name string, col int) {
	b := make([]byte, len(name)+4)

	binary.BigEndian.PutUint32(b, uint32(col))
	b[0] = 2
	copy(b[4:], name)

	r.Send(b)
}

func (r *RemotePlayer) SendLeave() { r.Send([]byte{3}) }

func (r *RemotePlayer) SendEndRound(win bool) {
	if win {
		r.Send([]byte{4})
	} else {
		r.Send([]byte{5})
	}
}

func (r *RemotePlayer) SendNextRound(isFirst bool) {
	if isFirst {
		r.Send([]byte{6})
	} else {
		r.Send([]byte{7})
	}
}