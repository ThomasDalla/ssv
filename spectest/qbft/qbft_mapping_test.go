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
		spectests.ThirteenOperators(), // TODO(nkryuchkov): failure

		proposal.HappyFlow(),
		proposal.NotPreparedPreviouslyJustification(), // TODO(nkryuchkov): failure
		proposal.PreparedPreviouslyJustification(),    // TODO(nkryuchkov): failure
		proposal.DifferentJustifications(),            // TODO(nkryuchkov): failure
		proposal.JustificationsNotHeighest(),          // TODO(nkryuchkov): failure
		proposal.JustificationsValueNotJustified(),    // TODO(nkryuchkov): failure
		proposal.DuplicateMsg(),
		proposal.FirstRoundJustification(),
		proposal.FutureRoundNoAcceptedProposal(),                  // TODO(nkryuchkov): failure
		proposal.FutureRoundAcceptedProposal(),                    // TODO(nkryuchkov): failure
		proposal.PastRound(),                                      // TODO(nkryuchkov): failure
		proposal.ImparsableProposalData(),                         // TODO(nkryuchkov): failure
		proposal.InvalidRoundChangeJustificationPrepared(),        // TODO(nkryuchkov): failure
		proposal.InvalidRoundChangeJustification(),                // TODO(nkryuchkov): failure
		proposal.PreparedPreviouslyNoRCJustificationQuorum(),      // TODO(nkryuchkov): failure
		proposal.NoRCJustification(),                              // TODO(nkryuchkov): failure
		proposal.PreparedPreviouslyNoPrepareJustificationQuorum(), // TODO(nkryuchkov): failure
		proposal.PreparedPreviouslyDuplicatePrepareMsg(),          // TODO(nkryuchkov): failure
		proposal.PreparedPreviouslyDuplicateRCMsg(),               // TODO(nkryuchkov): failure
		proposal.DuplicateRCMsg(),                                 // TODO(nkryuchkov): failure
		proposal.InvalidPrepareJustificationValue(),               // TODO(nkryuchkov): failure
		proposal.InvalidPrepareJustificationRound(),               // TODO(nkryuchkov): failure
		proposal.InvalidProposalData(),                            // TODO(nkryuchkov): failure
		proposal.InvalidValueCheck(),                              // TODO(nkryuchkov): failure
		proposal.MultiSigner(),                                    // TODO(nkryuchkov): failure
		proposal.PostDecided(),
		proposal.PostPrepared(),           // TODO(nkryuchkov): failure
		proposal.SecondProposalForRound(), // TODO(nkryuchkov): failure
		proposal.WrongHeight(),            // TODO(nkryuchkov): failure
		proposal.WrongProposer(),          // TODO(nkryuchkov): failure
		proposal.WrongSignature(),         // TODO(nkryuchkov): failure

		commit.CurrentRound(),
		commit.FutureRound(),            // TODO(nkryuchkov): failure
		commit.PastRound(),              // TODO(nkryuchkov): failure
		commit.DuplicateMsg(),           // TODO(nkryuchkov): failure
		commit.HappyFlow(),              // TODO(nkryuchkov): failure
		commit.InvalidCommitData(),      // TODO(nkryuchkov): failure
		commit.PostDecided(),            // TODO(nkryuchkov): failure
		commit.WrongData1(),             // TODO(nkryuchkov): failure
		commit.WrongData2(),             // TODO(nkryuchkov): failure
		commit.MultiSignerWithOverlap(), // TODO(nkryuchkov): failure
		commit.MultiSignerNoOverlap(),   // TODO(nkryuchkov): failure
		commit.Decided(),                // TODO(nkryuchkov): failure
		commit.NoPrevAcceptedProposal(), // TODO(nkryuchkov): failure
		commit.WrongHeight(),            // TODO(nkryuchkov): failure
		commit.ImparsableCommitData(),   // TODO(nkryuchkov): failure
		commit.WrongSignature(),         // TODO(nkryuchkov): failure

		roundchange.HappyFlow(),          // TODO(nkryuchkov): failure
		roundchange.PreviouslyPrepared(), // TODO(nkryuchkov): failure
		roundchange.F1Speedup(),          // TODO(nkryuchkov): failure
		roundchange.F1SpeedupPrepared(),  // TODO(nkryuchkov): failure
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

