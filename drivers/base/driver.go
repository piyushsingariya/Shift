package base

import (
	"github.com/gear5sh/gear5/protocol"
	"github.com/gear5sh/gear5/types"
	"github.com/gear5sh/gear5/typeutils"
)

type Driver struct {
	SourceStreams map[string]*types.Stream // locally cached streams; It contains all streams
	GroupRead     bool                     // Used in CDC mode
}

func NewBase() *Driver {
	return &Driver{
		SourceStreams: make(map[string]*types.Stream),
	}
}

func (d *Driver) BulkRead() bool {
	return d.GroupRead
}

func (d *Driver) UpdateState(stream protocol.Stream, data types.RecordData) error {
	datatype, err := stream.Schema().GetType(stream.Cursor())
	if err != nil {
		return err
	}

	if cursorVal, found := data[stream.Cursor()]; found && cursorVal != nil {
		// compare with current state
		if stream.GetState() != nil {
			state, err := typeutils.MaximumOnDataType(datatype, stream.GetState(), cursorVal)
			if err != nil {
				return err
			}

			stream.SetState(state)
		} else {
			// directly update
			stream.SetState(cursorVal)
		}
	}

	return nil
}
