package qbft

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	specqbft "github.com/bloxapp/ssv-spec/qbft"
	"github.com/bloxapp/ssv-spec/qbft/spectest"
	spectests "github.com/bloxapp/ssv-spec/qbft/spectest/tests"
	"github.com/bloxapp/ssv-spec/qbft/spectest/tests/commit"
	"github.com/bloxapp/ssv-spec/qbft/spectest/tests/messages"
	"github.com/bloxapp/ssv-spec/qbft/spectest/tests/prepare"
	"github.com/bloxapp/ssv-spec/qbft/spectest/tests/proposal"
	"github.com/bloxapp/ssv-spec/qbft/spectest/tests/proposer"
	"github.com/bloxapp/ssv-spec/qbft/spectest/tests/roundchange"
	spectypes "github.com/bloxapp/ssv-spec/types"
	"github.com/bloxapp/ssv-spec/types/testingutils"
	"github.com/herumi/bls-eth-go-binary/bls"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	forksprotocol "github.com/bloxapp/ssv/protocol/forks"
	beaconprotocol "github.com/bloxapp/ssv/protocol/v1/blockchain/beacon"
	"github.com/bloxapp/ssv/protocol/v1/message"
	protocolp2p "github.com/bloxapp/ssv/protocol/v1/p2p"
	qbftprotocol "github.com/bloxapp/ssv/protocol/v1/qbft"
	forksfactory "github.com/bloxapp/ssv/protocol/v1/qbft/controller/forks/factory"
	"github.com/bloxapp/ssv/protocol/v1/qbft/instance"
	"github.com/bloxapp/ssv/protocol/v1/qbft/instance/leader/constant"
	"github.com/bloxapp/ssv/protocol/v1/qbft/instance/msgcont"
	qbftstorage "github.com/bloxapp/ssv/protocol/v1/qbft/storage"
	"github.com/bloxapp/ssv/protocol/v1/validator"
	"github.com/bloxapp/ssv/storage"
	"github.com/bloxapp/ssv/storage/basedb"
	"github.com/bloxapp/ssv/utils/logex"
)

