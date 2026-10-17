package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

const pageSize = 100
const maxResponseBytes = 4 << 20

var namespacePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,62}$`)

var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

type settings struct {
	URL               string `json:"url"`
	Namespace         string `json:"namespace"`
	AllowInsecureHTTP bool   `json:"allowInsecureHTTP"`
}

type Datasource struct {
	settings settings
	token    string
	client   *http.Client
}

func NewDatasource(ctx context.Context, instance backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	var config settings
	if len(instance.JSONData) > 0 && json.Unmarshal(instance.JSONData, &config) != nil {
		return nil, errors.New("invalid Commander connection settings")
	}
	opts, err := instance.HTTPClientOptions(ctx)
	if err != nil {
		return nil, errors.New("invalid HTTP client settings")
	}
	opts.ConfigureClient = func(_ httpclient.Options, client *http.Client) {
		client.Timeout = 10 * time.Second
		// Never forward credentials to a redirect target, including the same host.
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	client, err := httpclient.New(opts)
	if err != nil {
		return nil, errors.New("could not create Commander HTTP client")
	}
	return &Datasource{settings: config, token: instance.DecryptedSecureJSONData["token"], client: client}, nil
}

func (d *Datasource) Dispose() { d.client.CloseIdleConnections() }

func (d *Datasource) endpoint() (*url.URL, error) {
	if d.settings.URL == "" || d.token == "" || d.settings.Namespace == "" {
		return nil, errors.New("configure a Commander URL, namespace and access token in the data source")
	}
	u, err := url.Parse(d.settings.URL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, errors.New("Commander URL must be an HTTP(S) URL without credentials, query or fragment")
	}
	if u.Scheme == "http" && !d.settings.AllowInsecureHTTP {
		return nil, errors.New("use HTTPS, or explicitly allow HTTP for a trusted local development connection")
	}
	if !namespacePattern.MatchString(d.settings.Namespace) {
		return nil, errors.New("invalid configured namespace")
	}
	u = u.JoinPath("api/v1/namespaces", d.settings.Namespace, "agents")
	return u, nil
}

// Decode only the fields allowed into Grafana. Configurations and credentials
// from the management API must never become frames, error text or log messages.
type agent struct {
	Metadata struct {
		UID          string `json:"instanceUid"`
		Namespace    string `json:"namespace"`
		Type         string `json:"type"`
		Capabilities uint64 `json:"capabilities"`
		Description  struct {
			Identifying    map[string]string `json:"identifyingAttributes"`
			NonIdentifying map[string]string `json:"nonIdentifyingAttributes"`
		} `json:"description"`
	} `json:"metadata"`
	Status struct {
		Connected *bool `json:"connected"`
		Health    struct {
			Healthy *bool `json:"healthy"`
		} `json:"componentHealth"`
		LastReported string `json:"lastReportedAt"`
	} `json:"status"`
}

type agentPage struct {
	Items    []agent `json:"items"`
	Metadata struct {
		Continue string `json:"continue"`
	} `json:"metadata"`
}

func (d *Datasource) list(ctx context.Context, cursor string) (*agentPage, backend.Status, error) {
	endpoint, err := d.endpoint()
	if err != nil {
		return nil, backend.StatusBadRequest, err
	}
	if len(cursor) > 4096 {
		return nil, backend.StatusBadRequest, errors.New("continuation token is too long")
	}
	params := url.Values{"limit": {fmt.Sprint(pageSize)}}
	if cursor != "" {
		params.Set("continue", cursor)
	}
	endpoint.RawQuery = params.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, backend.StatusBadRequest, errors.New("invalid Commander request")
	}
	request.Header.Set("Authorization", "Bearer "+d.token)
	request.Header.Set("Accept", "application/json")
	response, err := d.client.Do(request)
	if err != nil {
		return nil, backend.StatusBadGateway, errors.New("Commander request failed or timed out; check the connection")
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, backend.StatusUnauthorized, errors.New("Commander access token is invalid or expired; update the data source token")
	case http.StatusForbidden:
		return nil, backend.StatusForbidden, errors.New("Commander denied access to this namespace")
	default:
		return nil, backend.StatusBadGateway, fmt.Errorf("Commander returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return nil, backend.StatusBadGateway, errors.New("could not read Commander response")
	}
	if len(body) > maxResponseBytes {
		return nil, backend.StatusBadGateway, errors.New("Commander response exceeds the 4 MiB safety limit")
	}
	var page agentPage
	if json.Unmarshal(body, &page) != nil || page.Items == nil || len(page.Items) > pageSize || len(page.Metadata.Continue) > 4096 {
		return nil, backend.StatusBadGateway, errors.New("invalid or oversized Commander agent list")
	}
	for _, item := range page.Items {
		if item.Metadata.UID == "" || item.Metadata.Namespace != d.settings.Namespace {
			return nil, backend.StatusBadGateway, errors.New("Commander returned an invalid agent identity or namespace")
		}
	}
	return &page, backend.StatusOK, nil
}

func (d *Datasource) QueryData(ctx context.Context, request *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()
	if len(request.Queries) > 10 {
		return nil, errors.New("at most 10 fleet queries are allowed per request")
	}
	for _, query := range request.Queries {
		var model struct {
			QueryType string `json:"queryType"`
			Cursor    string `json:"cursor"`
		}
		if len(query.JSON) > 0 && json.Unmarshal(query.JSON, &model) != nil {
			response.Responses[query.RefID] = backend.ErrDataResponse(backend.StatusBadRequest, "invalid fleet query")
			continue
		}
		if model.QueryType != "" && model.QueryType != "fleet" {
			response.Responses[query.RefID] = backend.ErrDataResponse(backend.StatusBadRequest, "only fleet queries are supported")
			continue
		}
		page, status, err := d.list(ctx, model.Cursor)
		if err != nil {
			response.Responses[query.RefID] = backend.ErrDataResponse(status, err.Error())
			continue
		}
		frame := data.NewFrame("fleet",
			data.NewField("Instance UID", nil, []string{}), data.NewField("Name", nil, []string{}),
			data.NewField("Namespace", nil, []string{}), data.NewField("Type", nil, []string{}),
			data.NewField("Connection", nil, []string{}), data.NewField("Health", nil, []string{}),
			data.NewField("Last reported", nil, []string{}),
		)
		frame.Meta = &data.FrameMeta{Custom: map[string]string{"namespace": d.settings.Namespace, "nextCursor": page.Metadata.Continue}}
		for _, item := range page.Items {
			connection := "Unknown"
			if item.Status.Connected != nil {
				connection = "Disconnected"
				if *item.Status.Connected {
					connection = "Connected"
				}
			}
			health := "Unknown"
			if item.Metadata.Capabilities&2048 != 0 && item.Status.Health.Healthy != nil {
				health = "Unhealthy"
				if *item.Status.Health.Healthy {
					health = "Healthy"
				}
			}
			name := strings.TrimSpace(item.Metadata.Description.NonIdentifying["host.name"])
			if name == "" {
				name = strings.TrimSpace(item.Metadata.Description.Identifying["service.instance.id"])
			}
			if name == "" {
				name = item.Metadata.UID
			}
			frame.AppendRow(item.Metadata.UID, name, item.Metadata.Namespace, item.Metadata.Type, connection, health, item.Status.LastReported)
		}
		response.Responses[query.RefID] = backend.DataResponse{Frames: data.Frames{frame}}
	}
	return response, nil
}

func (d *Datasource) CheckHealth(ctx context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	_, _, err := d.list(ctx, "")
	if err != nil {
		return &backend.CheckHealthResult{Status: backend.HealthStatusError, Message: err.Error()}, nil
	}
	return &backend.CheckHealthResult{Status: backend.HealthStatusOk, Message: "Authorized to read agents in namespace " + d.settings.Namespace}, nil
}
