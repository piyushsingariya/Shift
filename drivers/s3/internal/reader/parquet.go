package reader

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"sync/atomic"
	"time"

	_ "github.com/akrennmair/parquet-go-block-compressors/lz4raw" // registers the LZ4 block compressor with the LZ4_RAW compression type with parquet-go
	_ "github.com/akrennmair/parquet-go-brotli"                   // registers the Brotli block compressor with parquet-go

	_ "github.com/akrennmair/parquet-go-lzo"  // registers the LZO block compressor with parquet-go
	_ "github.com/akrennmair/parquet-go-zstd" // registers the Zstd block compressor with parquet-go
	"github.com/gear5sh/gear5/logger"
	"github.com/gear5sh/gear5/types"
	"github.com/gear5sh/gear5/typeutils"
	"github.com/gear5sh/gear5/utils"

	"github.com/aws/aws-sdk-go/service/s3"
	goparquet "github.com/fraugster/parquet-go"
	s3parquet "github.com/xitongsys/parquet-go-source/s3"
)

// ParquetType represents the data types used in Parquet.
type ParquetType struct {
	DataType     types.DataType
	ParquetTypes []string
}

// ParquetTypes defines all possible Parquet data types.
var ParquetTypes = map[string]ParquetType{
	"string": {
		DataType:     types.STRING,
		ParquetTypes: []string{"BYTE_ARRAY"},
	},
	"boolean": {
		DataType:     types.BOOL,
		ParquetTypes: []string{"BOOLEAN"},
	},
	"number": {
		DataType:     types.FLOAT64,
		ParquetTypes: []string{"DOUBLE", "FLOAT"},
	},
	"integer": {
		DataType:     types.INT64,
		ParquetTypes: []string{"INT32", "INT64", "INT96"},
	},
	"decimal": {
		DataType:     types.FLOAT64,
		ParquetTypes: []string{"INT32", "INT64", "FIXED_LEN_BYTE_ARRAY"},
	},
	"timestamp": {
		DataType:     types.TIMESTAMP,
		ParquetTypes: []string{"INT32", "INT64", "INT96"},
	},
	"date": {
		DataType:     types.TIMESTAMP,
		ParquetTypes: []string{"INT32", "INT64", "INT96"},
	},
	"time": {
		DataType:     types.TIMESTAMP,
		ParquetTypes: []string{"INT32", "INT64", "INT96"},
	},
}

type Parquet struct {
	fileKey      string
	next         bool
	batchSize    *int64
	reader       *goparquet.FileReader
	isPreloading atomic.Bool
}

func InitParquet(s3Session *s3.S3, bucket, fileKey string, batchSize *int64) (*Parquet, error) {
	source, err := s3parquet.NewS3FileReaderWithClient(context.Background(), s3Session, bucket, fileKey)
	if err != nil {
		return nil, err
	}

	filereader, err := goparquet.NewFileReader(source)
	if err != nil {
		return nil, err
	}

	reader := Parquet{
		fileKey:      fileKey,
		next:         true,
		reader:       filereader,
		batchSize:    batchSize,
		isPreloading: atomic.Bool{},
	}

	reader.isPreloading.Store(false)
	reader.preload()

	return &reader, nil
}

func (p *Parquet) parseFieldType(neededLogicalType, neededPQType string) (types.DataType, error) {
	if pqType, found := ParquetTypes[neededLogicalType]; found {
		if neededPQType != "" && !utils.ExistInArray(pqType.ParquetTypes, neededPQType) {
			return "", fmt.Errorf("incorrect parquet physical type[%s]; logical type[%s]", neededPQType, neededLogicalType)
		}

		return pqType.DataType, nil
	}

	for _, pqType := range ParquetTypes {
		if utils.ExistInArray(pqType.ParquetTypes, neededPQType) {
			return pqType.DataType, nil
		}
	}

	return "", fmt.Errorf("incorrect parquet physical type[%s]; logical type[%s]", neededPQType, neededLogicalType)
}

func (p *Parquet) GetSchema() (map[string]*types.Property, error) {
	output := make(map[string]*types.Property)
	for _, column := range p.reader.Columns() {
		columnType, err := p.parseFieldType(p.getLogicalTypeFromSDK(column), column.Type().String())
		if err != nil {
			return nil, err
		}

		output[column.Name()] = &types.Property{
			Type: []types.DataType{columnType},
		}
	}

	return output, nil
}

func (p *Parquet) getLogicalTypeFromSDK(column *goparquet.Column) string {
	logicalType := column.Element().GetLogicalType()
	if logicalType == nil {
		return "unknown"
	}

	if logicalType.IsSetSTRING() {
		return "string"
	}
	if logicalType.IsSetMAP() {
		return "map"
	}
	if logicalType.IsSetLIST() {
		return "list"
	}
	if logicalType.IsSetENUM() {
		return "enum"
	}
	if logicalType.IsSetDECIMAL() {
		return "decimal"
	}
	if logicalType.IsSetDATE() {
		return "date"
	}
	if logicalType.IsSetTIME() {
		return "time"
	}
	if logicalType.IsSetTIMESTAMP() {
		return "timestamp"
	}
	if logicalType.IsSetINTEGER() {
		return "integer"
	}
	if logicalType.IsSetUNKNOWN() {
		return "unknown"
	}
	if logicalType.IsSetJSON() {
		return "json"
	}
	if logicalType.IsSetBSON() {
		return "bson"
	}
	if logicalType.IsSetUUID() {
		return "uuid"
	}

	return "unknown"
}

func (p *Parquet) Read() ([]map[string]any, error) {
	batch := []map[string]any{}

	// preloading going on
	for p.isPreloading.Load() {
		time.Sleep(time.Second)
	}

	for idx := int64(0); idx < *p.batchSize && (p.reader.CurrentRowGroup() != nil && int64(idx) < p.reader.CurrentRowGroup().NumRows); idx++ {
		record, err := p.reader.NextRow()
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			// found end of file
			// set next to false and break
			p.next = false
			break
		}

		for key, value := range record {
			column := p.reader.GetColumnByName(key)
			if column == nil {
				continue
			}

			converted, err := p.convertFieldData(p.getLogicalTypeFromSDK(column), column.Type().String(), value)
			if err != nil {
				return nil, err
			}
			record[key] = converted
		}

		batch = append(batch, utils.OperateOnDynamicMap(record, func(s string) string {
			output, err := url.PathUnescape(s)
			if err != nil {
				return s
			}
			return output
		}))
	}

	p.preload()

	return batch, nil
}

func (p *Parquet) HasNext() bool {
	return p.next
}

func (p *Parquet) preload() {
	go func() {
		defer p.isPreloading.Store(false)
		// if preloading is not going on preload
		for !p.isPreloading.Load() {
			p.isPreloading.Store(true)
			err := p.reader.PreLoad()
			if err != nil && err != io.EOF {
				logger.Errorf("preloading failed: %s", err)
			}
		}
	}()
}

func (p *Parquet) convertFieldData(logicalType, pqType string, fieldValue any) (any, error) {
	if fieldValue == nil {
		return nil, nil
	}

	datatype, err := p.parseFieldType(logicalType, pqType)
	if err != nil {
		return nil, err
	}

	return typeutils.ReformatValueOnDataTypes([]types.DataType{datatype}, fieldValue)
}
