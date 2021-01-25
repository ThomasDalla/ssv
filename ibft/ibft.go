package ibft

import (
	"encoding/hex"
	"time"

	"github.com/sirupsen/logrus"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"go.uber.org/zap"

	"github.com/bloxapp/ssv/ibft/types"
)

func Place() {
	att := eth.Attestation{}
	att.String()

	blk := eth.BeaconBlock{}
	blk.String()
}

type iBFTInstance struct {
	me               *types.Node
	state            *types.State
	network          types.Networker
	implementation   types.Implementor
	params           *types.InstanceParams
	roundChangeTimer *time.Timer
	log              *logrus.Entry

	// messages
	prePrepareMessages  *types.MessagesContainer
	prepareMessages     *types.MessagesContainer
	commitMessages      *types.MessagesContainer
	roundChangeMessages *types.MessagesContainer

	// flags
	started     bool
	decided     chan bool
	changeRound chan bool
}

// New is the constructor of iBFTInstance
func New(
	me *types.Node,
	network types.Networker,
	implementation types.Implementor,
	params *types.InstanceParams,
) *iBFTInstance {
	return &iBFTInstance{
		me:             me,
		state:          &types.State{},
		network:        network,
		implementation: implementation,
		params:         params,
		log: logrus.WithFields(logrus.Fields{
			"node_id": me.IbftId,
		}),

		prePrepareMessages:  types.NewMessagesContainer(),
		prepareMessages:     types.NewMessagesContainer(),
		commitMessages:      types.NewMessagesContainer(),
		roundChangeMessages: types.NewMessagesContainer(),

		started:     false,
		decided:     make(chan bool),
		changeRound: make(chan bool),
	}
}

/**
### Algorithm 1 IBFT pseudocode for process pi: constants, state variables, and ancillary procedures
 procedure Start(λ, value)
 	λi ← λ
 	ri ← 1
 	pri ← ⊥
 	pvi ← ⊥
 	inputV aluei ← value
 	if leader(hi, ri) = pi then
 		broadcast ⟨PRE-PREPARE, λi, ri, inputV aluei⟩ message
 		set timeri to running and expire after t(ri)
*/
func (i *iBFTInstance) Start(lambda []byte, inputValue []byte) error {
	i.log.Info("Node is starting iBFT instance", zap.String("instance", hex.EncodeToString(lambda)))
	i.state.Round = 1
	i.state.Lambda = lambda
	i.state.InputValue = inputValue
	i.state.Stage = types.RoundState_Preprepare

	if i.implementation.IsLeader(i.state) {
		i.log.Info("Node is leader for round 1")
		msg := &types.Message{
			Type:   types.RoundState_Preprepare,
			Round:  i.state.Round,
			Lambda: i.state.Lambda,
			Value:  i.state.InputValue,
		}
		if err := i.network.Broadcast(msg); err != nil {
			return err
		}
	}
	i.started = true
	i.triggerRoundChangeOnTimer()
	return nil
}

// Committed returns a channel which indicates when this instance of iBFT decided an input value.
func (i *iBFTInstance) Committed() chan bool {
	return i.decided
}

// StartEventLoopAndMessagePipeline - the iBFT instance is message driven with an 'upon' logic.
// each message type has it's own pipeline of checks and actions, called by the networker implementation.
// Internal chan monitor if the instance reached decision or if a round change is required.
func (i *iBFTInstance) StartEventLoopAndMessagePipeline() {
	id := string(i.me.IbftId)
	i.network.SetMessagePipeline(id, types.RoundState_Preprepare, []types.PipelineFunc{
		MsgTypeCheck(types.RoundState_Preprepare),
		i.ValidateLambda(),
		i.ValidateRound(),
		i.AuthMsg(),
		i.validatePrePrepareMsg(),
		i.uponPrePrepareMsg(),
	})
	i.network.SetMessagePipeline(id, types.RoundState_Prepare, []types.PipelineFunc{
		MsgTypeCheck(types.RoundState_Prepare),
		i.ValidateLambda(),
		i.ValidateRound(),
		i.AuthMsg(),
		i.validatePrepareMsg(),
		i.uponPrepareMsg(),
	})
	i.network.SetMessagePipeline(id, types.RoundState_Commit, []types.PipelineFunc{
		MsgTypeCheck(types.RoundState_Commit),
		i.ValidateLambda(),
		i.ValidateRound(),
		i.AuthMsg(),
		i.validateCommitMsg(),
		i.uponCommitMsg(),
	})
	i.network.SetMessagePipeline(id, types.RoundState_ChangeRound, []types.PipelineFunc{
		MsgTypeCheck(types.RoundState_ChangeRound),
		i.ValidateLambda(),
		i.ValidateRound(), // TODO - should we validate round?
		i.AuthMsg(),
		i.validateChangeRoundMsg(),
		i.uponChangeRoundMsg(),
	})

	go func() {
		for {
			select {
			// When decided is triggered the iBFT instance has concluded and should stop.
			case <-i.decided:
				i.log.Info("iBFT instance decided, exiting..")
				//close(msgChan) // TODO - find a safe way to close connection
				//return
			// Change round is called when no Quorum was achieved within a time duration
			case <-i.changeRound:
				go i.uponChangeRoundTrigger()
			}
		}
	}()
}

/**
"Timer:
	In addition to the state variables, each correct process pi also maintains a timer represented by timeri,
	which is used to trigger a round change when the algorithm does not sufficiently progress.
	The timer can be in one of two states: running or expired.
	When set to running, it is also set a time t(ri), which is an exponential function of the round number ri, after which the state changes to expired."
*/
func (i *iBFTInstance) triggerRoundChangeOnTimer() {
	// make sure previous timer is stopped
	i.stopRoundChangeTimer()

	// stat new timer
	roundTimeout := uint64(i.params.ConsensusParams.RoundChangeDuration) * mathutil.PowerOf2(i.state.Round)
	i.roundChangeTimer = time.NewTimer(time.Duration(roundTimeout))
	go func() {
		<-i.roundChangeTimer.C
		i.changeRound <- true
		i.stopRoundChangeTimer()
	}()
}

func (i *iBFTInstance) stopRoundChangeTimer() {
	if i.roundChangeTimer != nil {
		i.roundChangeTimer.Stop()
		i.roundChangeTimer = nil
	}
}