// nolint
func testsToRun() map[string]struct{} {
	testList := spectest.AllTests
	testList = []spectest.SpecTest{
		proposer.FourOperators(),
		proposer.SevenOperators(),
		proposer.TenOperators(),
		proposer.ThirteenOperators(),

		messages.RoundChangeDataInvalidJustifications(),
		messages.RoundChangeDataInvalidPreparedRound(),
		messages.RoundChangeDataInvalidPreparedValue(),
		messages.RoundChangePrePreparedJustifications(),
		messages.RoundChangeNotPreparedJustifications(),
		messages.CommitDataEncoding(),
		messages.DecidedMsgEncoding(),
		messages.MsgNilIdentifier(),
		messages.MsgNonZeroIdentifier(),
		messages.MsgTypeUnknown(),
		messages.PrepareDataEncoding(),
		messages.ProposeDataEncoding(),
		messages.MsgDataNil(),
		messages.MsgDataNonZero(),
		messages.SignedMsgSigTooShort(),
		messages.SignedMsgSigTooLong(),
		messages.SignedMsgNoSigners(),
		messages.GetRoot(),
		messages.SignedMessageEncoding(),
		messages.CreateProposal(),
		messages.CreateProposalPreviouslyPrepared(),
		messages.CreateProposalNotPreviouslyPrepared(),
		messages.CreatePrepare(),
		messages.CreateCommit(),
		messages.CreateRoundChange(),
		messages.CreateRoundChangePreviouslyPrepared(),
		messages.RoundChangeDataEncoding(),

		spectests.HappyFlow(),
		spectests.SevenOperators(),
		spectests.TenOperators(),
		spectests.ThirteenOperators(),

		proposal.HappyFlow(),
		proposal.NotPreparedPreviouslyJustification(), // TODO(nkryuchkov): failure; need to handle proposal justifications
		proposal.PreparedPreviouslyJustification(),    // TODO(nkryuchkov): failure; handle PJ (proposal justifications) & substitute identifier in justifications in mapping
		proposal.DifferentJustifications(),            // TODO(nkryuchkov): failure; handle PJ
		proposal.JustificationsNotHeighest(),          // TODO(nkryuchkov): failure; handle PJ
		proposal.JustificationsValueNotJustified(),    // TODO(nkryuchkov): failure; handle PJ
		proposal.DuplicateMsg(),                       // TODO(nkryuchkov): failure; need to check if proposal was already accepted
		proposal.FirstRoundJustification(),
		proposal.FutureRoundNoAcceptedProposal(), // TODO(nkryuchkov): failure; need to decline proposals from future rounds if no proposal was accepted for current round
		proposal.FutureRoundAcceptedProposal(),   // TODO(nkryuchkov): failure; need to accept proposals from future rounds if already accepted proposal for current round
		proposal.PastRound(),                     // TODO(nkryuchkov): failure; need to decline proposals from past rounds
		proposal.ImparsableProposalData(),
		proposal.InvalidRoundChangeJustificationPrepared(),        // TODO(nkryuchkov): failure; need to handle invalid rc justification
		proposal.InvalidRoundChangeJustification(),                // TODO(nkryuchkov): failure; need to handle invalid rc justification
		proposal.PreparedPreviouslyNoRCJustificationQuorum(),      // TODO(nkryuchkov): failure; need to check if proposal for round >1 was prepared or not prepared previously with or without quorum of prepared or round change msgs justification
		proposal.NoRCJustification(),                              // TODO(nkryuchkov): failure; need to check if proposal for round >1 was prepared or not prepared previously with or without quorum of prepared or round change msgs justification
		proposal.PreparedPreviouslyNoPrepareJustificationQuorum(), // TODO(nkryuchkov): failure; need to check if proposal for round >1 was prepared or not prepared previously with or without quorum of prepared or round change msgs justification
		proposal.PreparedPreviouslyDuplicatePrepareMsg(),          // TODO(nkryuchkov): failure; need to check if proposal for round >1 was prepared or not prepared previously with or without quorum of prepared or round change msgs justification
		proposal.PreparedPreviouslyDuplicateRCMsg(),               // TODO(nkryuchkov): failure; need to check if proposal for round >1 was prepared or not prepared previously with or without quorum of prepared or round change msgs justification
		proposal.DuplicateRCMsg(),                                 // TODO(nkryuchkov): failure; need to check if proposal for round >1 was prepared or not prepared previously with or without quorum of prepared or round change msgs justification
		proposal.InvalidPrepareJustificationValue(),               // TODO(nkryuchkov): failure; need to check if proposal for round >1 was prepared or not prepared previously but one of the prepare justifications has value != highest prepared or round != highest prepared round
		proposal.InvalidPrepareJustificationRound(),               // TODO(nkryuchkov): failure; need to check if proposal for round >1 was prepared or not prepared previously but one of the prepare justifications has value != highest prepared or round != highest prepared round
		proposal.InvalidProposalData(),
		proposal.InvalidValueCheck(), // TODO(nkryuchkov): failure; need to check message data
		proposal.MultiSigner(),
		proposal.PostDecided(),            // TODO(nkryuchkov): failure; need to avoid processing proposal message after instance became prepared or decided
		proposal.PostPrepared(),           // TODO(nkryuchkov): failure; need to avoid processing proposal message after instance became prepared or decided
		proposal.SecondProposalForRound(), // TODO(nkryuchkov): failure; need to forbid second proposal for round
		proposal.WrongHeight(),            // TODO(nkryuchkov): failure; need to expect the same error in spec if height is wrong
		proposal.WrongProposer(),
		proposal.WrongSignature(), // TODO(nkryuchkov): failure; need to check why there's no error if message is signed secret key different from signer ID's key

		prepare.DuplicateMsg(),
		prepare.HappyFlow(), // TODO(nkryuchkov): failure; substitution of message signature works incorrectly
		prepare.ImparsableProposalData(),
		prepare.InvalidPrepareData(), // TODO(nkryuchkov): failure; need to expect same error in spec if message is wrong
		prepare.MultiSigner(),        // TODO(nkryuchkov): failure; need to check that message has only 1 signer
		prepare.NoPreviousProposal(), // TODO(nkryuchkov): failure; need to fail to process message if proposal was not received
		prepare.OldRound(),           // TODO(nkryuchkov): failure; need to fail to process message if its round is not equal to current one
		prepare.FutureRound(),        // TODO(nkryuchkov): failure; need to fail to process message if its round is not equal to current one
		prepare.PostDecided(),        // TODO(nkryuchkov): failure; substitution of message signature messages works incorrectly
		prepare.WrongData(),          // TODO(nkryuchkov): failure; need to check if message data is different from proposal data
		prepare.WrongHeight(),        // TODO(nkryuchkov): failure; need to expect the same error in spec if height is wrong
		prepare.WrongSignature(),     // TODO(nkryuchkov): failure; need to check why there's no error if message is signed secret key different from signer ID's key

		commit.CurrentRound(),
		commit.FutureRound(), // TODO(nkryuchkov): failure; need to fail to process message if its round is not equal to current one
		commit.PastRound(),   // TODO(nkryuchkov): failure; need to fail to process message if its round is not equal to current one
		commit.DuplicateMsg(),
		commit.HappyFlow(),         // TODO(nkryuchkov): failure; substitution of message signature messages works incorrectly
		commit.InvalidCommitData(), // TODO(nkryuchkov): failure; need to expect same error in spec if message is wrong
		commit.PostDecided(),
		commit.WrongData1(),             // TODO(nkryuchkov): failure; need to check if message data is different from proposal data
		commit.WrongData2(),             // TODO(nkryuchkov): failure; need to check if message data is different from proposal data
		commit.MultiSignerWithOverlap(), // TODO(nkryuchkov): failure; need to fix case when multi signer commit msg which does overlap previous valid commit signers and previous valid commits
		commit.MultiSignerNoOverlap(),   // TODO(nkryuchkov): failure; substitution of message signature messages works incorrectly
		commit.Decided(),                // TODO(nkryuchkov): failure; need to fix case when multi signer commit msg which does overlap previous valid commit signers and previous valid commits
		commit.NoPrevAcceptedProposal(), // TODO(nkryuchkov): failure; need to fail to process message if proposal was not received
		commit.WrongHeight(),            // TODO(nkryuchkov): failure; need to expect the same error in spec if height is wrong
		commit.ImparsableCommitData(),   // TODO(nkryuchkov): failure; substitution of message signature messages works incorrectly
		commit.WrongSignature(),         // TODO(nkryuchkov): failure; need to check why there's no error if message is signed secret key different from signer ID's key

		roundchange.HappyFlow(),         // TODO(nkryuchkov): failure; substitution of message signature messages works incorrectly
		roundchange.F1Speedup(),         // TODO(nkryuchkov): failure; data inside ProposalAcceptedForCurrentRound misses RoundChangeJustification
		roundchange.F1SpeedupPrepared(), // TODO(nkryuchkov): failure; need to substitute identifier in justifications in mapping
		roundchange.WrongHeight(),       // TODO(nkryuchkov): failure; need to expect the same error in spec if height is wrong
		roundchange.WrongSig(),          // TODO(nkryuchkov): failure; need to check why there's no error if message is signed secret key different from signer ID's key
		roundchange.MultiSigner(),       // TODO(nkryuchkov): failure; need to check that message has only 1 signer
		roundchange.NotPrepared(),
		roundchange.Prepared(),                // TODO(nkryuchkov): failure; need to substitute identifier in justifications in mapping
		roundchange.PeerPrepared(),            // TODO(nkryuchkov): failure; need to substitute identifier in justifications in mapping
		roundchange.JustificationWrongValue(), // TODO(nkryuchkov): failure; need to substitute identifier in justifications in mapping
		roundchange.NextProposalValueWrong(),  // TODO(nkryuchkov): failure; need to substitute identifier in justifications in mapping
		roundchange.JustificationWrongRound(),
		roundchange.JustificationNoQuorum(),     // TODO(nkryuchkov): failure; need to substitute identifier in justifications in mapping
		roundchange.JustificationMultiSigners(), // TODO(nkryuchkov): failure; need to substitute identifier in justifications in mapping
		roundchange.JustificationInvalidSig(),   // TODO(nkryuchkov): failure; need to substitute identifier in justifications in mapping
		roundchange.JustificationInvalidRound(),
		roundchange.JustificationInvalidPrepareData(), // TODO(nkryuchkov): failure; need to substitute identifier in justifications in mapping
		roundchange.JustificationDuplicateMsg(),       // TODO(nkryuchkov): failure; need to substitute identifier in justifications in mapping
		roundchange.InvalidRoundChangeData(),          // TODO(nkryuchkov): failure; need to check message data
		roundchange.FutureRound(),                     // TODO(nkryuchkov): failure; need to substitute identifier in justifications in mapping
		roundchange.PastRound(),                       // TODO(nkryuchkov): failure; need to substitute identifier in justifications in mapping
		roundchange.F1SpeedupDifferentRounds(),        // TODO(nkryuchkov): failure; need to substitute identifier in justifications in mapping
		roundchange.DuplicateMsgQuorum(),              // TODO(nkryuchkov): failure; need to substitute identifier in justifications in mapping
		roundchange.DuplicateMsgPartialQuorum(),       // TODO(nkryuchkov): failure; need to substitute identifier in justifications in mapping
		roundchange.DuplicateMsgPrepared(),            // TODO(nkryuchkov): failure; need to substitute identifier in justifications in mapping
		roundchange.ImparsableRoundChangeData(),
	}

	result := make(map[string]struct{})
	for _, test := range testList {
		result[test.TestName()] = struct{}{}
	}

	return result
}

