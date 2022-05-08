package messages

import (
	"github.com/bloxapp/ssv/docs/spec/qbft"
	"github.com/bloxapp/ssv/docs/spec/ssv/spectest/tests"
	"github.com/bloxapp/ssv/docs/spec/types"
	"github.com/bloxapp/ssv/docs/spec/types/testingutils"
)

// UnknownMsgType tests an unknown SSVMessage type
func UnknownMsgType() *tests.SpecTest {
	ks := testingutils.Testing4SharesSet()
	dr := testingutils.AttesterRunner(ks)

	msg := testingutils.SSVMsgAttester(nil, testingutils.PostConsensusAttestationMsgWithNoMsgSigners(ks.Shares[1], 1, qbft.FirstHeight))
	msg.MsgType = 3
	msgs := []*types.SSVMessage{
		msg,
	}

	return &tests.SpecTest{
		Name:                    "wrong SSVMessage type",
		Runner:                  dr,
		Messages:                msgs,
		PostDutyRunnerStateRoot: "74234e98afe7498fb5daf1f36ac2d78acc339464f950703b8c019892f982b90b",
		ExpectedError:           "PartialSignatureMessage Height doesn't match duty runner's Height",
	}
}