func mapErrors(err string) string {
	switch err {
	// TODO(nkryuchkov): ensure that cases below are not bugs
	case "proposal invalid: proposal not justified: prepares has no quorum",
		"proposal invalid: proposal not justified: change round has not quorum",
		"proposal invalid: proposal not justified: change round msg not valid: basic round Change validation failed: round change msg signature invalid: failed to verify signature",
		"proposal invalid: proposal not justified: proposed data doesn't match highest prepared",
		"proposal invalid: proposal not justified: signed prepare not valid",
		"proposal invalid: proposal is not valid with current state",
		"invalid prepare msg: msg round wrong":
		return "pre-prepare message sender (id 1) is not the round's leader (expected 2)"
	case "invalid signed message: message data is invalid":
		return "could not decode commit data from message: unexpected end of JSON input"
	case "invalid prepare msg: msg Height wrong":
		return "invalid message sequence number: expected: 0, actual: 2"
	case "commit msg invalid: could not get msg commit data: could not decode commit data from message: invalid character '\\x01' looking for beginning of value":
		return "could not decode commit data from message: invalid character '\\x01' looking for beginning of value"
	case "proposal invalid: could not get proposal data: could not decode proposal data from message: invalid character '\\x01' looking for beginning of value":
		return "could not decode proposal data from message: invalid character '\\x01' looking for beginning of value"
	case "invalid prepare msg: could not get prepare data: could not decode prepare data from message: invalid character '\\x01' looking for beginning of value":
		return "could not decode prepare data from message: invalid character '\\x01' looking for beginning of value"
	case "commit msg invalid: commit Height is wrong":
		return "invalid message sequence number: expected: 0, actual: 10"
	}

	return err
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
	qbftInstance := newQbftInstance(t, logger, qbftStorage, p2pNet, beacon, share, forkVersion)
	qbftInstance.Init()
	qbftInstance.State().InputValue.Store(test.Pre.StartValue)
	qbftInstance.State().Round.Store(specqbft.Round(1)) // start from 1
	//time.Sleep(time.Second * 3) // 3s round

	signatureMapping := make(map[string]signatureAndID)

	var lastErr error
	for _, msg := range test.InputMessages {
		origSignAndID := signatureAndID{
			Signature:  msg.Signature,
			Identifier: msg.Message.Identifier,
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
				Data: msg.Message.Data,
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
		// TODO(nkryuchkov): review mapErrors function
		//require.EqualError(t, lastErr, mapErrors(test.ExpectedError))
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

type mockLeaderSelector struct {
	committeeSize uint64
}

func (m mockLeaderSelector) Calculate(round uint64) uint64 {
	return (round - 1) % m.committeeSize
}

func newQbftInstance(t *testing.T, logger *zap.Logger, qbftStorage qbftstorage.QBFTStore, net protocolp2p.MockNetwork, beacon *validator.TestBeacon, share *beaconprotocol.Share, forkVersion forksprotocol.ForkVersion) instance.Instancer {
	const height = 0
	fork := forksfactory.NewFork(forkVersion)
	identifier := spectypes.NewMsgID(share.PublicKey.Serialize(), spectypes.BNRoleAttester)

	return instance.NewInstance(&instance.Options{
		Logger:           logger,
		ValidatorShare:   share,
		Network:          net,
		LeaderSelector:   mockLeaderSelector{uint64(share.CommitteeSize())},
		Config:           qbftprotocol.DefaultConsensusParams(),
		Identifier:       identifier,
		Height:           height,
		RequireMinPeers:  false,
		Fork:             fork.InstanceFork(),
		Signer:           beacon,
		ChangeRoundStore: qbftStorage,
	})
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