func TestQBFTMapping(t *testing.T) {
	path, _ := os.Getwd()
	fileName := "tests.json"
	filePath := path + "/" + fileName
	jsonTests, err := ioutil.ReadFile(filePath)
	if err != nil {
		resp, err := http.Get("https://raw.githubusercontent.com/bloxapp/ssv-spec/main/qbft/spectest/generate/tests.json")
		require.NoError(t, err)

		defer func() {
			require.NoError(t, resp.Body.Close())
		}()

		jsonTests, err = ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		require.NoError(t, ioutil.WriteFile(filePath, jsonTests, 0644))
	}

	untypedTests := map[string]interface{}{}
	if err := json.Unmarshal(jsonTests, &untypedTests); err != nil {
		panic(err.Error())
	}

	testMap := testsToRun() // TODO(nkryuchkov): remove

	tests := make(map[string]spectest.SpecTest)
	for name, test := range untypedTests {
		name, test := name, test

		testName := strings.Split(name, "_")[1]
		testType := strings.Split(name, "_")[0]

		if _, ok := testMap[testName]; !ok {
			continue
		}

		switch testType {
		case reflect.TypeOf(&spectests.MsgProcessingSpecTest{}).String():
			byts, err := json.Marshal(test)
			require.NoError(t, err)
			typedTest := &spectests.MsgProcessingSpecTest{}
			require.NoError(t, json.Unmarshal(byts, &typedTest))

			t.Run(typedTest.TestName(), func(t *testing.T) {
				runMsgProcessingSpecTest(t, typedTest)
			})
		case reflect.TypeOf(&spectests.MsgSpecTest{}).String():
			byts, err := json.Marshal(test)
			require.NoError(t, err)
			typedTest := &spectests.MsgSpecTest{}
			require.NoError(t, json.Unmarshal(byts, &typedTest))

			tests[testName] = typedTest
			t.Run(typedTest.TestName(), func(t *testing.T) {
				runMsgSpecTest(t, typedTest)
			})
		}
	}
}

