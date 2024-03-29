package driver

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gobwas/glob"
	"github.com/piyushsingariya/drivers/s3/models"
	"github.com/piyushsingariya/drivers/s3/reader"
	"github.com/piyushsingariya/shift/jsonschema"
	"github.com/piyushsingariya/shift/jsonschema/schema"
	"github.com/piyushsingariya/shift/logger"
	shiftmodels "github.com/piyushsingariya/shift/models"
	protocol "github.com/piyushsingariya/shift/protocol"
	"github.com/piyushsingariya/shift/safego"
	"github.com/piyushsingariya/shift/types"
	"github.com/piyushsingariya/shift/typing"
	"github.com/piyushsingariya/shift/utils"
)

const patternSymbols = "*[]!{}"

type S3 struct {
	cursorField string
	config      *models.Config
	session     *session.Session
	catalog     *shiftmodels.Catalog
	state       shiftmodels.State
	client      *s3.S3
	batchSize   int64
}

func (s *S3) Setup(config any, catalog *shiftmodels.Catalog, state shiftmodels.State, batchSize int64) error {
	cfg := models.Config{}
	err := utils.Unmarshal(config, &cfg)
	if err != nil {
		return err
	}

	err = cfg.Validate()
	if err != nil {
		return err
	}
	s.config = &cfg

	s.session, err = newSession(cfg.Credentials)
	if err != nil {
		return err
	}

	s.client = s3.New(s.session)
	s.batchSize = batchSize
	s.catalog = catalog
	s.state = state
	s.cursorField = "last_modified_date"

	return nil
}

func (s *S3) Spec() (schema.JSONSchema, error) {
	return jsonschema.Reflect(models.Config{})
}

func (s *S3) Check() error {
	for stream, pattern := range s.config.Streams {
		err := s.iteration(pattern, 1, func(reader reader.Reader, file *s3.Object) (bool, error) {
			// break iteration after single item
			return false, nil
		})
		if err != nil {
			return fmt.Errorf("failed to check stream[%s] pattern[%s]: %s", stream, pattern, err)
		}
	}

	return nil
}

func (s *S3) Discover() ([]*shiftmodels.Stream, error) {
	streams := []*shiftmodels.Stream{}
	for stream, pattern := range s.config.Streams {
		var schema map[string]*shiftmodels.Property
		var err error
		err = s.iteration(pattern, 1, func(reader reader.Reader, file *s3.Object) (bool, error) {
			schema, err = reader.GetSchema()
			return false, err
		})
		if err != nil {
			return nil, fmt.Errorf("failed to check stream[%s] pattern[%s]: %s", stream, pattern, err)
		}

		if schema == nil {
			return nil, fmt.Errorf("no schema found")
		}

		streams = append(streams, &shiftmodels.Stream{
			Name:                stream,
			Namespace:           pattern,
			SupportedSyncModes:  []types.SyncMode{types.Incremental, types.FullRefresh},
			SourceDefinedCursor: true,
			DefaultCursorFields: []string{s.cursorField},
			JSONSchema: &shiftmodels.Schema{
				Properties: schema,
			},
		})
	}

	return streams, nil
}

func (s *S3) Catalog() *shiftmodels.Catalog {
	return s.catalog
}

func (s *S3) Type() string {
	return "S3"
}

