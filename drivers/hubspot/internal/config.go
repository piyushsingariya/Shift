package driver

import (
	"fmt"
	"time"

	"github.com/gear5sh/gear5/logger"
)

type Config struct {
	// Credentials for connecting to the Google Sheets API
	//
	// @jsonschema(
	// required=true,
	// oneOf=["Client","PrivateApp"]
	// )
	Credentials interface{} `json:"credentials"`
	// UTC date and time in the format 2017-01-25T00:00:00Z. Any data before this date will not be replicated.
	//
	// @jsonschema(
	// required=true
	// )
	StartDate time.Time `json:"start_date"`
}

// Authenticate via Hubspot (OAuth)
type Client struct {
	// Enter your Hubspot application Client ID
	//
	// @jsonschema(
	// title="Client ID"
	// )
	ClientID string `json:"client_id"`
	// Enter your Hubspot application Client Secret
	//
	// @jsonschema(
	// title="Client Secret"
	// )
	ClientSecret string `json:"client_secret"`
	// Enter your Hubspot application Refresh Token
	//
	// @jsonschema(
	// title="Refresh Token"
	// )
	RefreshToken string `json:"refresh_token"`
}

// Private App Authentication
type PrivateApp struct {
	// Hubspot Access Token
	//
	// @jsonschema(
	// title="Access Token"
	// )
	AccessToken string `json:"access_token"`
}

func (c *Config) ValidateAndPopulateDefaults() error {
	timeString := "2017-01-25T00:00:00Z"
	defaultTime, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		return err
	}

	if c.StartDate.Before(defaultTime) {
		logger.Warnf("found start time %v before default time, using default time: %v", c.StartDate, defaultTime)
		c.StartDate = defaultTime
	}

	if c.Credentials == nil {
		return fmt.Errorf("credentials not found")
	}

	return nil
}
