package driver

import (
	"fmt"
	"strings"

	"github.com/gear5sh/gear5/drivers/base"
	"github.com/gear5sh/gear5/logger"
	"github.com/gear5sh/gear5/protocol"
	"github.com/gear5sh/gear5/types"
	"github.com/gear5sh/gear5/utils"
	"gopkg.in/Iwark/spreadsheet.v2"
)

type GoogleSheets struct {
	*base.Driver
	*spreadsheet.Service

	config *Config
}

func (gs *GoogleSheets) Setup(config any, base *base.Driver) error {
	gs.Driver = base

	conf := &Config{}
	if err := utils.Unmarshal(config, conf); err != nil {
		return err
	}

	if err := conf.ValidateAndPopulateDefaults(); err != nil {
		return fmt.Errorf("failed to validate config: %s", err)
	}

	client, err := NewClient(conf)
	if err != nil {
		return err
	}
	gs.config = conf
	gs.Service = client

	return nil
}

func (gs *GoogleSheets) Spec() any {
	return Config{}
}

func (gs *GoogleSheets) Type() string {
	return "Google-Sheets"
}

func (gs *GoogleSheets) Check() error {
	_, _, err := gs.getAllSheetStreams()
	if err != nil {
		return err
	}

	return nil
}

func (gs *GoogleSheets) Discover() ([]*types.Stream, error) {
	streams, _, err := gs.getAllSheetStreams()
	if err != nil {
		return nil, err
	}

	return streams, nil
}

func (gs *GoogleSheets) Read(stream protocol.Stream, channel chan<- types.Record) error {
	spreadsheetID := gs.config.SpreadsheetID

	logger.Infof("Starting sync for spreadsheet [%s]", spreadsheetID)

	_, streamNamesToSheet, err := gs.getAllSheetStreams()
	if err != nil {
		return err
	}

	sheet, found := streamNamesToSheet[stream.Name()]
	if !found {
		logger.Infof("sheet not found with stream name [%s] in spreadsheet; skipping", stream.Name())
		return nil
	}

	indexToHeaders, err := GetIndexToColumn(sheet)
	if err != nil {
		return fmt.Errorf("failed to mark headers to index: %s", err)
	}

	logger.Infof("Row count in sheet %s[id: %d]:%d", sheet.Properties.Title, sheet.Properties.ID, sheet.Properties.GridProperties.RowCount-1)

	for rowCursor := int64(1); rowCursor < int64(len(sheet.Rows)); rowCursor++ {
		// make a batch of records
		record := types.Record{Stream: stream.Name(), Namespace: stream.Namespace(), Data: make(map[string]interface{})}
		for i, pointer := range sheet.Rows[rowCursor] {
			record.Data[indexToHeaders[i]] = pointer.Value
		}

		channel <- record
	}

	logger.Infof("Total records fetched %s[%d]", stream.Name(), len(sheet.Rows)-1)

	return err
}

func (gs *GoogleSheets) getAllSheetStreams() ([]*types.Stream, map[string]spreadsheet.Sheet, error) {
	logger.Infof("fetching spreadsheet[%s]", gs.config.SpreadsheetID)
	googleSpreadsheet, err := gs.FetchSpreadsheet(gs.config.SpreadsheetID)
	if err != nil {
		return nil, nil, err
	}

	streams := []*types.Stream{}
	streamNameToSheet := make(map[string]spreadsheet.Sheet)
	for _, sheet := range googleSpreadsheet.Sheets {
		headers, err := LoadHeaders(sheet)
		if err != nil {
			if strings.Contains(err.Error(), EmptySheetError) {
				logger.Infof("Skipping empty sheet: %s", err.Error())
				continue
			}
			return nil, nil, err
		}

		if gs.config.NameConversion != nil && *gs.config.NameConversion {
			for i := range headers {
				headers[i], err = SafeNameConversion(headers[i])
				if err != nil {
					logger.Errorf("failed to safely convert header %s: %s", headers[i], err)
				}
			}
		}

		headers, duplicateHeaders := GetValidHeadersAndDuplicates(headers)
		if len(duplicateHeaders) > 0 {
			return nil, nil, fmt.Errorf("found duplicate headers in Sheet[%s]: %s", sheet.Properties.Title, strings.Join(duplicateHeaders, ", "))
		}

		streams = append(streams, headersToStream(sheet.Properties.Title, headers))
		streamNameToSheet[sheet.Properties.Title] = sheet
	}

	return streams, streamNameToSheet, nil
}
