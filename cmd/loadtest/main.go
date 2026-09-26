// Command loadtest exercises a running OpAMP Commander with simulated agents.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"maps"
	"math"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	mongooptions "go.mongodb.org/mongo-driver/v2/mongo/options"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	api "github.com/minuk-dev/opampcommander/pkg/client"
)

type options struct {
	URL            string
	APIURL         string
	MetricsURL     string
	MongoURI       string
	Token          string
	Username       string
	Password       string
	Count          int
	Rate           int
	Duration       time.Duration
	ReportInterval time.Duration
	ChurnInterval  time.Duration
	OpsInterval    time.Duration
	Output         string
}

type summary struct {
	Count int     `json:"count"`
	P50MS float64 `json:"p50_ms"`
	P95MS float64 `json:"p95_ms"`
	P99MS float64 `json:"p99_ms"`
}

type report struct {
	StartedAt        time.Time          `json:"started_at"`
	DurationSeconds  float64            `json:"duration_seconds"`
	RequestedAgents  int                `json:"requested_agents"`
	ConnectionEvents int                `json:"connection_events"`
	ConnectPerSecond float64            `json:"connect_per_second"`
	ConnectionErrors int                `json:"connection_errors"`
	ServerErrors     int                `json:"server_errors"`
	ConfigPushes     int                `json:"config_pushes"`
	ConnectLatency   summary            `json:"connect_latency"`
	Propagation      summary            `json:"propagation_latency"`
	MetricsStart     map[string]float64 `json:"metrics_start,omitempty"`
	MetricsEnd       map[string]float64 `json:"metrics_end,omitempty"`
	PeakMemoryBytes  float64            `json:"peak_memory_bytes,omitempty"`
	PeakGoroutines   float64            `json:"peak_goroutines,omitempty"`
	MetricsError     string             `json:"metrics_error,omitempty"`
	MongoStart       map[string]int64   `json:"mongo_start,omitempty"`
	MongoEnd         map[string]int64   `json:"mongo_end,omitempty"`
	MongoPerSecond   map[string]float64 `json:"mongo_per_second,omitempty"`
	MongoError       string             `json:"mongo_error,omitempty"`
}

type recorder struct {
	mu         sync.Mutex
	connected  int
	connErrors int
	serverErrs int
	connect    []time.Duration
	propagate  []time.Duration
	pushAt     map[string]time.Time
	pushes     int
}

type agent struct {
	client    client.OpAMPClient
	started   uint64
	mu        sync.Mutex
	effective map[string]*protobufs.AgentConfigObject
	seen      map[string]bool
}

