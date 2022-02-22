package qbft

import (
	"bytes"
	"fmt"

	"github.com/bloxapp/ssv/docs/spec/types"
	"github.com/bloxapp/ssv/utils/threadsafe"
	"github.com/pkg/errors"
)

// HistoricalInstanceCapacity represents the upper bound of instances a controller can process messages for as messages are not
// guaranteed to arrive in a timely fashion, we physically limit how far back the controller will process messages for
const HistoricalInstanceCapacity int = 5

type instances [HistoricalInstanceCapacity]IInstance

func (i instances) FindInstance(height uint64) IInstance {
	for _, inst := range i {
		if inst != nil {
			if inst.GetHeight() == height {
				return inst
			}
		}
	}
	return nil
}

type IController interface {
	// StartNewInstance will start a new QBFT instance, if can't will return error
	StartNewInstance(value []byte) error
	// ProcessMsg processes a new msg, returns true if decided, non nil byte slice if decided (decided value) and error
	// decided returns just once per instance as true, following messages (for example additional commit msgs) will not return decided true
	ProcessMsg(msg *SignedMessage) (bool, []byte, error)
	// InstanceForHeight returns an instance for a specific height, nil if not found
	InstanceForHeight(height uint64) IInstance
	// GetHeight returns the current running instance height or, if not started, the last decided height
	GetHeight() uint64
	// GetIdentifier returns QBFT identifier, used to identify messages
	GetIdentifier() []byte
}

// Controller is a QBFT coordinator responsible for starting and following the entire life cycle of multiple QBFT instances
type Controller struct {
	identifier []byte
	height     *threadsafe.SafeUint64 // incremental height for instances
	// storedInstances stores the last HistoricalInstanceCapacity in an array for message processing purposes.
	storedInstances instances
	signer          types.SSVSigner
	valueCheck      proposedValueCheck
	storage         Storage
	network         Network
}

// StartNewInstance will start a new QBFT instance, if can't will return error
func (c *Controller) StartNewInstance(value []byte) error {
	if err := c.canStartInstance(value); err != nil {
		return errors.Wrap(err, "can't start new QBFT instance")
	}

	c.bumpHeight()
	newInstance := c.addAndStoreNewInstance()
	newInstance.Start(value, c.GetHeight())

	return nil
}

// ProcessMsg processes a new msg, returns true if decided, non nil byte slice if decided (decided value) and error
// decided returns just once per instance as true, following messages (for example additional commit msgs) will not return decided true
func (c *Controller) ProcessMsg(msg *SignedMessage) (bool, []byte, error) {
	if !bytes.Equal(c.GetIdentifier(), msg.Message.Identifier) {
		return false, nil, errors.New(fmt.Sprintf("message doesn't belong to identifier %x", c.GetIdentifier()))
	}

	inst := c.InstanceForHeight(msg.Message.Height)
	if inst == nil {
		return false, nil, errors.New(fmt.Sprintf("instance for height %d,  identifier %x not found", msg.Message.Height, c.GetIdentifier()))
	}

	prevDecided, _ := inst.IsDecided()
	decided, decidedValue, aggregatedCommit, err := inst.ProcessMsg(msg)
	if err != nil {
		return false, nil, errors.Wrap(err, "could not process msg")
	}

	// save the highest decided
	if decided && inst.GetHeight() == c.GetHeight() { // It's the highest instance
		if err := c.storage.SaveHighestDecided(aggregatedCommit); err != nil {
			// LOG
		}
	}

	// if previously decided we do not return decided true again
	if prevDecided {
		return false, nil, err
	}

	// Broadcast decided msg
	if err := c.network.BroadcastDecided(aggregatedCommit); err != nil {
		//TODO We do not return error here, just Log broadcasting error.
		return decided, decidedValue, nil
	}

	return decided, decidedValue, nil
}

func (c *Controller) InstanceForHeight(height uint64) IInstance {
	return c.storedInstances.FindInstance(height)
}

// GetHeight returns the current running instance height or, if not started, the last decided height
func (c *Controller) GetHeight() uint64 {
	return c.height.Get()
}

func (c *Controller) bumpHeight() {
	c.height.Set(c.height.Get() + 1)
}

// GetIdentifier returns QBFT identifier, used to identify messages
func (c *Controller) GetIdentifier() []byte {
	return c.identifier
}

// addAndStoreNewInstance returns creates a new QBFT instance, stores it in an array and returns it
func (c *Controller) addAndStoreNewInstance() Instance {
	panic("implement")
}

func (c *Controller) canStartInstance(value []byte) error {
	// check prev instance
	inst := c.storedInstances.FindInstance(c.GetHeight())
	if inst == nil {
		return errors.New("could not find previous instance")
	}
	if decided, _ := inst.IsDecided(); !decided {
		return errors.New("previous instance hasn't decided")
	}

	// check value
	if err := c.valueCheck(value); err != nil {
		return errors.Wrap(err, "value invalid")
	}
	panic("implement")
}