func runMsgProcessingSpecTest(t *testing.T, test *spectests.MsgProcessingSpecTest) {
	ctx := context.TODO()
	logger := logex.Build(test.Name, zapcore.DebugLevel, nil)

	forkVersion := forksprotocol.GenesisForkVersion
	pi, _ := protocolp2p.GenPeerID()
	p2pNet := protocolp2p.NewMockNetwork(logger, pi, 10)
	beacon := validator.NewTestBeacon(t)

	var keysSet *testingutils.TestKeySet
	switch len(test.Pre.State.Share.Committee) {
	case 4:
		keysSet = testingutils.Testing4SharesSet()
	case 7:
		keysSet = testingutils.Testing7SharesSet()
	case 10:
		keysSet = testingutils.Testing10SharesSet()
	case 13:
		keysSet = testingutils.Testing13SharesSet()
	default:
		t.Error("unknown key set length")
	}

	db, err := storage.GetStorageFactory(basedb.Options{
		Type:   "badger-memory",
		Logger: logger,
		Ctx:    ctx,
	})
	require.NoError(t, err)

	qbftStorage := qbftstorage.NewQBFTStore(db, logger, spectypes.BNRoleAttester.String())

	share := &beaconprotocol.Share{
		NodeID:    1,
		PublicKey: keysSet.ValidatorPK,
		Committee: make(map[spectypes.OperatorID]*beaconprotocol.Node),
	}

	commonIdentifier := findCommonIdentifier(t, test.InputMessages)

	var mappedCommittee []*spectypes.Operator
	for i := range test.Pre.State.Share.Committee {
		operatorID := spectypes.OperatorID(i) + 1
		require.NoError(t, beacon.AddShare(keysSet.Shares[operatorID]))

		pk := keysSet.Shares[operatorID].GetPublicKey().Serialize()
		share.Committee[spectypes.OperatorID(i)+1] = &beaconprotocol.Node{
			IbftID: uint64(i) + 1,
			Pk:     pk,
		}

		share.OperatorIds = append(share.OperatorIds, uint64(i)+1)

		mappedCommittee = append(mappedCommittee, &spectypes.Operator{
			OperatorID: operatorID,
			PubKey:     pk,
		})
	}

	mappedShare := &spectypes.Share{
		OperatorID:      share.NodeID,
		ValidatorPubKey: share.PublicKey.Serialize(),
		SharePubKey:     share.Committee[share.NodeID].Pk,
		Committee:       mappedCommittee,
		Quorum:          keysSet.Threshold,
		PartialQuorum:   keysSet.PartialThreshold,
		DomainType:      spectypes.PrimusTestnet,
		Graffiti:        nil,
	}

	identifier := spectypes.NewMsgID(share.PublicKey.Serialize(), spectypes.BNRoleAttester)
	qbftInstance := newQbftInstance(logger, qbftStorage, p2pNet, beacon, share, mappedShare, forkVersion)
	qbftInstance.Init()
	qbftInstance.State().InputValue.Store(test.Pre.StartValue)
	qbftInstance.State().Round.Store(test.Pre.State.Round)
	qbftInstance.State().Height.Store(test.Pre.State.Height)
	//time.Sleep(time.Second * 3) // 3s round

	signatureMapping := make(map[string]signatureAndID)

	var lastErr error
	for _, msg := range test.InputMessages {
		signedMessage := substituteMessageData(t, msg, identifier, keysSet, signatureMapping)

		if _, err := qbftInstance.ProcessMsg(signedMessage); err != nil {
			lastErr = err
		}
	}

	//time.Sleep(time.Second * 3) // 3s round

	mappedInstance := new(specqbft.Instance)
	if qbftInstance != nil {
		preparedValue := qbftInstance.State().GetPreparedValue()
		if len(preparedValue) == 0 {
			preparedValue = nil
		}
		round := qbftInstance.State().GetRound()
		//if round == 0 {
		//	round = 1
		//}

		_, decidedErr := qbftInstance.CommittedAggregatedMsg()
		var decidedValue []byte
		if decidedErr == nil && len(test.InputMessages) != 0 {
			decidedValue = test.InputMessages[0].Message.Identifier
		}

		mappedInstance.State = &specqbft.State{
			Share:                           mappedShare,
			ID:                              test.Pre.State.ID,
			Round:                           round,
			Height:                          qbftInstance.State().GetHeight(),
			LastPreparedRound:               qbftInstance.State().GetPreparedRound(),
			LastPreparedValue:               preparedValue,
			ProposalAcceptedForCurrentRound: test.Pre.State.ProposalAcceptedForCurrentRound,
			Decided:                         decidedErr == nil,
			DecidedValue:                    decidedValue,
			ProposeContainer:                convertToSpecContainer(t, qbftInstance.Containers()[specqbft.ProposalMsgType], signatureMapping),
			PrepareContainer:                convertToSpecContainer(t, qbftInstance.Containers()[specqbft.PrepareMsgType], signatureMapping),
			CommitContainer:                 convertToSpecContainer(t, qbftInstance.Containers()[specqbft.CommitMsgType], signatureMapping),
			RoundChangeContainer:            convertToSpecContainer(t, qbftInstance.Containers()[specqbft.RoundChangeMsgType], signatureMapping),
		}

		allMessages := mappedInstance.State.ProposeContainer.AllMessaged()
		if len(allMessages) != 0 {
			mappedInstance.State.ProposalAcceptedForCurrentRound = allMessages[0]
		}

		mappedInstance.StartValue = qbftInstance.State().GetInputValue()
	}

	if len(test.ExpectedError) != 0 {
		require.EqualError(t, lastErr, test.ExpectedError)
	} else {
		require.NoError(t, lastErr)
	}

	mappedRoot, err := mappedInstance.State.GetRoot()
	require.NoError(t, err)
	require.Equal(t, test.PostRoot, hex.EncodeToString(mappedRoot))

	type BroadcastMessagesGetter interface {
		GetBroadcastMessages() []spectypes.SSVMessage
	}

	outputMessages := p2pNet.(BroadcastMessagesGetter).GetBroadcastMessages()

	require.Equal(t, len(test.OutputMessages), len(outputMessages))

	for i, outputMessage := range outputMessages {
		signedMessage := &specqbft.SignedMessage{}
		require.NoError(t, signedMessage.Decode(outputMessage.Data))

		signedMessage.Message.Identifier = commonIdentifier
		signedMessage.Signature = nil

		pk, err := share.OperatorSharePubKey()
		require.NoError(t, err)

		sig, err := beacon.SignIBFTMessage(signedMessage, pk.Serialize(), message.QBFTSigType)
		require.NoError(t, err)

		signedMessage.Signature = sig

		convertedMessage := convertSignedMessage(t, keysSet, signedMessage)
		require.Equal(t, test.OutputMessages[i], convertedMessage)
	}

	db.Close()
}

