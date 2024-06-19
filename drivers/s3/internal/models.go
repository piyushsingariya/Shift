package driver

import (
	"fmt"

	"github.com/gear5sh/gear5/logger"
	"github.com/gear5sh/gear5/utils"
)

// Authenticate via Access and Secret Keys
type BaseAWS struct {
	// AccessKey for AWS
	//
	// @jsonschema(
	// required=true
	// )
	AccessKey string `json:"access_key" validate:"required"`
	// SecretAccessKey for AWS
	//
	// @jsonschema(
	// required=true
	// )
	SecretAccessKey string `json:"secret_access_key" validate:"required"`
	// Region for AWS
	//
	// @jsonschema(
	// required=true,
	// enum=["require","disable","verify-ca","verify-full"]
	// )
	Region string `json:"region" validate:"required"`
}

// Authenticate via AssumeRole in foreign Account
type AssumeRoleAWS struct {
	BaseAWS
	// Remote AccountID for AWS
	//
	// @jsonschema(
	// required=true
	// )
	AccountID string `json:"account_id"`
	// RoleName to assume in given AccountID for AWS
	//
	// @jsonschema(
	// required=true
	// )
	RoleName string `json:"role_name"`
}

type Config struct {
	_ struct{} `additionalProperties:"false"`                     // Tags of unnamed field are applied to parent schema.
	_ struct{} `title:"S3" description:"Config to connect to S3"` // Multiple unnamed fields can be used.

	// Stream Name with Patterns
	//
	// @jsonschema(
	// required=true
	// )
	Streams map[string]string `json:"streams" validate:"required" description:"Stream Names with Patterns" required:"true"`
	// FileType
	//
	// @jsonschema(
	// required=true
	// )
	Type string `json:"type" validate:"required"`
	// Bucket Name
	//
	// @jsonschema(
	// required=true
	// )
	Bucket string `json:"bucket" validate:"required"`
	// Bucket Region for AWS
	//
	// @jsonschema(
	// required=true
	// )
	Region string `json:"region" validate:"required"`
	// Credentials for connecting to AWS
	//
	// @jsonschema(
	// required=true,
	// oneOf=["BaseAWS","AssumeRoleAWS"]
	// )
	Credentials interface{} `json:"credentials" validate:"required"`
	// Parallel factor for preload
	//
	// @jsonschema(
	// required=true
	// )
	PreLoadFactor int64 `json:"parallel_factor"`
}

func (c *Config) Validate() error {
	err := utils.Validate(c)
	if err != nil {
		return fmt.Errorf("config validation failed: %s", err)
	}

	if c.PreLoadFactor < 5 {
		logger.Infof("Preload factor %d less than 5: using 5 instead", c.PreLoadFactor)
		c.PreLoadFactor = 5
	}

	return nil
}
