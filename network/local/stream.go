package local

import (
	"github.com/bloxapp/ssv/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

// Stream is used by local network
type Stream struct {
	From        peer.ID
	To          peer.ID
	ReceiveChan chan *network.SyncMessage
}

// NewLocalStream returs a stream instance
func NewLocalStream(From peer.ID, To peer.ID) *Stream {
	return &Stream{
		From:        From,
		To:          To,
		ReceiveChan: make(chan *network.SyncMessage),
	}
}

// Read  implementation
func (s *Stream) Read(p []byte) (n int, err error) {
	panic("implement")
}

// WriteSynMsg implementation
func (s *Stream) WriteSynMsg(msg *network.SyncMessage) (n int, err error) {
	s.ReceiveChan <- msg
	return 0, nil
}

// Write implementation
func (s *Stream) Write(p []byte) (n int, err error) {
	panic("implement")
}

// Close implementation
func (s *Stream) Close() error {
	panic("implement")
}

// CloseWrite implementation
func (s *Stream) CloseWrite() error {
	panic("implement")
}

// RemotePeer implementation
func (s *Stream) RemotePeer() string {
	panic("implement")
}