func substituteMessageData(t *testing.T, msg *specqbft.SignedMessage, identifier spectypes.MessageID, keysSet *testingutils.TestKeySet, signatureMapping map[string]signatureAndID) *specqbft.SignedMessage {
	origSignAndID := signatureAndID{
		Signature:  msg.Signature,
		Identifier: msg.Message.Identifier,
	}

	switch msg.Message.MsgType {
	case specqbft.ProposalMsgType:
		proposalData, err := msg.Message.GetProposalData()
		require.NoError(t, err)

		for i, innerMsg := range proposalData.RoundChangeJustification {
			proposalData.RoundChangeJustification[i] = substituteMessageData(t, innerMsg, identifier, keysSet, signatureMapping)
		}

		for i, innerMsg := range proposalData.PrepareJustification {
			proposalData.PrepareJustification[i] = substituteMessageData(t, innerMsg, identifier, keysSet, signatureMapping)
		}

		msg.Message.Data, err = proposalData.Encode()
		require.NoError(t, err)

	case specqbft.PrepareMsgType:
		prepareData, err := msg.Message.GetPrepareData()
		require.NoError(t, err)

		_ = prepareData // TODO(nkryuchkov): probably need to parse and substitute

	case specqbft.CommitMsgType:
		commitData, err := msg.Message.GetCommitData()
		require.NoError(t, err)

		_ = commitData // TODO(nkryuchkov): probably need to parse and substitute

	case specqbft.RoundChangeMsgType:
		roundChangeData, err := msg.Message.GetRoundChangeData()
		require.NoError(t, err)

		for i, innerMsg := range roundChangeData.RoundChangeJustification {
			roundChangeData.RoundChangeJustification[i] = substituteMessageData(t, innerMsg, identifier, keysSet, signatureMapping)
		}

		msg.Message.Data, err = roundChangeData.Encode()
		require.NoError(t, err)
	}

	modifiedMsg := &specqbft.SignedMessage{
		Signature: msg.Signature,
		Signers:   msg.Signers,
		Message: &specqbft.Message{
			MsgType:    msg.Message.MsgType,
			Height:     msg.Message.Height,
			Round:      msg.Message.Round,
			Identifier: identifier[:],
			//Identifier: msg.Message.Identifier,
			Data: msg.Message.Data, // TODO(nkryuchkov): substitute identifier in justifications
		},
	}

	domain := spectypes.PrimusTestnet
	sigType := spectypes.QBFTSignatureType
	r, err := spectypes.ComputeSigningRoot(modifiedMsg, spectypes.ComputeSignatureDomain(domain, sigType))
	require.NoError(t, err)

	var aggSig *bls.Sign
	for _, signer := range modifiedMsg.Signers {
		sig := keysSet.Shares[signer].SignByte(r)
		if aggSig == nil {
			aggSig = sig
		} else {
			aggSig.Add(sig)
		}
	}

	serializedSig := aggSig.Serialize()

	signatureMapping[hex.EncodeToString(serializedSig)] = origSignAndID
	signedMessage := convertSignedMessage(t, keysSet, modifiedMsg)
	signedMessage.Signature = serializedSig
	return signedMessage
}

