package driver

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gear5sh/gear5/logger"
	"github.com/gear5sh/gear5/types"
	"github.com/gear5sh/gear5/utils"
	"golang.org/x/oauth2"
)

var ValidJsonSchemaTypes = []types.DataType{
	types.STRING,
	types.INT64,
	types.FLOAT64,
	types.BOOL,
	types.OBJECT,
	types.ARRAY,
}

var KnownConvertibleSchemaTypes = map[string]types.Property{
	"bool":         {Type: []types.DataType{types.BOOL}},
	"enumeration":  {Type: []types.DataType{types.STRING}},
	"date":         {Type: []types.DataType{types.TIMESTAMP}},
	"date-time":    {Type: []types.DataType{types.TIMESTAMP}},
	"datetime":     {Type: []types.DataType{types.TIMESTAMP}},
	"json":         {Type: []types.DataType{types.STRING}},
	"phone_number": {Type: []types.DataType{types.STRING}},
}

var CustomFieldTypeToValue = map[reflect.Type]string{
	reflect.TypeOf(true):     "boolean",
	reflect.TypeOf(""):       "string",
	reflect.TypeOf(2.000000): "number",
	reflect.TypeOf(1):        "integer",
}

var CustomFieldValueToType []string

const BaseURL = "https://api.hubapi.com"

func formatEndpoint(urn string) string {
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(BaseURL, "/"), strings.TrimPrefix(urn, "/"))
}

func newClient(config *Config) (*http.Client, string, error) {
	var client *http.Client
	var accessToken string
	if ok, _ := utils.IsOfType(config.Credentials, "client_id"); ok {
		logger.Info("Credentials found to be OAuth")
		oauth := &Client{}
		if err := utils.Unmarshal(config.Credentials, oauth); err != nil {
			return nil, "", err
		}

		// Create a new OAuth2 config
		config := &oauth2.Config{
			ClientID:     oauth.ClientID,
			ClientSecret: oauth.ClientSecret,
			Endpoint: oauth2.Endpoint{
				TokenURL: formatEndpoint("/oauth/v1/token"),
			},
		}

		// Create a new token source using the refresh token
		tokenSource := config.TokenSource(context.TODO(), &oauth2.Token{
			RefreshToken: oauth.RefreshToken,
		})

		token, err := tokenSource.Token()
		if err != nil {
			return nil, "", fmt.Errorf("failed to retrive access token from refresh token: %s", err)
		}

		accessToken = token.AccessToken

		// Create a new OAuth2 client
		client = oauth2.NewClient(context.TODO(), tokenSource)
	} else if ok, _ := utils.IsOfType(config.Credentials, "access_token"); ok {
		logger.Info("Credentials found to be Private App")
		privateApp := &PrivateApp{}
		err := utils.Unmarshal(config.Credentials, privateApp)
		if err != nil {
			return nil, "", err
		}

		// Create a new OAuth2 config
		config := &oauth2.Config{
			Endpoint: oauth2.Endpoint{
				TokenURL: formatEndpoint("/oauth/v1/token"),
			},
		}

		// Create a new token source using the refresh token
		tokenSource := config.TokenSource(context.TODO(), &oauth2.Token{
			AccessToken: privateApp.AccessToken,
		})

		accessToken = privateApp.AccessToken

		// Create a new OAuth2 client
		client = oauth2.NewClient(context.TODO(), tokenSource)
	} else {
		return nil, "", fmt.Errorf("invalid credentials format, expected formats are: %T and %T", Client{}, PrivateApp{})
	}

	if client == nil {
		return nil, "", fmt.Errorf("failed to create hubspot authorized client")
	}

	return client, accessToken, nil
}

func getFieldProps(fieldType string) *types.Property {
	if utils.ExistInArray(ValidJsonSchemaTypes, types.DataType(fieldType)) {
		return &types.Property{
			Type: []types.DataType{types.DataType(fieldType)},
		}
	}

	if property, found := KnownConvertibleSchemaTypes[fieldType]; !found {
		return &property
	} else {
		return &types.Property{
			Type: []types.DataType{types.DataType(fieldType)},
		}
	}
}

// def _parse_and_handle_errors(response) -> Union[MutableMapping[str, Any], List[MutableMapping[str, Any]]]:
//         """Handle response"""
//         message = "Unknown error"
//         if response.headers.get("content-type") == "application/json;charset=utf-8" and response.status_code != HTTPStatus.OK:
//             message = response.json().get("message")

//         if response.status_code == HTTPStatus.FORBIDDEN:
//             """Once hit the forbidden endpoint, we return the error message from response."""
//             pass
//         elif response.status_code in (HTTPStatus.UNAUTHORIZED, CLOUDFLARE_ORIGIN_DNS_ERROR):
//             raise HubspotInvalidAuth(message, response=response)
//         elif response.status_code == HTTPStatus.TOO_MANY_REQUESTS:
//             retry_after = response.headers.get("Retry-After")
//             raise HubspotRateLimited(
//                 f"429 Rate Limit Exceeded: API rate-limit has been reached until {retry_after} seconds."
//                 " See https://developers.hubspot.com/docs/api/usage-details",
//                 response=response,
//             )
//         elif response.status_code in (HTTPStatus.BAD_GATEWAY, HTTPStatus.SERVICE_UNAVAILABLE):
//             raise HubspotTimeout(message, response=response)
//         else:
//             response.raise_for_status()

//         return response.json()
