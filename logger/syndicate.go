package logger

import (
	"encoding/json"
	"os"

	"github.com/piyushsingariya/syndicate/constants"
	"github.com/piyushsingariya/syndicate/models"
)

func LogSpec(spec map[string]interface{}) {
	message := models.Message{}
	message.Spec = spec
	message.Type = constants.SpecType

	Info("logging spec")
	json.NewEncoder(os.Stdout).Encode(message)
}

func LogConnectionStatus(err error) {
	message := models.Message{}
	message.Type = constants.ConnectionStatusType
	message.ConnectionStatus = &models.StatusRow{}
	if err != nil {
		message.ConnectionStatus.Message = err.Error()
		message.ConnectionStatus.Status = constants.ConnectionFailed
	} else {
		message.ConnectionStatus.Status = constants.ConnectionSucceed
	}

	json.NewEncoder(os.Stdout).Encode(message)

	os.Exit(1)
}