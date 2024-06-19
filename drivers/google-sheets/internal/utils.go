package driver

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/goccy/go-json"

	"github.com/gear5sh/gear5/logger"
	"github.com/gear5sh/gear5/types"
	"github.com/gear5sh/gear5/utils"
	unidecode "github.com/mozillazg/go-unidecode"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gopkg.in/Iwark/spreadsheet.v2"
)

const DefaultSeparator = "_"

func NewClient(config *Config) (*spreadsheet.Service, error) {
	// create api context
	ctx := context.Background()

	var client *http.Client
	var err error
	var credBytes []byte
	if ok, _ := utils.IsOfType(config.Credentials, "client_id"); ok {
		logger.Info("Credentials found to be OAuth")
		oauth := &Client{}
		if err := utils.Unmarshal(config.Credentials, oauth); err != nil {
			return nil, err
		}

		// Create a new OAuth2 config
		config := &oauth2.Config{
			ClientID:     oauth.ClientID,
			ClientSecret: oauth.ClientSecret,
			Endpoint:     google.Endpoint,
			Scopes:       []string{spreadsheet.Scope}, // Adjust the scopes as needed
		}

		// Create a new token source using the refresh token
		tokenSource := config.TokenSource(context.TODO(), &oauth2.Token{
			RefreshToken: oauth.RefreshToken,
		})

		// Create a new OAuth2 client
		client = oauth2.NewClient(context.TODO(), tokenSource)
	} else if ok, err := utils.IsOfType(config.Credentials, "service_account_info"); ok {
		logger.Info("Credentials found to be Service Account")
		serviceAccount := &Service{}
		if err := utils.Unmarshal(config.Credentials, serviceAccount); err != nil {
			return nil, err
		}

		// get bytes from base64 encoded google service accounts key
		credBytes, err = json.Marshal(map[string]string{
			"type":           "service_account",
			"private_key_id": serviceAccount.ServiceAccountInfo,
		})
		if err != nil {
			return nil, err
		}

		// authenticate and get configuration
		jwtConfig, err := google.JWTConfigFromJSON(credBytes, spreadsheet.Scope)
		if err != nil {
			return nil, err
		}

		// create client with config and context
		client = jwtConfig.Client(ctx)
	} else {
		return nil, fmt.Errorf("invalid credentials format, expected formats are: %T and %T", Client{}, Service{})
	}

	if client == nil {
		return nil, fmt.Errorf("failed to create spreadsheet configuration client")
	}

	service := spreadsheet.NewServiceWithClient(client)
	if service == nil {
		return nil, fmt.Errorf("failed to create spreadsheet service")
	}

	// fetch spreadsheet
	_, err = service.FetchSpreadsheet(config.SpreadsheetID)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return service, nil
}

func LoadHeaders(sheet spreadsheet.Sheet) ([]string, error) {
	if len(sheet.Rows) == 0 {
		return []string{}, fmt.Errorf("expected data for exactly one range for sheet %s", sheet.Properties.Title)
	}

	if sheet.Properties.GridProperties.ColumnCount == 0 {
		logger.Warnf("[%s] The sheet %s (ID %s) columns are empty!", EmptySheetError, sheet.Properties.Title, sheet.Properties.ID)
		return []string{}, nil
	}

	if sheet.Properties.GridProperties.RowCount == 0 {
		logger.Warnf("[%s] The sheet %s (ID %s) rows are empty!", EmptySheetError, sheet.Properties.Title, sheet.Properties.ID)
		return []string{}, nil
	}

	if sheet.Properties.GridProperties.RowCount == 1 {
		logger.Warnf("[%s] no data found in the sheet %s (ID %s) rows are empty!", EmptySheetError, sheet.Properties.Title, sheet.Properties.ID)
		return []string{}, nil
	}

	var headers []string

	for _, pointer := range sheet.Rows[0] {
		headers = append(headers, pointer.Value)
	}

	return headers, nil
}

func GetValidHeadersAndDuplicates(headers []string) ([]string, []string) {
	fields := []string{}
	duplicateHeaders := []string{}
	set := make(map[string]bool)

	for _, header := range headers {
		if _, found := set[header]; found {
			duplicateHeaders = append(duplicateHeaders, header)
			fields = append(fields, header)
		} else {
			set[header] = true
			fields = append(fields, header)
		}
	}

	return fields, duplicateHeaders
}

func SafeNameConversion(text string) (string, error) {
	//   convert name using a set of rules, for example: '1MyName' -> '_1_my_name'
	pattern, err := regexp.Compile("[A-Z]+[a-z]*|[a-z]+|\\d+|(?P<NoToken>[^a-zA-Z\\d]+)")
	if err != nil {
		return text, err
	}

	text = unidecode.Unidecode(text)

	tokens := []string{}
	for _, m := range pattern.FindAllStringSubmatch(text, -1) {
		if m[1] == "" {
			tokens = append(tokens, m[0])
		} else {
			tokens = append(tokens, "")
		}
	}

	if len(tokens) >= 3 {
		newTokens := []string{tokens[0]}
		for _, t := range tokens[1 : len(tokens)-1] {
			if t != "" {
				newTokens = append(newTokens, t)
			}
		}
		newTokens = append(newTokens, tokens[len(tokens)-1])
		tokens = newTokens
	}

	if len(tokens) > 0 {
		if _, err := strconv.Atoi(tokens[0]); err == nil {
			tokens = append([]string{""}, tokens...)
		}
	}

	text = strings.Join(tokens, DefaultSeparator)

	return strings.ToLower(text), nil
}

func headersToStream(sheetName string, headers []string) *types.Stream {
	stream := types.NewStream(sheetName, "")
	stream.WithSyncMode(types.FULLREFRESH)

	for _, header := range headers {
		stream.UpsertField(header, types.STRING, false)
	}

	return stream
}

func GetIndexToColumn(sheet spreadsheet.Sheet) (map[int]string, error) {
	headers, err := LoadHeaders(sheet)
	if err != nil {
		return nil, err
	}

	output := make(map[int]string)

	for i := range headers {
		headers[i], err = SafeNameConversion(headers[i])
		if err != nil {
			return nil, err
		}

		output[i] = headers[i]
	}

	return output, nil
}