func main() {
	if err := mainRun(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func mainRun() error {
	var o options
	flag.StringVar(&o.URL, "url", "ws://localhost:8080/api/v1/opamp", "OpAMP WebSocket URL")
	flag.StringVar(&o.APIURL, "api-url", "http://localhost:8080", "REST API base URL")
	flag.StringVar(&o.MetricsURL, "metrics-url", "http://localhost:9090/metrics", "server Prometheus endpoint (empty to disable)")
	flag.StringVar(&o.MongoURI, "mongo-uri", "", "MongoDB URI for serverStatus operation counters (optional)")
	flag.StringVar(&o.Token, "token", os.Getenv("OPAMP_LOAD_TOKEN"), "REST bearer token for config pushes")
	flag.StringVar(&o.Username, "username", "", "REST basic-auth username for config pushes")
	flag.StringVar(&o.Password, "password", "", "REST basic-auth password for config pushes")
	flag.IntVar(&o.Count, "count", 100, "number of concurrent agents")
	flag.IntVar(&o.Rate, "rate", 20, "new connections per second")
	flag.DurationVar(&o.Duration, "duration", time.Minute, "total run duration")
	flag.DurationVar(&o.ReportInterval, "report-interval", 10*time.Second, "health report interval")
	flag.DurationVar(&o.ChurnInterval, "churn-interval", 0, "replace one connected agent at this interval (0 disables)")
	flag.DurationVar(&o.OpsInterval, "ops-interval", 0, "push a new group config at this interval (0 disables)")
	flag.StringVar(&o.Output, "output", "loadtest-result.json", "JSON report path")
	flag.Parse()
	if err := validate(o); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, o.Duration)
	defer cancel()
	return run(ctx, o)
}

func validate(o options) error {
	if o.Count < 1 || o.Rate < 1 || o.Duration <= 0 || o.ReportInterval <= 0 || o.ChurnInterval < 0 || o.OpsInterval < 0 {
		return errors.New("count, rate, duration and report-interval must be positive; churn-interval and ops-interval cannot be negative")
	}
	if !strings.HasPrefix(o.URL, "ws://") && !strings.HasPrefix(o.URL, "wss://") {
		return errors.New("url must be a ws:// or wss:// URL")
	}
	if o.OpsInterval > 0 && o.Token == "" && (o.Username == "" || o.Password == "") {
		return errors.New("-token or both -username and -password are required for management operations")
	}
	return nil
}

func run(ctx context.Context, o options) error {
	started := time.Now()
	r := &recorder{pushAt: make(map[string]time.Time)}
	result := report{StartedAt: started, RequestedAgents: o.Count}
	if o.MetricsURL != "" {
		result.MetricsStart, result.MetricsError = scrape(ctx, o.MetricsURL)
		result.recordPeak(result.MetricsStart)
	}
	var mongoClient *mongo.Client
	if o.MongoURI != "" {
		var err error
		mongoClient, err = mongo.Connect(mongooptions.Client().ApplyURI(o.MongoURI))
		if err != nil {
			return fmt.Errorf("connect MongoDB: %w", err)
		}
		defer func() { _ = mongoClient.Disconnect(context.Background()) }()
		result.MongoStart, result.MongoError = mongoCounters(ctx, mongoClient)
	}

	agents := make([]*agent, o.Count)
	var group *v1.AgentGroup
	var apiClient *api.Client
	if o.OpsInterval > 0 {
		if o.Token != "" {
			apiClient = api.New(o.APIURL, api.WithBearerToken(o.Token))
		} else {
			apiClient = api.New(o.APIURL, api.WithBasicAuth(o.Username, o.Password))
		}
		name := "loadtest-" + uuid.NewString()[:8]
		var err error
		group, err = apiClient.AgentGroupService.CreateAgentGroup(ctx, "default", &v1.AgentGroup{
			Metadata: v1.Metadata{Name: name, Namespace: "default"},
			Spec:     v1.Spec{Selector: v1.AgentSelector{IdentifyingAttributes: map[string]string{"service.name": name}}},
		})
		if err != nil {
			return fmt.Errorf("create load-test group: %w", err)
		}
		defer func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := apiClient.AgentGroupService.DeleteAgentGroup(cleanup, "default", name); err != nil {
				log.Printf("delete load-test group: %v", err)
			}
		}()
	}

	serviceName := "loadtest"
	if group != nil {
		serviceName = group.Metadata.Name
	}
	spacing := time.Second / time.Duration(o.Rate)
	if spacing == 0 {
		spacing = time.Nanosecond
	}
	connectTicker := time.NewTicker(spacing)
	defer connectTicker.Stop()
	for i := range agents {
		select {
		case <-ctx.Done():
		case <-connectTicker.C:
		}
		if ctx.Err() != nil {
			break
		}
		a, err := startAgent(ctx, o.URL, serviceName, r)
		if err != nil {
			r.mu.Lock()
			r.connErrors++
			r.mu.Unlock()
			log.Printf("start agent %d: %v", i, err)
			continue
		}
		agents[i] = a
	}
	defer stopAgents(agents)
	log.Printf("started %d agents", o.Count)

	reportTicker := time.NewTicker(o.ReportInterval)
	defer reportTicker.Stop()
	var churn, ops *time.Ticker
	if o.ChurnInterval > 0 {
		churn = time.NewTicker(o.ChurnInterval)
		defer churn.Stop()
	}
	if o.OpsInterval > 0 {
		ops = time.NewTicker(o.OpsInterval)
		defer ops.Stop()
	}
	var churnC, opsC <-chan time.Time
	if churn != nil {
		churnC = churn.C
	}
	if ops != nil {
		opsC = ops.C
	}
	next := 0
	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
		case <-reportTicker.C:
			if o.MetricsURL != "" {
				metrics, errText := scrape(ctx, o.MetricsURL)
				if errText != "" {
					result.MetricsError = errText
				} else {
					result.recordPeak(metrics)
				}
			}
			for _, a := range agents {
				if a != nil {
					if err := a.client.SetHealth(health(a.started)); err != nil {
						log.Printf("report health: %v", err)
					}
				}
			}
		case <-churnC:
			if agents[next] != nil {
				stopAgent(agents[next])
			}
			agents[next], _ = startAgent(ctx, o.URL, serviceName, r)
			next = (next + 1) % len(agents)
		case <-opsC:
			version := uuid.NewString()
			r.mu.Lock()
			r.pushAt[version] = time.Now()
			r.mu.Unlock()
			configName := "loadtest"
			group.Spec.AgentConfig = &v1.AgentConfig{AgentRemoteConfigs: []v1.AgentGroupRemoteConfig{{
				AgentRemoteConfigName: &configName,
				AgentRemoteConfigSpec: &v1.AgentRemoteConfigSpec{Value: "loadtest_version: " + version, ContentType: "application/yaml"},
			}}}
			if _, err := apiClient.AgentGroupService.UpdateAgentGroup(ctx, group); err != nil {
				log.Printf("push config: %v", err)
			} else {
				r.mu.Lock()
				r.pushes++
				r.mu.Unlock()
			}
		}
	}
	if o.MetricsURL != "" {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		var errText string
		result.MetricsEnd, errText = scrape(cleanup, o.MetricsURL)
		if errText != "" {
			result.MetricsError = errText
		}
		result.recordPeak(result.MetricsEnd)
		cancel()
	}
	if mongoClient != nil {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		result.MongoEnd, result.MongoError = mongoCounters(cleanup, mongoClient)
		cancel()
	}
	r.mu.Lock()
	result.ConnectionEvents = r.connected
	result.ConnectionErrors = r.connErrors
	result.ServerErrors = r.serverErrs
	result.ConfigPushes = r.pushes
	result.ConnectLatency = summarize(r.connect)
	result.Propagation = summarize(r.propagate)
	r.mu.Unlock()
	result.DurationSeconds = time.Since(started).Seconds()
	result.ConnectPerSecond = float64(result.ConnectionEvents) / result.DurationSeconds
	if result.MongoStart != nil && result.MongoEnd != nil {
		result.MongoPerSecond = make(map[string]float64)
		for name, before := range result.MongoStart {
			if after := result.MongoEnd[name]; after >= before {
				result.MongoPerSecond[name] = float64(after-before) / result.DurationSeconds
			}
		}
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	if err := os.WriteFile(o.Output, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	log.Printf("wrote %s", o.Output)
	if result.ConnectionEvents < o.Count || result.ConnectionErrors > 0 || result.ServerErrors > 0 ||
		(o.OpsInterval > 0 && (result.ConfigPushes == 0 || result.Propagation.Count == 0)) {
		return errors.New("load scenario failed; see report for connection, server, and propagation errors")
	}
	return nil
}

func (r *report) recordPeak(metrics map[string]float64) {
	r.PeakMemoryBytes = max(r.PeakMemoryBytes, metrics["go_memory_used_bytes"])
	r.PeakGoroutines = max(r.PeakGoroutines, metrics["go_goroutine_count"])
}

func startAgent(ctx context.Context, url, serviceName string, r *recorder) (*agent, error) {
	a := &agent{client: client.NewWebSocket(nil), started: uint64(time.Now().UnixNano()), effective: map[string]*protobufs.AgentConfigObject{}, seen: map[string]bool{}}
	uid := uuid.New()
	started := time.Now()
	if err := a.client.SetAgentDescription(&protobufs.AgentDescription{IdentifyingAttributes: []*protobufs.KeyValue{{
		Key: "service.name", Value: &protobufs.AnyValue{Value: &protobufs.AnyValue_StringValue{StringValue: serviceName}},
	}}}); err != nil {
		return nil, fmt.Errorf("set agent description: %w", err)
	}
	capabilities := protobufs.AgentCapabilities_AgentCapabilities_ReportsStatus |
		protobufs.AgentCapabilities_AgentCapabilities_ReportsHealth |
		protobufs.AgentCapabilities_AgentCapabilities_ReportsHeartbeat |
		protobufs.AgentCapabilities_AgentCapabilities_AcceptsRemoteConfig |
		protobufs.AgentCapabilities_AgentCapabilities_ReportsRemoteConfig |
		protobufs.AgentCapabilities_AgentCapabilities_ReportsEffectiveConfig
	if err := a.client.SetCapabilities(&capabilities); err != nil {
		return nil, fmt.Errorf("set capabilities: %w", err)
	}
	if err := a.client.SetHealth(health(a.started)); err != nil {
		return nil, fmt.Errorf("set health: %w", err)
	}
	heartbeat := 30 * time.Second
	err := a.client.Start(ctx, types.StartSettings{
		OpAMPServerURL: url, InstanceUid: types.InstanceUid(uid), HeartbeatInterval: &heartbeat,
		Callbacks: types.Callbacks{
			OnConnect: func(context.Context) {
				r.mu.Lock()
				r.connected++
				r.connect = append(r.connect, time.Since(started))
				r.mu.Unlock()
			},
			OnConnectFailed: func(_ context.Context, _ error) {
				r.mu.Lock()
				r.connErrors++
				r.mu.Unlock()
			},
			OnError: func(_ context.Context, _ *protobufs.ServerErrorResponse) {
				r.mu.Lock()
				r.serverErrs++
				r.mu.Unlock()
			},
			OnMessage: func(ctx context.Context, msg *types.MessageData) {
				if msg.RemoteConfig == nil {
					return
				}
				a.mu.Lock()
				a.effective = maps.Clone(msg.RemoteConfig.GetConfig().GetConfigMap())
				for _, file := range a.effective {
					version := strings.TrimSpace(strings.TrimPrefix(string(file.GetBody()), "loadtest_version: "))
					if a.seen[version] {
						continue
					}
					r.mu.Lock()
					if at, ok := r.pushAt[version]; ok {
						a.seen[version] = true
						r.propagate = append(r.propagate, time.Since(at))
					}
					r.mu.Unlock()
				}
				a.mu.Unlock()
				_ = a.client.SetRemoteConfigStatus(&protobufs.RemoteConfigStatus{
					LastRemoteConfigHash: msg.RemoteConfig.GetConfigHash(),
					Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
				})
				_ = a.client.UpdateEffectiveConfig(ctx)
			},
			GetEffectiveConfig: func(context.Context) (*protobufs.EffectiveConfig, error) {
				a.mu.Lock()
				defer a.mu.Unlock()
				return &protobufs.EffectiveConfig{ConfigMap: &protobufs.AgentConfigMap{ConfigMap: maps.Clone(a.effective)}}, nil
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("start OpAMP client: %w", err)
	}
	return a, nil
}

func health(started uint64) *protobufs.ComponentHealth {
	now := uint64(time.Now().UnixNano())
	return &protobufs.ComponentHealth{Healthy: true, StartTimeUnixNano: started, StatusTimeUnixNano: now, Status: "OK"}
}

func stopAgent(a *agent) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = a.client.Stop(ctx)
}

func stopAgents(agents []*agent) {
	var wg sync.WaitGroup
	for _, a := range agents {
		if a != nil {
			wg.Go(func() { stopAgent(a) })
		}
	}
	wg.Wait()
}

func summarize(values []time.Duration) summary {
	if len(values) == 0 {
		return summary{}
	}
	ordered := append([]time.Duration(nil), values...)
	slices.Sort(ordered)
	at := func(p float64) float64 {
		return float64(ordered[int(math.Ceil(p*float64(len(ordered))))-1]) / float64(time.Millisecond)
	}
	return summary{Count: len(ordered), P50MS: at(.5), P95MS: at(.95), P99MS: at(.99)}
}

func scrape(ctx context.Context, url string) (map[string]float64, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err.Error()
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err.Error()
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, res.Status
	}
	parser := expfmt.NewTextParser(model.UTF8Validation)
	families, err := parser.TextToMetricFamilies(res.Body)
	if err != nil {
		return nil, err.Error()
	}
	values := make(map[string]float64)
	for name, family := range families {
		for _, metric := range family.GetMetric() {
			if metric.GetGauge() != nil {
				values[name] += metric.GetGauge().GetValue()
			} else if metric.GetCounter() != nil {
				values[name] += metric.GetCounter().GetValue()
			}
		}
	}
	return values, ""
}

func mongoCounters(ctx context.Context, cli *mongo.Client) (map[string]int64, string) {
	var status struct {
		Opcounters struct {
			Query   int64 `bson:"query"`
			Getmore int64 `bson:"getmore"`
			Insert  int64 `bson:"insert"`
			Update  int64 `bson:"update"`
			Delete  int64 `bson:"delete"`
		} `bson:"opcounters"`
	}
	if err := cli.Database("admin").RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}}).Decode(&status); err != nil {
		return nil, err.Error()
	}
	return map[string]int64{
		"query": status.Opcounters.Query, "getmore": status.Opcounters.Getmore,
		"insert": status.Opcounters.Insert, "update": status.Opcounters.Update, "delete": status.Opcounters.Delete,
	}, ""
}
