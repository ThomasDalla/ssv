package ssv

import (
	spec "github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/bloxapp/ssv/beacon"
	"github.com/bloxapp/ssv/docs/spec/qbft"
	"github.com/pkg/errors"
)

// postConsensusState holds all the relevant progress the duty runner made for finalizing duty execution
type postConsensusState struct {
	attestation *spec.Attestation
	proposal    *spec.SignedBeaconBlock

	collectedPartialSigs map[qbft.NodeID][]byte
	postConsensusSigRoot []byte

	qbftHeight uint64 // post consensus for QBFT height

	finished bool
}

// ReconstructAttestationSig aggregates collected partial sigs, reconstructs a valid sig and returns an attestation obj with the reconstructed sig
func (pcs *postConsensusState) ReconstructAttestationSig() (*spec.Attestation, error) {
	panic("implement")
}

func (pcs *postConsensusState) HasPostConsensusSigQuorum() bool {
	panic("implement")
}

// Finished returns true if post consensus duty execution is finished
func (pcs *postConsensusState) Finished() bool {
	return pcs.finished
}

// DutyRunner is manages the execution of a duty from start to finish, it can only execute 1 duty at a time.
// Prev duty must finish before the next one can start.
type DutyRunner struct {
	beaconRoleType     beacon.RoleType
	validatorPK        []byte
	runningDuty        *beacon.Duty
	storage            Storage
	postConsensusState *postConsensusState
	qbftController     qbft.Controller
	nodeID             qbft.NodeID
}

// CanStartNewDuty returns nil if:
// - no running instance exists or
// - a QBFT instance decided and all post consensus sigs collectd or
// - a QBFT instance decided and 32 slots passed from decided duty
// else returns an error
func (dr *DutyRunner) CanStartNewDuty(duty *beacon.Duty) error {
	if dr.runningDuty != nil {
		if dr.
		// if 32 slots (1 epoch) passed from running duty, start a new duty
		if dr.runningDuty.Slot+32 < duty.Slot {
			return nil
		}
	}
	if !dr.postConsensusState.Finished() {
		return errors.New("duty post consensus is running")
	}

	return nil
}

// PostConsensusStateForHeight returns a postConsensusState instance for a specific height
func (dr *DutyRunner) PostConsensusStateForHeight(height uint64) *postConsensusState {
	if dr.postConsensusState != nil && dr.postConsensusState.qbftHeight == height {
		return dr.postConsensusState
	}
	return nil
}

func (dr *DutyRunner) resetForNewDuty() {
	dr.runningDuty = nil
	dr.postConsensusState = nil
}

// decideRunningInstance sets the decided duty and partially signs the decided data
func (dr *DutyRunner) decideRunningInstance(decidedValue consensusInputData, signer beacon.Signer) error {
	if err := dr.storage.SaveHighestDecided(dr.validatorPK, dr.beaconRoleType, decidedValue); err != nil {
		return errors.Wrap(err, "could not save decided")
	}

	dr.runningDuty = decidedValue.Duty

	switch dr.runningDuty.Type {
	case beacon.RoleTypeAttester:
		signedAttestation, r, err := signer.SignAttestation(decidedValue.AttestationData, dr.runningDuty, dr.runningDuty.PubKey[:])
		if err != nil {
			return errors.Wrap(err, "failed to sign attestation")
		}

		dr.postConsensusState = &postConsensusState{
			attestation:          signedAttestation,
			postConsensusSigRoot: ensureRoot(r),
			collectedPartialSigs: map[qbft.NodeID][]byte{},
		}
		return nil
	default:
		return errors.Errorf("unknown duty %s", dr.runningDuty.Type.String())
	}
}

// ensureRoot ensures that root will have sufficient allocated memory
// otherwise we get panic from bls:
// github.com/herumi/bls-eth-go-binary/bls.(*Sign).VerifyByte:738
func ensureRoot(root []byte) []byte {
	n := len(root)
	if n == 0 {
		n = 1
	}
	tmp := make([]byte, n)
	copy(tmp[:], root[:])
	return tmp[:]
}