// NOTE: S3 read doesn't perform neccessary checks such as matching cursor field present in stream since
// it works only on single cursor field
func (s *S3) Read(stream protocol.Stream, channel chan<- shiftmodels.Record) error {
	name, namespace := stream.Name(), stream.Namespace()
	// get pattern from stream name
	pattern := s.config.Streams[name]
	var localCursor *time.Time

	// if incremental check for state
	if stream.GetSyncMode() == types.Incremental {
		state := s.state.Get(name, namespace)
		if state != nil {
			value, found := state[s.cursorField]
			if found {
				stateCursor, err := typing.ReformatDate(value)
				if err != nil {
					logger.Warnf("failed to parse state for stream %s[%s]", name, namespace)
				} else {
					localCursor = &stateCursor
				}
			} else {
				logger.Warnf("Cursor field not found for stream %s[%s]", name, namespace)
			}
		} else {
			logger.Warnf("State not found for stream %s[%s]", name, namespace)
		}
	}

	err := s.iteration(pattern, s.config.PreLoadFactor, func(reader reader.Reader, file *s3.Object) (bool, error) {
		if localCursor != nil && file.LastModified.Before(*localCursor) {
			// continue iteration
			return true, nil
		}

		totalRecords := 0

		for reader.HasNext() {
			records, err := reader.Read()
			if err != nil {
				// discontinue iteration
				return false, fmt.Errorf("got error while reading records from %s[%s]: %s", name, namespace, err)
			}

			totalRecords += len(records)

			if len(records) == 0 {
				break
			}

			for _, record := range records {
				if !safego.Insert(channel, utils.ReformatRecord(name, namespace, record)) {
					// discontinue iteration since failed to insert records
					return false, nil
				}
			}
		}

		if localCursor == nil {
			localCursor = file.LastModified
		} else {
			localCursor = types.Time((utils.MaxDate(*localCursor, *file.LastModified)))
		}

		logger.Infof("%d Records found in file %s", totalRecords, *file.Key)
		// go to next file
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to read stream[%s] pattern[%s]: %s", name, pattern, err)
	}

	// update the state
	if stream.GetSyncMode() == types.Incremental {
		s.state.Update(name, namespace, map[string]any{
			s.cursorField: localCursor,
		})
	}

	return nil
}

func (s *S3) GetState() (*shiftmodels.State, error) {
	return &s.state, nil
}

func (s *S3) iteration(pattern string, preloadFactor int64, foreach func(reader reader.Reader, file *s3.Object) (bool, error)) error {
	re, err := glob.Compile(pattern)
	if err != nil {
		return fmt.Errorf("failed to complie file pattern please check: https://github.com/gobwas/glob#performance")
	}

	var continuationToken *string
	prefix := ""
	split := strings.Split(pattern, "/")
	for _, i := range split {
		if strings.ContainsAny(i, patternSymbols) {
			break
		}
		prefix = filepath.Join(prefix, i)
	}

	waitgroup := sync.WaitGroup{}
	consumer := make(chan struct {
		reader.Reader
		s3.Object
	}, preloadFactor)

	var breakIteration atomic.Bool
	breakIteration.Store(false)
	var consumerError error

	go func() {
		for file := range consumer {
			waitgroup.Add(1)

			// execute foreach
			next, err := foreach(file.Reader, &file.Object)
			if err != nil {
				consumerError = err
				safego.Close(consumer)
				waitgroup.Done()
				return
			}

			// break iteration
			if !next {
				safego.Close(consumer)
				waitgroup.Done()
				return
			}

			waitgroup.Done()
		}
	}()

	// List objects in the S3 bucket
s3Iteration:
	for {
		resp, err := s.client.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket:            aws.String(s.config.Bucket),
			Prefix:            aws.String(prefix),
			MaxKeys:           aws.Int64(10000000),
			ContinuationToken: continuationToken, // Initialize with nil
		})
		if err != nil {
			return fmt.Errorf("Error listing objects: %s", err)
		}

		// Iterate through the objects and process them
		for _, file := range resp.Contents {
			if re.Match(*file.Key) {
				re, err := reader.Init(s.client, s.config.Type, s.config.Bucket, *file.Key, s.batchSize)
				if err != nil {
					return fmt.Errorf("failed to initialize reader on file[%s]: %s", *file.Key, err)
				}

				if !safego.Insert(consumer, struct {
					reader.Reader
					s3.Object
				}{re, *file}) {
					break s3Iteration
				}
			}
		}

		// Check if there are more objects to retrieve
		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break // Break the loop if there are no more objects
		}

		// Update the continuation token for the next iteration
		continuationToken := resp.NextContinuationToken
		if continuationToken == nil {
			break // Break the loop if the continuation token is nil (should not happen)
		}
	}

	waitgroup.Wait()

	return consumerError
}