func findCommonIdentifier(t *testing.T, messages []*specqbft.SignedMessage) []byte {
	seen := make(map[string]struct{})
	for _, msg := range messages {
		if len(msg.Message.Identifier) == 0 {
			continue
		}
		if _, ok := seen[string(msg.Message.Identifier)]; !ok {
			seen[string(msg.Message.Identifier)] = struct{}{}
		}
	}

	if len(seen) != 1 {
		t.Fatal("test implementation doesn't support different identifiers")
	}

	return messages[0].Message.Identifier
}

func runMsgSpecTest(t *testing.T, test *spectests.MsgSpecTest) {
	var lastErr error

	keysSet := testingutils.Testing4SharesSet()

	for i, messageBytes := range test.EncodedMessages {
		m := &specqbft.SignedMessage{}
		if err := m.Decode(messageBytes); err != nil {
			lastErr = err
		}

		if len(test.ExpectedRoots) > 0 {
			r, err := convertSignedMessage(t, keysSet, m).GetRoot()
			if err != nil {
				lastErr = err
			}

			if !bytes.Equal(test.ExpectedRoots[i], r) {
				t.Fail()
			}
		}
	}

	for i, msg := range test.Messages {
		if err := msg.Validate(); err != nil {
			lastErr = err
		}

		switch msg.Message.MsgType {
		case specqbft.RoundChangeMsgType:
			rc := specqbft.RoundChangeData{}
			if err := rc.Decode(msg.Message.Data); err != nil {
				lastErr = err
			}
			if err := validateRoundChangeData(specToRoundChangeData(t, keysSet, rc)); err != nil {
				lastErr = err
			}
		}

		if len(test.Messages) > 0 {
			r1, err := msg.Encode()
			if err != nil {
				lastErr = err
			}

			r2, err := test.Messages[i].Encode()
			if err != nil {
				lastErr = err
			}
			if !bytes.Equal(r2, r1) {
				t.Fail()
			}
		}
	}

	if len(test.ExpectedError) != 0 {
		require.EqualError(t, lastErr, test.ExpectedError)
	} else {
		require.NoError(t, lastErr)
	}
}

