package scenarios

import (
	"sync"

	"go.uber.org/zap"

	"github.com/bloxapp/ssv/automation/qbft/runner"
)

var scenarios = &sync.Map{}

// NewScenario is a factory function to get or create scenarios
func NewScenario(name string, logger *zap.Logger) runner.Scenario {
	var s runner.Scenario
	raw, ok := scenarios.Load(name)
	if !ok {
		switch name {
		case ChangeRoundSpeedupScenario:
			s = newChangeRoundSpeedupScenario(logger)
		case F1MultiRoundScenario:
			s = newF1MultiRoundScenario(logger)
		case F1SpeedupScenario:
			s = newF1SpeedupScenario(logger)
		case FarFutureSyncScenario:
			s = newFarFutureSyncScenario(logger)
		case RegularScenario:
			s = newRegularScenario(logger)
		case SyncFailoverScenario:
			s = newSyncFailoverScenario(logger)
		case FullNodeScenario:
			s = newFullNodeScenario(logger)
		default:
			logger.Panic("could not find scenario")
		}
		if s != nil {
			scenarios.Store(s.Name(), s)
			return s
		}
		return nil
	}
	return raw.(runner.Scenario)
}
