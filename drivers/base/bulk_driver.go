package base

import (
	"github.com/gear5sh/gear5/protocol"
	"github.com/gear5sh/gear5/types"
	"github.com/gear5sh/gear5/utils"
)

// Pass dest with all fields initialized to handle nil state case
func ManageGlobalState[T any](state *types.State, dest *T, driver protocol.BulkDriver) error {
	state.Type = driver.StateType()

	if state.Global != nil {
		err := utils.Unmarshal(state.Global, dest)
		if err != nil {
			return err
		}
	}

	// set Global State reference
	state.Global = dest

	return nil
}