func validateRoundChangeData(d specqbft.RoundChangeData) error {
	if isRoundChangeDataPrepared(d) {
		if len(d.PreparedValue) == 0 {
			return errors.New("round change prepared value invalid")
		}
		if len(d.RoundChangeJustification) == 0 {
			return errors.New("round change justification invalid")
		}
		// TODO - should next proposal data be equal to prepared value?
	}

	if len(d.NextProposalData) == 0 {
		return errors.New("round change next value invalid")
	}
	return nil
}

func isRoundChangeDataPrepared(d specqbft.RoundChangeData) bool {
	return d.PreparedRound != 0 || len(d.PreparedValue) != 0
}

type signatureAndID struct {
	Signature  []byte
	Identifier []byte
}

type roundRobinLeaderSelector struct {
	state       *qbftprotocol.State
	mappedShare *spectypes.Share
}

func (m roundRobinLeaderSelector) Calculate(round uint64) uint64 {
	specState := &specqbft.State{
		Share:  m.mappedShare,
		Height: m.state.GetHeight(),
	}
	// RoundRobinProposer returns OperatorID which starts with 1.
	// As the result will be used to index OperatorIds which starts from 0,
	// the result of Calculate should be decremented.
	return uint64(specqbft.RoundRobinProposer(specState, specqbft.Round(round))) - 1
}

