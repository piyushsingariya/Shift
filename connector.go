package gear5

import (
	"fmt"
	"os"

	"github.com/gear5sh/gear5/logger"
	protocol "github.com/gear5sh/gear5/protocol"
	"github.com/gear5sh/gear5/safego"
	"github.com/spf13/cobra"
)

var (
	globalDriver  protocol.Driver
	globalAdapter protocol.Adapter
)

func RegisterDriver(driver protocol.Driver) {
	defer safego.Recovery(true)

	if globalAdapter != nil {
		logger.Fatal(fmt.Errorf("adapter already registered: %s", globalAdapter.Type()))
	}

	globalDriver = driver

	// Execute the root command
	err := protocol.CreateRootCommand(true, driver).Execute()
	if err != nil {
		logger.Fatal(err)
	}

	os.Exit(0)
}

func RegisterAdapter(adapter protocol.Adapter) (*cobra.Command, error) {
	if globalDriver != nil {
		return nil, fmt.Errorf("driver alraedy registered: %s", globalDriver.Type())
	}

	globalAdapter = adapter

	return protocol.CreateRootCommand(false, adapter), nil
}
