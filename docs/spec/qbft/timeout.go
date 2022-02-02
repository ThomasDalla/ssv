package qbft

import "github.com/pkg/errors"

type Timer interface {
	// TimeoutForRound will reset running timer if exists and will start a new timer for a specific round
	TimeoutForRound(round Round)
}

func uponTimout(state State) error {
	state.SetRound(state.GetRound() + 1)
	roundChange := createRoundChange(state, state.GetRound())

	if err := state.GetConfig().GetP2PNetwork().BroadcastSignedMessage(roundChange); err != nil {
		return errors.Wrap(err, "failed to broadcast round change message")
	}

	return nil
}
