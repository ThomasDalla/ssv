package synccommittee

import (
	"github.com/bloxapp/ssv/docs/spec/qbft"
	"github.com/bloxapp/ssv/docs/spec/ssv/spectest/tests"
	"github.com/bloxapp/ssv/docs/spec/types"
	"github.com/bloxapp/ssv/docs/spec/types/testingutils"
)

// HappyFlow tests a full valcheck + post valcheck + duty sig reconstruction flow
func HappyFlow() *tests.SpecTest {
	ks := testingutils.Testing4SharesSet()
	dr := testingutils.SyncCommitteeRunner(ks)

	msgs := []*types.SSVMessage{
		testingutils.SSVMsgSyncCommittee(testingutils.SignQBFTMsg(ks.Shares[1], 1, &qbft.Message{
			MsgType:    qbft.ProposalMsgType,
			Height:     qbft.FirstHeight,
			Round:      qbft.FirstRound,
			Identifier: testingutils.SyncCommitteeMsgID,
			Data:       testingutils.ProposalDataBytes(testingutils.TestSyncCommitteeConsensusDataByts, nil, nil),
		}), nil),
		testingutils.SSVMsgSyncCommittee(testingutils.SignQBFTMsg(ks.Shares[1], 1, &qbft.Message{
			MsgType:    qbft.PrepareMsgType,
			Height:     qbft.FirstHeight,
			Round:      qbft.FirstRound,
			Identifier: testingutils.SyncCommitteeMsgID,
			Data:       testingutils.PrepareDataBytes(testingutils.TestSyncCommitteeConsensusDataByts),
		}), nil),
		testingutils.SSVMsgSyncCommittee(testingutils.SignQBFTMsg(ks.Shares[2], 2, &qbft.Message{
			MsgType:    qbft.PrepareMsgType,
			Height:     qbft.FirstHeight,
			Round:      qbft.FirstRound,
			Identifier: testingutils.SyncCommitteeMsgID,
			Data:       testingutils.PrepareDataBytes(testingutils.TestSyncCommitteeConsensusDataByts),
		}), nil),
		testingutils.SSVMsgSyncCommittee(testingutils.SignQBFTMsg(ks.Shares[3], 3, &qbft.Message{
			MsgType:    qbft.PrepareMsgType,
			Height:     qbft.FirstHeight,
			Round:      qbft.FirstRound,
			Identifier: testingutils.SyncCommitteeMsgID,
			Data:       testingutils.PrepareDataBytes(testingutils.TestSyncCommitteeConsensusDataByts),
		}), nil),
		testingutils.SSVMsgSyncCommittee(testingutils.SignQBFTMsg(ks.Shares[1], 1, &qbft.Message{
			MsgType:    qbft.CommitMsgType,
			Height:     qbft.FirstHeight,
			Round:      qbft.FirstRound,
			Identifier: testingutils.SyncCommitteeMsgID,
			Data:       testingutils.CommitDataBytes(testingutils.TestSyncCommitteeConsensusDataByts),
		}), nil),
		testingutils.SSVMsgSyncCommittee(testingutils.SignQBFTMsg(ks.Shares[2], 2, &qbft.Message{
			MsgType:    qbft.CommitMsgType,
			Height:     qbft.FirstHeight,
			Round:      qbft.FirstRound,
			Identifier: testingutils.SyncCommitteeMsgID,
			Data:       testingutils.CommitDataBytes(testingutils.TestSyncCommitteeConsensusDataByts),
		}), nil),
		testingutils.SSVMsgSyncCommittee(testingutils.SignQBFTMsg(ks.Shares[3], 3, &qbft.Message{
			MsgType:    qbft.CommitMsgType,
			Height:     qbft.FirstHeight,
			Round:      qbft.FirstRound,
			Identifier: testingutils.SyncCommitteeMsgID,
			Data:       testingutils.CommitDataBytes(testingutils.TestSyncCommitteeConsensusDataByts),
		}), nil),

		testingutils.SSVMsgSyncCommittee(nil, testingutils.PostConsensusSyncCommitteeMsg(ks.Shares[1], 1)),
		testingutils.SSVMsgSyncCommittee(nil, testingutils.PostConsensusSyncCommitteeMsg(ks.Shares[2], 2)),
		testingutils.SSVMsgSyncCommittee(nil, testingutils.PostConsensusSyncCommitteeMsg(ks.Shares[3], 3)),
	}

	return &tests.SpecTest{
		Name:                    "sync committee happy flow",
		Runner:                  dr,
		Duty:                    testingutils.TestingSyncCommitteeDuty,
		Messages:                msgs,
		PostDutyRunnerStateRoot: "dcf2f1e1ce34b0c27ece5a14646f620a09276449a911e51466ac02061a8767e2",
	}
}
