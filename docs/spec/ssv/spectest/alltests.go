package spectest

import (
	tests2 "github.com/bloxapp/ssv/docs/spec/ssv/spectest/tests"
	"github.com/bloxapp/ssv/docs/spec/ssv/spectest/tests/consensus"
)

var AllTests = []*tests2.SpecTest{
	//tests2.HappyFullFlow(),
	//
	//postconsensus.ValidMessage(),
	//postconsensus.NoSigners(),
	//postconsensus.InvaliSignature(),
	//postconsensus.WrongSigningRoot(),
	//postconsensus.WrongBeaconChainSig(),
	//postconsensus.NoMessageSigners(),
	//postconsensus.MultipleSigners(),
	//postconsensus.MultipleMessageSigners(),
	//
	//consensus.UnknownDuty(),
	//consensus.WrongDutyRole(),
	//consensus.WrongDutyPubKey(),
	//consensus.DecidedValueSlotMismatch(),
	consensus.AttestationDataNil(),
}