func newQbftInstance(logger *zap.Logger, qbftStorage qbftstorage.QBFTStore, net protocolp2p.MockNetwork, beacon *validator.TestBeacon, share *beaconprotocol.Share, mappedShare *spectypes.Share, forkVersion forksprotocol.ForkVersion) instance.Instancer {
	const height = 0
	fork := forksfactory.NewFork(forkVersion)
	identifier := spectypes.NewMsgID(share.PublicKey.Serialize(), spectypes.BNRoleAttester)
	newInstance := instance.NewInstance(&instance.Options{
		Logger:           logger,
		ValidatorShare:   share,
		Network:          net,
		Config:           qbftprotocol.DefaultConsensusParams(),
		Identifier:       identifier,
		Height:           height,
		RequireMinPeers:  false,
		Fork:             fork.InstanceFork(),
		Signer:           beacon,
		ChangeRoundStore: qbftStorage,
	})
	//newInstance.(*instance.Instance).LeaderSelector = roundRobinLeaderSelector{newInstance.State(), mappedShare}
	// TODO(nkryuchkov): replace when ready
	newInstance.(*instance.Instance).LeaderSelector = &constant.Constant{LeaderIndex: 0}
	return newInstance
}

func convertSignedMessage(t *testing.T, keysSet *testingutils.TestKeySet, msg *specqbft.SignedMessage) *specqbft.SignedMessage {
	signers := append([]spectypes.OperatorID{}, msg.GetSigners()...)

	domain := spectypes.PrimusTestnet
	sigType := spectypes.QBFTSignatureType
	r, err := spectypes.ComputeSigningRoot(msg, spectypes.ComputeSignatureDomain(domain, sigType))
	require.NoError(t, err)

	var aggSig *bls.Sign
	for _, signer := range msg.Signers {
		sig := keysSet.Shares[signer].SignByte(r)
		if aggSig == nil {
			aggSig = sig
		} else {
			aggSig.Add(sig)
		}
	}

	return &specqbft.SignedMessage{
		Signature: aggSig.Serialize(),
		//Signature: []byte(msg.Signature),
		Signers: signers,
		Message: &specqbft.Message{
			MsgType:    msg.Message.MsgType,
			Height:     msg.Message.Height,
			Round:      msg.Message.Round,
			Identifier: msg.Message.Identifier,
			//Identifier: msg.Message.Identifier,
			Data: msg.Message.Data,
		},
	}
}

func specToRoundChangeData(t *testing.T, keysSet *testingutils.TestKeySet, spec specqbft.RoundChangeData) specqbft.RoundChangeData {
	rcj := make([]*specqbft.SignedMessage, 0, len(spec.RoundChangeJustification))
	for _, v := range spec.RoundChangeJustification {
		rcj = append(rcj, convertSignedMessage(t, keysSet, v))
	}

	return specqbft.RoundChangeData{
		PreparedValue:            spec.PreparedValue,
		PreparedRound:            spec.PreparedRound,
		NextProposalData:         spec.NextProposalData,
		RoundChangeJustification: rcj,
	}
}

func convertToSpecContainer(t *testing.T, container msgcont.MessageContainer, signatureToSpecSignatureAndID map[string]signatureAndID) *specqbft.MsgContainer {
	c := specqbft.NewMsgContainer()
	container.AllMessaged(func(round specqbft.Round, msg *specqbft.SignedMessage) {
		signers := append([]spectypes.OperatorID{}, msg.GetSigners()...)

		originalSignatureAndID := signatureToSpecSignatureAndID[hex.EncodeToString(msg.Signature)]

		// TODO need to use one of the message type (spec/protocol)
		ok, err := c.AddIfDoesntExist(&specqbft.SignedMessage{
			Signature: originalSignatureAndID.Signature,
			Signers:   signers,
			Message: &specqbft.Message{
				MsgType:    msg.Message.MsgType,
				Height:     msg.Message.Height,
				Round:      msg.Message.Round,
				Identifier: originalSignatureAndID.Identifier,
				Data:       msg.Message.Data,
			},
		})
		require.NoError(t, err)
		require.True(t, ok)
	})
	return c
}
