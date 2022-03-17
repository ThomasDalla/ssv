package qbft

import (
	"bytes"
	"github.com/bloxapp/ssv/docs/spec/types"
	"github.com/pkg/errors"
)

// uponCommit returns true if a quorum of commit messages was received.
func uponCommit(state *State, config IConfig, signedCommit *SignedMessage, commitMsgContainer *MsgContainer) (bool, []byte, *SignedMessage, error) {
	if state.ProposalAcceptedForCurrentRound == nil {
		return false, nil, nil, errors.New("did not receive proposal for this round")
	}

	if err := validateCommit(
		state,
		config,
		signedCommit,
		state.Height,
		state.Round,
		state.ProposalAcceptedForCurrentRound,
		state.Share.Committee,
	); err != nil {
		return false, nil, nil, errors.Wrap(err, "commit msg invalid")
	}

	addMsg, err := commitMsgContainer.AddIfDoesntExist(signedCommit)
	if err != nil {
		return false, nil, nil, errors.Wrap(err, "could not add commit msg to container")
	}
	if !addMsg {
		return false, nil, nil, nil // uponCommit was already called
	}

	// calculate commit quorum and act upon it
	msgCommitData, err := signedCommit.Message.GetCommitData()
	if err != nil {
		return false, nil, nil, errors.Wrap(err, "could not get msg commit data")
	}
	quorum, commitMsgs, err := commitQuorumForValue(state, commitMsgContainer, msgCommitData.Data)
	if err != nil {
		return false, nil, nil, errors.Wrap(err, "could not calculate commit quorum")
	}
	if quorum {
		agg, err := aggregateCommitMsgs(commitMsgs)
		if err != nil {
			return false, nil, nil, errors.Wrap(err, "could not aggregate commit msgs")
		}
		return true, msgCommitData.Data, agg, nil
	}
	return false, nil, nil, nil
}

func commitQuorumForValue(state *State, commitMsgContainer *MsgContainer, value []byte) (bool, []*SignedMessage, error) {
	commitMsgs := commitMsgContainer.MessagesForRound(state.Round)
	valueFiltered := make([]*SignedMessage, 0)
	for _, msg := range commitMsgs {
		commitData, err := msg.Message.GetCommitData()
		if err != nil {
			return false, nil, errors.Wrap(err, "could not get msg commit data")
		}
		if bytes.Equal(commitData.Data, value) {
			valueFiltered = append(valueFiltered, msg)
		}
	}

	return state.Share.HasQuorum(len(valueFiltered)), valueFiltered, nil
}

func aggregateCommitMsgs(msgs []*SignedMessage) (*SignedMessage, error) {
	if len(msgs) == 0 {
		return nil, errors.New("can't aggregate zero commit msgs")
	}

	var ret *SignedMessage
	for _, m := range msgs {
		if ret == nil {
			ret = m.DeepCopy()
		} else {
			if err := ret.Aggregate(m); err != nil {
				return nil, errors.Wrap(err, "could not aggregate commit msg")
			}
		}
	}
	return ret, nil
}

// didSendCommitForHeightAndRound returns true if sent commit msg for specific Height and round
/**
!exists m :: && m in current.messagesReceived
                            && m.Commit?
                            && var uPayload := m.commitPayload.unsignedPayload;
                            && uPayload.Height == |current.blockchain|
                            && uPayload.round == current.round
                            && recoverSignedCommitAuthor(m.commitPayload) == current.id
*/
func didSendCommitForHeightAndRound(state *State, commitMsgContainer *MsgContainer) bool {
	for _, msg := range commitMsgContainer.MessagesForRound(state.Round) {
		if msg.MatchedSigners([]types.OperatorID{state.Share.OperatorID}) {
			return true
		}
	}
	return false
}

/**
Commit(
                    signCommit(
                        UnsignedCommit(
                            |current.blockchain|,
                            current.round,
                            signHash(hashBlockForCommitSeal(proposedBlock), current.id),
                            digest(proposedBlock)),
                            current.id
                        )
                    );
*/
func createCommit(state *State, config IConfig, value []byte) (*SignedMessage, error) {
	commitData := &CommitData{
		Data: value,
	}
	dataByts, err := commitData.Encode()

	msg := &Message{
		MsgType:    CommitMsgType,
		Height:     state.Height,
		Round:      state.Round,
		Identifier: state.ID,
		Data:       dataByts,
	}
	sig, err := config.GetSigner().SignRoot(msg, types.QBFTSigType, state.Share.SharePubKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed signing commit msg")
	}

	signedMsg := &SignedMessage{
		Signature: sig,
		Signers:   []types.OperatorID{state.Share.OperatorID},
		Message:   msg,
	}
	return signedMsg, nil
}

func validateCommit(
	state *State,
	config IConfig,
	signedCommit *SignedMessage,
	height Height,
	round Round,
	proposedMsg *SignedMessage,
	operators []*types.Operator,
) error {
	if signedCommit.Message.MsgType != CommitMsgType {
		return errors.New("commit msg type is wrong")
	}
	if signedCommit.Message.Height != height {
		return errors.New("commit Height is wrong")
	}
	if signedCommit.Message.Round != round {
		return errors.New("commit round is wrong")
	}

	proposedCommitData, err := proposedMsg.Message.GetCommitData()
	if err != nil {
		return errors.Wrap(err, "could not get proposed commit data")
	}
	msgCommitData, err := signedCommit.Message.GetCommitData()
	if err != nil {
		return errors.Wrap(err, "could not get msg commit data")
	}
	if !bytes.Equal(proposedCommitData.Data, msgCommitData.Data) {
		return errors.New("proposed data different than commit msg data")
	}

	if len(signedCommit.GetSigners()) != 1 {
		return errors.New("commit msg allows 1 signer")
	}
	// TODO how to process Decided msgs with multiple signer?

	if err := signedCommit.Signature.VerifyByOperators(signedCommit, config.GetSignatureDomainType(), types.QBFTSigType, operators); err != nil {
		return errors.Wrap(err, "commit msg signature invalid")
	}
	return nil
}
