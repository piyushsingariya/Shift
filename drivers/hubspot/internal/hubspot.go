package driver

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gear5sh/gear5/drivers/base"
	"github.com/goccy/go-json"

	"github.com/gear5sh/gear5/jsonschema"
	"github.com/gear5sh/gear5/jsonschema/schema"
	"github.com/gear5sh/gear5/protocol"
	"github.com/gear5sh/gear5/types"
	"github.com/gear5sh/gear5/utils"
)

type Hubspot struct {
	*base.Driver

	allStreams  map[string]HubspotStream
	client      *http.Client
	accessToken string
	config      *Config
}

func (h *Hubspot) Setup(config any, base *base.Driver) error {
	h.Driver = base

	conf := &Config{}
	if err := utils.Unmarshal(config, conf); err != nil {
		return err
	}

	if err := conf.ValidateAndPopulateDefaults(); err != nil {
		return fmt.Errorf("failed to validate config: %s", err)
	}

	client, accessToken, err := newClient(conf)
	if err != nil {
		return err
	}

	h.config = conf
	h.client = client
	h.accessToken = accessToken
	h.setupAllStreams()

	return nil
}

func (h *Hubspot) Spec() (schema.JSONSchema, error) {
	return jsonschema.Reflect(Config{})
}

func (h *Hubspot) Check() error {
	return nil
}

func (h *Hubspot) Discover() ([]protocol.Stream, error) {
	streams := []protocol.Stream{}

	for _, hstream := range h.allStreams {
		stream := types.NewStream(hstream.Name(), "").WithSyncMode(hstream.Modes()...).
			WithPrimaryKey(hstream.PrimaryKey()...)
		stream.WithCursorField(hstream.cursorField())

		streams = append(streams, stream)
	}

	return streams, nil
}

func (h *Hubspot) Type() string {
	return "Hubspot"
}

// func (h *Hubspot) GetState() (*types.State, error) {
// 	state := &types.State{}
// 	for _, stream := range h.Catalog().Streams {
// 		if stream.SyncMode == types.Incremental {
// 			hubspotStream, found := h.allStreams[stream.Name()]
// 			if !found {
// 				return nil, fmt.Errorf("hubspot stream not found while getting state of incremental stream[%s]", stream.Name())
// 			}

// 			if !utils.ExistInArray(hubspotStream.Modes(), types.Incremental) {
// 				logger.Warnf("Skipping getting state from stream[%s], this stream doesn't support incremental", stream.Name())
// 				continue
// 			}

// 			state.Add(stream.Name(), stream.Name(), hubspotStream.state())
// 		}
// 	}

// 	return state, nil
// }

func (h *Hubspot) Read(stream protocol.Stream, channel chan<- types.Record) error {
	hstream, found := h.allStreams[stream.Name()]
	if !found {
		return fmt.Errorf("invalid stream passed: %s", stream.Name())
	}

	// hstream.setup(stream.GetSyncMode(), h.Get(stream.Name(), stream.Namespace()))

	err := hstream.readRecords(channel)
	if err != nil {
		return fmt.Errorf("error occurred: %s", err)
	}

	return nil
}

func (h *Hubspot) getGrantedScopes() ([]string, error) {
	req, err := http.NewRequest("GET", formatEndpoint(fmt.Sprintf("oauth/v1/access-tokens/%s", h.accessToken)), nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response := map[string]any{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response["scopes"].([]string), nil
}

func (h *Hubspot) register(streamName string, stream HubspotStream) {
	h.allStreams[streamName] = stream
}

func (h *Hubspot) setupAllStreams() {
	// initialization
	h.allStreams = map[string]HubspotStream{}

	h.register("contacts",
		newCRMSearchStream(*newIncrementalStream("contacts", "contact", h.client, h.config.StartDate),
			"id", "lastmodifieddate",
			[]string{"contacts", "companies"}, []string{"crm.objects.contacts.read"}))

	h.register("companies",
		newCRMSearchStream(
			*newIncrementalStream("companies", "company", h.client, h.config.StartDate),
			"id", "hs_lastmodifieddate",
			[]string{"contacts"}, []string{"crm.objects.contacts.read", "crm.objects.companies.read"}))

	h.register("engagementscalls",
		newCRMSearchStream(
			*newIncrementalStream("engagementscalls", "calls", h.client, h.config.StartDate),
			"id", "hs_lastmodifieddate",
			[]string{"contacts", "deal", "company", "tickets"}, []string{"crm.objects.contacts.read"}))

}
