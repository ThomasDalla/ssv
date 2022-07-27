package protcolp2p

import (
	"context"
	"github.com/bloxapp/ssv/protocol/v1/message"
	"go.uber.org/zap"
	"time"
)

// WaitForMinPeers waits until there are minPeers conntected for the given validator
func WaitForMinPeers(pctx context.Context, logger *zap.Logger, subscriber Subscriber, vpk message.ValidatorPK, minPeers int, interval time.Duration) error {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	for ctx.Err() == nil {
		time.Sleep(interval)
		logger.Debug("PEER TEST - wait for peers")
		peers, err := subscriber.Peers(vpk)
		if err != nil {
			logger.Warn("could not get peers of topic", zap.Error(err))
			continue
		}
		if len(peers) >= minPeers {
			return nil
		}
	}

	return ctx.Err()
}
