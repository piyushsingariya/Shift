package base

import (
	"time"

	"github.com/gear5sh/gear5/logger"
	"github.com/gear5sh/gear5/types"
)

type basestream interface {
	Name() string
	Namespace() string
}

func RetryOnFailure(attempts int, sleep *time.Duration, f func() error) (err error) {
	for i := 0; i < attempts; i++ {
		if err = f(); err == nil {
			return nil
		}

		logger.Infof("Retrying after %v...", sleep)
		time.Sleep(*sleep)
	}

	return err
}

func ReformatRecord(stream basestream, record map[string]any) types.Record {
	return types.Record{
		Stream:    stream.Name(),
		Namespace: stream.Namespace(),
		Data:      record,
	}
}
