package ssv

import (
	"bytes"
	"encoding/json"
	"github.com/bloxapp/ssv/docs/spec/types"
	"github.com/herumi/bls-eth-go-binary/bls"
	"github.com/pkg/errors"
)

type PostConsensusMessage struct {
	Height          uint64
	DutySignature   []byte // The beacon chain partial signature for a duty
	DutySigningRoot []byte // the root signed in DutySignature
	Signers         []types.NodeID
}

// Encode returns a msg encoded bytes or error
func (pcsm *PostConsensusMessage) Encode() ([]byte, error) {
	return json.Marshal(pcsm)
}

// Decode returns error if decoding failed
func (pcsm *PostConsensusMessage) Decode(data []byte) error {
	return json.Unmarshal(data, pcsm)
}

func (pcsm *PostConsensusMessage) GetRoot() []byte {
	panic("implement")
}

type SignedPostConsensusMessage struct {
	message   *PostConsensusMessage
	signature types.Signature
	signers   []types.NodeID
}

// Encode returns a msg encoded bytes or error
func (spcsm *SignedPostConsensusMessage) Encode() ([]byte, error) {
	return json.Marshal(spcsm)
}

// Decode returns error if decoding failed
func (spcsm *SignedPostConsensusMessage) Decode(data []byte) error {
	return json.Unmarshal(data, spcsm)
}

func (spcsm *SignedPostConsensusMessage) GetSignature() types.Signature {
	return spcsm.signature
}

func (spcsm *SignedPostConsensusMessage) GetSigners() []types.NodeID {
	return spcsm.signers
}

func (spcsm *SignedPostConsensusMessage) GetRoot() []byte {
	return spcsm.message.GetRoot()
}

func (spcsm *SignedPostConsensusMessage) Aggregate(signedMsg types.MessageSignature) error {
	if !bytes.Equal(spcsm.GetRoot(), signedMsg.GetRoot()) {
		return errors.New("can't aggregate msgs with different roots")
	}

	// verify no matching Signers
	for _, signerID := range spcsm.signers {
		for _, toMatchID := range signedMsg.GetSigners() {
			if signerID == toMatchID {
				return errors.New("signer IDs partially/ fully match")
			}
		}
	}

	allSigners := append(spcsm.signers, signedMsg.GetSigners()...)

	// verify and aggregate
	sig1, err := blsSig(spcsm.signature)
	if err != nil {
		return errors.Wrap(err, "could not parse DutySignature")
	}

	sig2, err := blsSig(signedMsg.GetSignature())
	if err != nil {
		return errors.Wrap(err, "could not parse DutySignature")
	}

	sig1.Add(sig2)
	spcsm.signature = sig1.Serialize()
	spcsm.signers = allSigners
	return nil
}

// MatchedSigners returns true if the provided signer ids are equal to GetSignerIds() without order significance
func (spcsm *SignedPostConsensusMessage) MatchedSigners(ids []types.NodeID) bool {
	toMatchCnt := make(map[types.NodeID]int)
	for _, id := range ids {
		toMatchCnt[id]++
	}

	foundCnt := make(map[types.NodeID]int)
	for _, id := range spcsm.GetSigners() {
		foundCnt[id]++
	}

	for id, cnt := range toMatchCnt {
		if cnt != foundCnt[id] {
			return false
		}
	}
	return true
}

func blsSig(sig []byte) (*bls.Sign, error) {
	ret := &bls.Sign{}
	if err := ret.Deserialize(sig); err != nil {
		return nil, errors.Wrap(err, "could not covert DutySignature byts to bls.sign")
	}
	return ret, nil
}
