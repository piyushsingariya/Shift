package protocol

import (
	"fmt"
	"sync"

	"github.com/piyushsingariya/shift/drivers/base"
	"github.com/piyushsingariya/shift/logger"
	"github.com/piyushsingariya/shift/safego"
	"github.com/piyushsingariya/shift/types"
	"github.com/piyushsingariya/shift/typing"
	"github.com/piyushsingariya/shift/utils"
	"github.com/spf13/cobra"
)

// DiscoverCmd represents the read command
var DiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Shift discover command",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return utils.CheckIfFilesExists(config)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		connector, not := rawConnector.(Driver)
		if !not {
			logger.Fatal(fmt.Errorf("expected type to be: Connector, found %T", connector))
		}

		err := connector.Setup(utils.ReadFile(config), base.NewDriver(nil, nil, batchSize))
		if err != nil {
			logger.Fatal(err)
		}

		err = connector.Check()
		if err != nil {
			logger.Fatal(err)
		}

		streams, err := connector.Discover()
		if err != nil {
			logger.Fatal(err)
		}

		if len(streams) == 0 {
			logger.Fatal("no streams found in connector")
		}

		recordsPerStream := 100
		group := sync.WaitGroup{}
		for _, wrappedStream := range wrapForSchemaDiscovery(streams) {
			stream := wrappedStream
			group.Add(1)

			go func() {
				objects := []types.RecordData{}
				channel := make(chan types.Record, recordsPerStream)
				count := 0
				go func() {
					err := connector.Read(stream, channel)
					if err != nil {
						logger.Fatalf("Error occurred while reading records from [%s]: %s", stream.Name(), err)
					}

					// close channel incase records are less than recordsPerStream
					safego.Close(channel)
				}()

				for record := range channel {
					count++
					objects = append(objects, record.Data)
					if count >= recordsPerStream {
						safego.Close(channel)
					}
				}

				properties, err := typing.Resolve(objects...)
				if err != nil {
					logger.Fatal(err)
				}

				stream.Stream.JSONSchema = &types.Schema{
					Properties: properties,
				}

				group.Done()
			}()
		}

		group.Wait()
		logger.LogCatalog(streams)
		return nil
	},
}

func wrapForSchemaDiscovery(streams []*types.Stream) []*types.WrappedStream {
	wrappedStreams := []*types.WrappedStream{}

	for _, stream := range streams {
		// only adding streams for which json schema needs to be discovered
		if stream.JSONSchema == nil {
			wrappedStreams = append(wrappedStreams, &types.WrappedStream{
				SyncMode: types.FullRefresh,
				Stream:   stream,
			})
		}
	}

	return wrappedStreams
}
