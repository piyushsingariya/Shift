package reader

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/s3"
	shiftmodels "github.com/piyushsingariya/shift/models"
)

var FileTypes = []string{"parquet"}

type Reader interface {
	GetSchema() (map[string]*shiftmodels.Property, error)
	Read() ([]map[string]any, error)
	HasNext() bool
}

func Init(s3 *s3.S3, _type, bucket, file string, batchSize int64) (Reader, error) {
	switch strings.ToLower(_type) {
	case "parquet":
		return InitParquet(s3, bucket, file, batchSize)
	default:
		return nil, fmt.Errorf("reader not available to file format: %s", _type)
	}
}
