package integration

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	gosync "sync"
	"testing"

	"github.com/acarl005/stripansi"
	"github.com/blang/semver/v4"
	"github.com/fatih/color"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kong/deck/cmd"
	deckDiff "github.com/kong/go-database-reconciler/pkg/diff"
	deckDump "github.com/kong/go-database-reconciler/pkg/dump"
	"github.com/kong/go-database-reconciler/pkg/file"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/require"
)

func getKongAddress() string {
	address := os.Getenv("DECK_KONG_ADDR")
	if address != "" {
		return address
	}
	return "http://localhost:8001"
}

func getTestClient() (*kong.Client, error) {
	ctx := context.Background()
	controlPlaneName := os.Getenv("DECK_KONNECT_RUNTIME_GROUP_NAME")
	if controlPlaneName == "" {
		controlPlaneName = os.Getenv("DECK_KONNECT_CONTROL_PLANE_NAME")
	}
	konnectConfig := utils.KonnectConfig{
		Address:          os.Getenv("DECK_KONNECT_ADDR"),
		Email:            os.Getenv("DECK_KONNECT_EMAIL"),
		Password:         os.Getenv("DECK_KONNECT_PASSWORD"),
		Token:            os.Getenv("DECK_KONNECT_TOKEN"),
		ControlPlaneName: controlPlaneName,
	}
	if (konnectConfig.Email != "" && konnectConfig.Password != "") || konnectConfig.Token != "" {
		return cmd.GetKongClientForKonnectMode(ctx, &konnectConfig)
	}
	return utils.GetKongClient(utils.KongClientConfig{
		Address: getKongAddress(),
	})
}

func runWhenKonnect(t *testing.T) {
	t.Helper()

	if os.Getenv("DECK_KONNECT_EMAIL") == "" &&
		os.Getenv("DECK_KONNECT_PASSWORD") == "" &&
		os.Getenv("DECK_KONNECT_TOKEN") == "" {
		t.Skip("non-Konnect test instance, skipping")
	}
}

func skipWhenKonnect(t *testing.T) {
	t.Helper()

	if os.Getenv("DECK_KONNECT_EMAIL") != "" ||
		os.Getenv("DECK_KONNECT_PASSWORD") != "" ||
		os.Getenv("DECK_KONNECT_TOKEN") != "" {
		t.Skip("non-Kong test instance, skipping")
	}
}

func runWhenKongOrKonnect(t *testing.T, kongSemverRange string) {
	t.Helper()

	if os.Getenv("DECK_KONNECT_EMAIL") != "" &&
		os.Getenv("DECK_KONNECT_PASSWORD") != "" &&
		os.Getenv("DECK_KONNECT_TOKEN") != "" {
		return
	}
	kong.RunWhenKong(t, kongSemverRange)
}

func runWhenEnterpriseOrKonnect(t *testing.T, kongSemverRange string) {
	t.Helper()

	if os.Getenv("DECK_KONNECT_EMAIL") != "" &&
		os.Getenv("DECK_KONNECT_PASSWORD") != "" &&
		os.Getenv("DECK_KONNECT_TOKEN") != "" {
		return
	}
	kong.RunWhenEnterprise(t, kongSemverRange, kong.RequiredFeatures{})
}

func runWhen(t *testing.T, mode string, semverRange string) {
	t.Helper()

	switch mode {
	case "kong":
		skipWhenKonnect(t)
		kong.RunWhenKong(t, semverRange)
	case "enterprise":
		skipWhenKonnect(t)
		kong.RunWhenEnterprise(t, semverRange, kong.RequiredFeatures{})
	case "konnect":
		runWhenKonnect(t)
	}
}

func sortSlices(x, y interface{}) bool {
	var xName, yName string
	switch xEntity := x.(type) {
	case *kong.Service:
		yEntity := y.(*kong.Service)
		xName = *xEntity.Name
		yName = *yEntity.Name
	case *kong.Route:
		yEntity := y.(*kong.Route)
		xName = *xEntity.Name
		yName = *yEntity.Name
	case *kong.Vault:
		yEntity := y.(*kong.Vault)
		xName = *xEntity.Prefix
		yName = *yEntity.Prefix
	case *kong.Consumer:
		yEntity := y.(*kong.Consumer)
		if xEntity.Username != nil {
			xName = *xEntity.Username
		} else {
			xName = *xEntity.ID
		}
		if yEntity.Username != nil {
			yName = *yEntity.Username
		} else {
			yName = *yEntity.ID
		}
	case *kong.ConsumerGroup:
		yEntity := y.(*kong.ConsumerGroup)
		xName = *xEntity.Name
		yName = *yEntity.Name
	case *kong.ConsumerGroupObject:
		yEntity := y.(*kong.ConsumerGroupObject)
		xName = *xEntity.ConsumerGroup.Name
		yName = *yEntity.ConsumerGroup.Name
	case *kong.ConsumerGroupPlugin:
		yEntity := y.(*kong.ConsumerGroupPlugin)
		xName = *xEntity.ConsumerGroup.ID
		yName = *yEntity.ConsumerGroup.ID
	case *kong.KeyAuth:
		yEntity := y.(*kong.KeyAuth)
		xName = *xEntity.Key
		yName = *yEntity.Key
	case *kong.Plugin:
		yEntity := y.(*kong.Plugin)
		xName = *xEntity.Name
		yName = *yEntity.Name
		if xEntity.Route != nil {
			xName += *xEntity.Route.ID
		}
		if xEntity.Service != nil {
			xName += *xEntity.Service.ID
		}
		if xEntity.Consumer != nil {
			xName += *xEntity.Consumer.ID
		}
		if xEntity.ConsumerGroup != nil {
			xName += *xEntity.ConsumerGroup.ID
		}
		if yEntity.Route != nil {
			yName += *yEntity.Route.ID
		}
		if yEntity.Service != nil {
			yName += *yEntity.Service.ID
		}
		if yEntity.Consumer != nil {
			yName += *yEntity.Consumer.ID
		}
		if yEntity.ConsumerGroup != nil {
			yName += *yEntity.ConsumerGroup.ID
		}
	case *kong.Key:
		yEntity := y.(*kong.Key)
		xName = *xEntity.Name
		yName = *yEntity.Name
	case *kong.KeySet:
		yEntity := y.(*kong.KeySet)
		xName = *xEntity.Name
		yName = *yEntity.Name
	}
	return xName < yName
}

func testKongState(t *testing.T, client *kong.Client, isKonnect bool,
	expectedState utils.KongRawState, ignoreFields []cmp.Option,
) {
	t.Helper()

	// Get entities from Kong
	ctx := context.Background()
	dumpConfig := deckDump.Config{}
	if expectedState.RBACEndpointPermissions != nil {
		dumpConfig.RBACResourcesOnly = true
	}
	if isKonnect {
		controlPlaneName := os.Getenv("DECK_KONNECT_CONTROL_PLANE_NAME")
		if controlPlaneName == "" {
			controlPlaneName = os.Getenv("DECK_KONNECT_CONTROL_PLANE_NAME")
		}
		if controlPlaneName != "" {
			dumpConfig.KonnectControlPlane = controlPlaneName
		} else {
			dumpConfig.KonnectControlPlane = "default"
		}
	}
	kongState, err := deckDump.Get(ctx, client, dumpConfig)
	if err != nil {
		t.Error(err.Error())
	}

	opt := []cmp.Option{
		cmpopts.IgnoreFields(kong.Service{}, "CreatedAt", "UpdatedAt"),
		cmpopts.IgnoreFields(kong.Route{}, "CreatedAt", "UpdatedAt"),
		cmpopts.IgnoreFields(kong.Plugin{}, "ID", "CreatedAt"),
		cmpopts.IgnoreFields(kong.Upstream{}, "ID", "CreatedAt"),
		cmpopts.IgnoreFields(kong.Target{}, "ID", "CreatedAt"),
		cmpopts.IgnoreFields(kong.CACertificate{}, "ID", "CreatedAt"),
		cmpopts.IgnoreFields(kong.RBACEndpointPermission{}, "Role", "CreatedAt"),
		cmpopts.IgnoreFields(kong.RBACRole{}, "ID", "CreatedAt"),
		cmpopts.IgnoreFields(kong.Consumer{}, "ID", "CreatedAt"),
		cmpopts.IgnoreFields(kong.Vault{}, "ID", "CreatedAt", "UpdatedAt"),
		cmpopts.IgnoreFields(kong.Certificate{}, "ID", "CreatedAt"),
		cmpopts.IgnoreFields(kong.SNI{}, "ID", "CreatedAt"),
		cmpopts.IgnoreFields(kong.Consumer{}, "CreatedAt", "ID"),
		cmpopts.IgnoreFields(kong.ConsumerGroup{}, "CreatedAt", "ID"),
		cmpopts.IgnoreFields(kong.ConsumerGroupPlugin{}, "CreatedAt", "ID"),
		cmpopts.IgnoreFields(kong.KeyAuth{}, "ID", "CreatedAt"),
		cmpopts.IgnoreFields(kong.Key{}, "ID", "CreatedAt", "UpdatedAt"),
		cmpopts.IgnoreFields(kong.KeySet{}, "ID", "CreatedAt", "UpdatedAt"),
		cmpopts.SortSlices(sortSlices),
		cmpopts.SortSlices(func(a, b *string) bool { return *a < *b }),
		cmpopts.EquateEmpty(),
	}
	opt = append(opt, ignoreFields...)

	if diff := cmp.Diff(kongState, &expectedState, opt...); diff != "" {
		t.Error(diff)
	}
}

func reset(t *testing.T, opts ...string) {
	deckCmd := cmd.NewRootCmd()
	args := []string{"gateway", "reset", "--force"}
	if len(opts) > 0 {
		args = append(args, opts...)
	}
	deckCmd.SetArgs(args)
	if err := deckCmd.Execute(); err != nil {
		t.Fatal(err.Error(), "failed to reset Kong's state")
	}
}

func readFile(filepath string) (string, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// setup sets deck env variable to prevent analytics in tests and registers reset
// command with t.Cleanup().
//
// NOTE: Can't be called with tests running t.Parallel() because of the usage
// of t.Setenv().
func setup(t *testing.T) {
	// disable analytics for integration tests
	t.Setenv("DECK_ANALYTICS", "off")
	t.Cleanup(func() {
		reset(t)
	})
}

func sync(kongFile string, opts ...string) error {
	deckCmd := cmd.NewRootCmd()
	args := []string{"gateway", "sync", kongFile}
	if len(opts) > 0 {
		args = append(args, opts...)
	}
	deckCmd.SetArgs(args)
	return deckCmd.ExecuteContext(context.Background())
}

func multiFileSync(kongFiles []string, opts ...string) error {
	deckCmd := cmd.NewRootCmd()
	args := []string{"gateway", "sync"}
	args = append(args, kongFiles...)
	if len(opts) > 0 {
		args = append(args, opts...)
	}
	deckCmd.SetArgs(args)
	return deckCmd.ExecuteContext(context.Background())
}

func apply(kongFile string, opts ...string) error {
	deckCmd := cmd.NewRootCmd()
	args := []string{"gateway", "apply", kongFile}
	if len(opts) > 0 {
		args = append(args, opts...)
	}
	deckCmd.SetArgs(args)
	return deckCmd.ExecuteContext(context.Background())
}

func diff(kongFile string, opts ...string) (string, error) {
	deckCmd := cmd.NewRootCmd()
	args := []string{"gateway", "diff", kongFile}
	if len(opts) > 0 {
		args = append(args, opts...)
	}
	deckCmd.SetArgs(args)

	// overwrite default standard output
	r, w, _ := os.Pipe()
	color.Output = w

	// execute decK command
	cmdErr := deckCmd.ExecuteContext(context.Background())

	// read command output
	w.Close()
	out, _ := io.ReadAll(r)

	return stripansi.Strip(string(out)), cmdErr
}

func dump(opts ...string) (string, error) {
	deckCmd := cmd.NewRootCmd()
	args := []string{"gateway", "dump"}
	if len(opts) > 0 {
		args = append(args, opts...)
	}
	deckCmd.SetArgs(args)

	// capture command output to be used during tests
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmdErr := deckCmd.ExecuteContext(context.Background())

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = rescueStdout

	return stripansi.Strip(string(out)), cmdErr
}

func fileLint(opts ...string) (string, error) {
	deckCmd := cmd.NewRootCmd()
	args := []string{"file", "lint"}
	if len(opts) > 0 {
		args = append(args, opts...)
	}
	deckCmd.SetArgs(args)

	// capture command output to be used during tests
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmdErr := deckCmd.ExecuteContext(context.Background())

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = rescueStdout

	return stripansi.Strip(string(out)), cmdErr
}

func ping(opts ...string) error {
	deckCmd := cmd.NewRootCmd()
	args := []string{"gateway", "ping"}
	if len(opts) > 0 {
		args = append(args, opts...)
	}
	deckCmd.SetArgs(args)
	return deckCmd.ExecuteContext(context.Background())
}

func fetchCurrentState(ctx context.Context, client *kong.Client, dumpConfig deckDump.Config) (*state.KongState, error) {
	rawState, err := deckDump.Get(ctx, client, dumpConfig)
	if err != nil {
		return nil, err
	}

	currentState, err := state.Get(rawState)
	if err != nil {
		return nil, err
	}
	return currentState, nil
}

func getKongVersion(ctx context.Context, t *testing.T, client *kong.Client) semver.Version {
	root, err := client.Root(ctx)
	require.NoError(t, err, "Should get no error in getting root endpoint of Kong")
	versionStr := kong.VersionFromInfo(root)
	kv, err := kong.ParseSemanticVersion(versionStr)
	require.NoErrorf(t, err, "failed to parse semantic version from version string %s", versionStr)
	return semver.Version{
		Major: kv.Major(),
		Minor: kv.Minor(),
		Patch: kv.Patch(),
	}
}

// mustResetKongState resets Kong state. Intended to replace `reset` which uses deck command.
func mustResetKongState(ctx context.Context, t *testing.T, client *kong.Client, dumpConfig deckDump.Config) {
	t.Helper()

	emptyRawState := utils.KongRawState{}
	targetState, err := state.Get(&emptyRawState)
	require.NoError(t, err)

	currentState, err := fetchCurrentState(ctx, client, dumpConfig)
	require.NoError(t, err, "failed to fetch current state")

	sc, err := deckDiff.NewSyncer(deckDiff.SyncerOpts{
		CurrentState: currentState,
		TargetState:  targetState,
		KongClient:   client,
	})
	require.NoError(t, err, "failed to create syncer")

	_, errs, _ := sc.Solve(ctx, 1, false, false)
	require.Empty(t, errs, "failed to apply diffs to Kong: %d errors occurred", len(errs))
}

func stateFromFile(
	ctx context.Context, t *testing.T,
	filename string, client *kong.Client, dumpConfig deckDump.Config,
) *state.KongState {
	currentState, err := state.NewKongState()
	require.NoError(t, err, "stateFromFile: failed to build an initial empty KongState")

	targetContent, err := file.GetContentFromFiles([]string{filename}, false)
	require.NoErrorf(t, err, "failed to get file content from file %s", filename)

	rawState, err := file.Get(ctx, targetContent, file.RenderConfig{
		CurrentState: currentState,
		KongVersion:  getKongVersion(ctx, t, client),
	}, dumpConfig, client)
	require.NoError(t, err, "failed to get raw Kong state from client")

	targetState, err := state.Get(rawState)
	require.NoError(t, err, "failed to get KongState from raw state")

	return targetState
}

func logEntityChanges(t *testing.T, stats deckDiff.Stats, entityChanges deckDiff.EntityChanges) {
	for _, creating := range entityChanges.Creating {
		t.Logf("creating %s %s", creating.Kind, creating.Name)
	}
	for _, updating := range entityChanges.Updating {
		t.Logf("updating %s %s", updating.Kind, updating.Name)
	}
	for _, deleting := range entityChanges.Deleting {
		t.Logf("deleting %s %s", deleting.Kind, deleting.Name)
	}
	t.Logf("Summary: %d creates, %d updates, %d deletes",
		stats.CreateOps.Count(),
		stats.UpdateOps.Count(),
		stats.UpdateOps.Count(),
	)
}

// recordRequestProxy is a reverse proxy of Kong gateway admin API endpoints
// to record the request sent to Kong.
type RecordRequestProxy struct {
	lock     gosync.RWMutex
	proxy    *httputil.ReverseProxy
	requests []*http.Request
}

// NewRecordRequestProxy returns a recordRequestProxy sending requests to the target URL.
func NewRecordRequestProxy(target *url.URL) *RecordRequestProxy {
	return &RecordRequestProxy{
		proxy: httputil.NewSingleHostReverseProxy(target),
	}
}

func (p *RecordRequestProxy) addRequest(req *http.Request, bodyContent []byte) {
	p.lock.Lock()
	defer p.lock.Unlock()
	// Create a new reader to replace the body because the original body closes after request sent.
	reader := io.NopCloser(bytes.NewBuffer(bodyContent))
	req.Body = reader
	p.requests = append(p.requests, req)
}

func (p *RecordRequestProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	buf, _ := io.ReadAll(req.Body)
	p.addRequest(req.Clone(context.Background()), buf)
	reader := io.NopCloser(bytes.NewBuffer(buf))
	req.Body = reader
	p.proxy.ServeHTTP(rw, req)
}

func (p *RecordRequestProxy) dumpRequests() []*http.Request {
	p.lock.RLock()
	defer p.lock.RUnlock()
	reqs := make([]*http.Request, 0, len(p.requests))
	for _, req := range p.requests {
		reqs = append(reqs, req.Clone(context.Background()))
	}
	return reqs
}

var _ http.Handler = &RecordRequestProxy{}

type configFactory struct {
	Service                           func(id string, host string, name string) *kong.Service
	Plugin                            func(id string, name string, config kong.Configuration) *kong.Plugin
	RateLimitingConfiguration         func() kong.Configuration
	RateLimitingAdvancedConfiguration func() kong.Configuration
	OpenIDConnectConfiguration        func() kong.Configuration
}

var DefaultConfigFactory = configFactory{
	Service: func(id string, host string, name string) *kong.Service {
		return &kong.Service{
			ID:             kong.String(id),
			Host:           kong.String(host),
			Name:           kong.String(name),
			ConnectTimeout: kong.Int(60000),
			Port:           kong.Int(80),
			Path:           nil,
			Protocol:       kong.String("http"),
			ReadTimeout:    kong.Int(60000),
			Retries:        kong.Int(5),
			WriteTimeout:   kong.Int(60000),
			Tags:           []*string{kong.String("test")},
			Enabled:        kong.Bool(true),
		}
	},
	Plugin: func(id, name string, config kong.Configuration) *kong.Plugin {
		return &kong.Plugin{
			ID:        kong.String(id),
			Name:      kong.String(name),
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
			Config:    config,
		}
	},
	RateLimitingConfiguration: func() kong.Configuration {
		return kong.Configuration{
			"day":                 nil,
			"error_code":          float64(429),
			"error_message":       "API rate limit exceeded",
			"fault_tolerant":      true,
			"header_name":         nil,
			"hide_client_headers": false,
			"hour":                float64(10000),
			"limit_by":            string("consumer"),
			"minute":              nil,
			"month":               nil,
			"path":                nil,
			"policy":              string("redis"),
			"redis": map[string]any{
				"database":    float64(0),
				"host":        string("localhost"),
				"password":    nil,
				"port":        float64(6379),
				"server_name": nil,
				"ssl":         bool(false),
				"ssl_verify":  bool(false),
				"timeout":     float64(2000),
				"username":    nil,
			},
			"redis_database":    float64(0),
			"redis_host":        "localhost",
			"redis_password":    nil,
			"redis_port":        float64(6379),
			"redis_server_name": nil,
			"redis_ssl_verify":  bool(false),
			"redis_ssl":         bool(false),
			"redis_timeout":     float64(2000),
			"redis_username":    nil,
			"second":            nil,
			"sync_rate":         float64(-1),
			"year":              nil,
		}
	},
	RateLimitingAdvancedConfiguration: func() kong.Configuration {
		return kong.Configuration{
			"consumer_groups":         nil,
			"dictionary_name":         string("kong_rate_limiting_counters"),
			"disable_penalty":         bool(false),
			"enforce_consumer_groups": bool(false),
			"error_code":              float64(429),
			"error_message":           "API rate limit exceeded",
			"header_name":             nil,
			"hide_client_headers":     false,
			"identifier":              string("consumer"),
			"limit":                   []any{float64(10)},
			"namespace":               string("ZEz47TWgUrv01HenyQBQa8io06MWsp0L"),
			"path":                    nil,
			"redis": map[string]any{
				"cluster_addresses":        nil,
				"cluster_max_redirections": float64(5),
				"cluster_nodes":            nil,
				"connect_timeout":          float64(2000),
				"connection_is_proxied":    bool(false),
				"database":                 float64(0),
				"host":                     string("127.0.0.5"),
				"keepalive_backlog":        nil,
				"keepalive_pool_size":      float64(256),
				"password":                 nil,
				"port":                     float64(6380),
				"read_timeout":             float64(2000),
				"send_timeout":             float64(2000),
				"sentinel_addresses":       nil,
				"sentinel_master":          string("mymaster"),
				"sentinel_nodes":           nil,
				"sentinel_password":        nil,
				"sentinel_role":            string("master"),
				"sentinel_username":        nil,
				"server_name":              nil,
				"ssl":                      bool(false),
				"ssl_verify":               bool(false),
				"timeout":                  float64(2000),
				"username":                 nil,
			},
			"retry_after_jitter_max": float64(0),
			"strategy":               string("redis"),
			"sync_rate":              float64(10),
			"window_size":            []any{float64(60)},
			"window_type":            string("sliding"),
		}
	},
	OpenIDConnectConfiguration: func() kong.Configuration {
		return kong.Configuration{
			"anonymous":                      nil,
			"audience_claim":                 []any{string("aud")},
			"audience_required":              nil,
			"audience":                       nil,
			"authenticated_groups_claim":     nil,
			"authorization_cookie_http_only": true,
			"authorization_cookie_name":      string("authorization"),
			"authorization_cookie_path":      string("/"),
			"authorization_cookie_domain":    nil,
			"authorization_cookie_same_site": string("Default"),
			"authorization_cookie_secure":    nil,
			"authorization_endpoint":         nil,
			"auth_methods": []any{
				string("password"), string("client_credentials"), string("authorization_code"),
				string("bearer"),
				string("introspection"),
				string("userinfo"),
				string("kong_oauth2"),
				string("refresh_token"),
				string("session"),
			},
			"authorization_query_args_client": nil,
			"authorization_query_args_names":  nil,
			"authorization_query_args_values": nil,
			"authorization_rolling_timeout":   float64(600),
			"cache_introspection":             true,
			"cache_tokens":                    bool(true),
			"cache_tokens_salt":               nil,
			"cache_ttl":                       float64(3600),
			"cache_token_exchange":            true,
			"cache_ttl_max":                   nil,
			"cache_ttl_min":                   nil,
			"cache_ttl_neg":                   nil,
			"cache_ttl_resurrect":             nil,
			"cache_user_info":                 true,
			"claims_forbidden":                nil,
			"client_alg":                      nil,
			"client_arg":                      string("client_id"),
			"client_auth":                     nil,
			"client_credentials_param_type":   []any{string("header"), string("query"), string("body")},
			"client_id":                       nil,
			"client_jwk":                      nil,
			"client_secret":                   nil,
			"cluster_cache_strategy":          string("off"),
			"cluster_cache_redis": map[string]any{
				"username":                 nil,
				"sentinel_master":          nil,
				"sentinel_role":            nil,
				"connect_timeout":          float64(2000),
				"sentinel_nodes":           nil,
				"read_timeout":             float64(2000),
				"sentinel_password":        nil,
				"host":                     string("127.0.0.1"),
				"ssl":                      false,
				"cluster_addresses":        nil,
				"database":                 float64(0),
				"cluster_max_redirections": float64(5),
				"sentinel_addresses":       nil,
				"timeout":                  float64(2000),
				"connection_is_proxied":    false,
				"cluster_nodes":            nil,
				"sentinel_username":        nil,
				"keepalive_pool_size":      float64(256),
				"keepalive_backlog":        nil,
				"port":                     float64(6379),
				"server_name":              nil,
				"password":                 nil,
				"send_timeout":             float64(2000),
				"ssl_verify":               false,
			},
			"consumer_by":                                       []any{string("username"), string("custom_id")},
			"consumer_claim":                                    nil,
			"consumer_optional":                                 false,
			"credential_claim":                                  []any{string("sub")},
			"disable_session":                                   nil,
			"discovery_headers_names":                           nil,
			"discovery_headers_values":                          nil,
			"display_errors":                                    false,
			"domains":                                           nil,
			"downstream_access_token_header":                    nil,
			"downstream_access_token_jwk_header":                nil,
			"downstream_headers_claims":                         nil,
			"downstream_headers_names":                          nil,
			"downstream_id_token_header":                        nil,
			"downstream_id_token_jwk_header":                    nil,
			"downstream_introspection_header":                   nil,
			"downstream_introspection_jwt_header":               nil,
			"downstream_refresh_token_header":                   nil,
			"downstream_session_id_header":                      nil,
			"downstream_user_info_header":                       nil,
			"downstream_user_info_jwt_header":                   nil,
			"dpop_proof_lifetime":                               float64(300),
			"dpop_use_nonce":                                    bool(false),
			"enable_hs_signatures":                              false,
			"end_session_endpoint":                              nil,
			"expose_error_code":                                 true,
			"extra_jwks_uris":                                   nil,
			"forbidden_destroy_session":                         true,
			"forbidden_error_message":                           string("Forbidden"),
			"forbidden_redirect_uri":                            nil,
			"groups_claim":                                      []any{string("groups")},
			"groups_required":                                   nil,
			"hide_credentials":                                  bool(false),
			"http_proxy":                                        nil,
			"http_proxy_authorization":                          nil,
			"http_version":                                      float64(1.1),
			"https_proxy":                                       nil,
			"https_proxy_authorization":                         nil,
			"id_token_param_name":                               nil,
			"id_token_param_type":                               []any{string("header"), string("query"), string("body")},
			"ignore_signature":                                  []any{},
			"introspect_jwt_tokens":                             false,
			"introspection_accept":                              string("application/json"),
			"introspection_check_active":                        true,
			"introspection_endpoint_auth_method":                nil,
			"introspection_endpoint":                            nil,
			"introspection_headers_client":                      nil,
			"introspection_headers_names":                       nil,
			"introspection_headers_values":                      nil,
			"introspection_hint":                                string("access_token"),
			"introspection_post_args_client":                    nil,
			"introspection_post_args_names":                     nil,
			"introspection_post_args_values":                    nil,
			"introspection_token_param_name":                    string("token"),
			"issuer":                                            string("https://accounts.google.test/.well-known/openid-configuration"), //nolint:lll
			"issuers_allowed":                                   nil,
			"keepalive":                                         true,
			"jwt_session_claim":                                 string("sid"),
			"jwt_session_cookie":                                nil,
			"leeway":                                            float64(0),
			"login_action":                                      string("upstream"),
			"login_methods":                                     []any{string("authorization_code")},
			"login_redirect_uri":                                nil,
			"login_redirect_mode":                               string("fragment"),
			"login_tokens":                                      []any{string("id_token")},
			"logout_methods":                                    []any{string("POST"), string("DELETE")},
			"logout_post_arg":                                   nil,
			"logout_query_arg":                                  nil,
			"logout_redirect_uri":                               nil,
			"logout_revoke_refresh_token":                       true,
			"logout_revoke":                                     false,
			"logout_revoke_access_token":                        true,
			"logout_uri_suffix":                                 nil,
			"max_age":                                           nil,
			"mtls_introspection_endpoint":                       nil,
			"mtls_revocation_endpoint":                          nil,
			"mtls_token_endpoint":                               nil,
			"no_proxy":                                          nil,
			"password_param_type":                               []any{string("header"), string("query"), string("body")},
			"preserve_query_args":                               bool(false),
			"proof_of_possession_auth_methods_validation":       true,
			"proof_of_possession_dpop":                          string("off"),
			"proof_of_possession_mtls":                          string("off"),
			"pushed_authorization_request_endpoint_auth_method": nil,
			"pushed_authorization_request_endpoint":             nil,
			"redirect_uri":                                      nil,
			"redis": map[string]any{
				"cluster_addresses":        nil,
				"cluster_max_redirections": 5,
				"cluster_nodes":            nil,
				"connect_timeout":          float64(2000),
				"connection_is_proxied":    bool(false),
				"database":                 float64(0),
				"host":                     string("127.0.0.1"),
				"keepalive_backlog":        nil,
				"keepalive_pool_size":      float64(256),
				"password":                 nil,
				"port":                     float64(6379),
				"prefix":                   nil,
				"read_timeout":             float64(2000),
				"send_timeout":             float64(2000),
				"sentinel_addresses":       nil,
				"sentinel_master":          nil,
				"sentinel_nodes":           nil,
				"sentinel_password":        nil,
				"sentinel_role":            nil,
				"sentinel_username":        nil,
				"server_name":              nil,
				"socket":                   nil,
				"ssl":                      bool(false),
				"ssl_verify":               bool(false),
				"timeout":                  float64(2000),
				"username":                 nil,
			},
			"rediscovery_lifetime":                   float64(30),
			"refresh_token_param_name":               nil,
			"refresh_token_param_type":               []any{string("header"), string("query"), string("body")},
			"refresh_tokens":                         true,
			"require_proof_key_for_code_exchange":    nil,
			"require_pushed_authorization_requests":  nil,
			"require_signed_request_object":          nil,
			"resolve_distributed_claims":             false,
			"response_mode":                          string("query"),
			"response_type":                          []any{string("code")},
			"reverify":                               bool(false),
			"revocation_endpoint_auth_method":        nil,
			"revocation_endpoint":                    nil,
			"revocation_token_param_name":            string("token"),
			"roles_claim":                            []any{string("roles")},
			"roles_required":                         nil,
			"run_on_preflight":                       bool(true),
			"scopes":                                 []any{string("openid")},
			"scopes_claim":                           []any{string("scope")},
			"scopes_required":                        nil,
			"search_user_info":                       bool(false),
			"session_absolute_timeout":               float64(86400),
			"session_audience":                       string("default"),
			"session_cookie_domain":                  nil,
			"session_cookie_http_only":               true,
			"session_cookie_name":                    string("session"),
			"session_cookie_path":                    string("/"),
			"session_cookie_same_site":               string("Lax"),
			"session_cookie_secure":                  nil,
			"session_enforce_same_subject":           bool(false),
			"session_hash_storage_key":               bool(false),
			"session_hash_subject":                   bool(false),
			"session_idling_timeout":                 float64(900),
			"session_memcached_host":                 string("127.0.0.1"),
			"session_memcached_port":                 float64(11211),
			"session_memcached_prefix":               nil,
			"session_memcached_socket":               nil,
			"session_redis_cluster_max_redirections": 5,
			"session_redis_cluster_nodes":            nil,
			"session_redis_connect_timeout":          float64(2000),
			"session_redis_host":                     string("127.0.0.1"),
			"session_redis_port":                     float64(6379),
			"session_redis_prefix":                   nil,
			"session_redis_read_timeout":             float64(2000),
			"session_redis_send_timeout":             float64(2000),
			"session_redis_server_name":              nil,
			"session_redis_socket":                   nil,
			"session_redis_ssl_verify":               false,
			"session_redis_ssl":                      false,
			"session_redis_username":                 nil,
			"session_redis_password":                 nil,
			"session_remember_absolute_timeout":      float64(2592000),
			"session_remember_cookie_name":           string("remember"),
			"session_remember_rolling_timeout":       float64(604800),
			"bearer_token_cookie_name":               nil,
			"bearer_token_param_type":                []any{string("header"), string("query"), string("body")},
			"by_username_ignore_case":                bool(false),
			"session_remember":                       false,
			"session_request_headers":                nil,
			"session_response_headers":               nil,
			"session_rolling_timeout":                float64(3600),
			"session_secret":                         nil,
			"session_storage":                        string("cookie"),
			"session_store_metadata":                 bool(false),
			"ssl_verify":                             false,
			"timeout":                                float64(10000),
			"tls_client_auth_cert_id":                nil,
			"tls_client_auth_ssl_verify":             true,
			"token_cache_key_include_scope":          bool(false),
			"token_endpoint":                         nil,
			"token_endpoint_auth_method":             nil,
			"token_exchange_endpoint":                nil,
			"token_headers_client":                   nil,
			"token_headers_grants":                   nil,
			"token_headers_names":                    nil,
			"token_headers_prefix":                   nil,
			"token_headers_replay":                   nil,
			"token_headers_values":                   nil,
			"token_post_args_client":                 nil,
			"token_post_args_names":                  nil,
			"token_post_args_values":                 nil,
			"unauthorized_destroy_session":           true,
			"unauthorized_error_message":             string("Unauthorized"),
			"unauthorized_redirect_uri":              nil,
			"unexpected_redirect_uri":                nil,
			"upstream_access_token_header":           string("authorization:bearer"),
			"upstream_access_token_jwk_header":       nil,
			"upstream_headers_claims":                nil,
			"upstream_headers_names":                 nil,
			"upstream_id_token_header":               nil,
			"upstream_id_token_jwk_header":           nil,
			"upstream_introspection_header":          nil,
			"upstream_introspection_jwt_header":      nil,
			"upstream_refresh_token_header":          nil,
			"upstream_session_id_header":             nil,
			"upstream_user_info_header":              nil,
			"upstream_user_info_jwt_header":          nil,
			"userinfo_accept":                        string("application/json"),
			"userinfo_endpoint":                      nil,
			"userinfo_headers_client":                nil,
			"userinfo_headers_names":                 nil,
			"userinfo_headers_values":                nil,
			"userinfo_query_args_client":             nil,
			"userinfo_query_args_names":              nil,
			"userinfo_query_args_values":             nil,
			"using_pseudo_issuer":                    bool(false),
			"verify_claims":                          bool(true),
			"verify_nonce":                           true,
			"verify_parameters":                      false,
			"verify_signature":                       true,
		}
	},
}

var DefaultConfigFactory39x = configFactory{
	Service: func(id string, host string, name string) *kong.Service {
		return &kong.Service{
			ID:             kong.String(id),
			Host:           kong.String(host),
			Name:           kong.String(name),
			ConnectTimeout: kong.Int(60000),
			Port:           kong.Int(80),
			Path:           nil,
			Protocol:       kong.String("http"),
			ReadTimeout:    kong.Int(60000),
			Retries:        kong.Int(5),
			WriteTimeout:   kong.Int(60000),
			Tags:           []*string{kong.String("test")},
			Enabled:        kong.Bool(true),
		}
	},
	Plugin: func(id, name string, config kong.Configuration) *kong.Plugin {
		return &kong.Plugin{
			ID:        kong.String(id),
			Name:      kong.String(name),
			Enabled:   kong.Bool(true),
			Protocols: []*string{kong.String("grpc"), kong.String("grpcs"), kong.String("http"), kong.String("https")},
			Config:    config,
		}
	},
	RateLimitingConfiguration: func() kong.Configuration {
		return kong.Configuration{
			"day":                 nil,
			"error_code":          float64(429),
			"error_message":       "API rate limit exceeded",
			"fault_tolerant":      true,
			"header_name":         nil,
			"hide_client_headers": false,
			"hour":                float64(10000),
			"limit_by":            string("consumer"),
			"minute":              nil,
			"month":               nil,
			"path":                nil,
			"policy":              string("redis"),
			"redis": map[string]any{
				"database":    float64(0),
				"host":        string("localhost"),
				"password":    nil,
				"port":        float64(6379),
				"server_name": nil,
				"ssl":         bool(false),
				"ssl_verify":  bool(false),
				"timeout":     float64(2000),
				"username":    nil,
			},
			"redis_database":    float64(0),
			"redis_host":        "localhost",
			"redis_password":    nil,
			"redis_port":        float64(6379),
			"redis_server_name": nil,
			"redis_ssl_verify":  bool(false),
			"redis_ssl":         bool(false),
			"redis_timeout":     float64(2000),
			"redis_username":    nil,
			"second":            nil,
			"sync_rate":         float64(-1),
			"year":              nil,
		}
	},
	RateLimitingAdvancedConfiguration: func() kong.Configuration {
		return kong.Configuration{
			"compound_identifier":     nil,
			"consumer_groups":         nil,
			"dictionary_name":         string("kong_rate_limiting_counters"),
			"disable_penalty":         bool(false),
			"enforce_consumer_groups": bool(false),
			"error_code":              float64(429),
			"error_message":           "API rate limit exceeded",
			"header_name":             nil,
			"hide_client_headers":     false,
			"identifier":              string("consumer"),
			"limit":                   []any{float64(10)},
			"lock_dictionary_name":    string("kong_locks"),
			"namespace":               string("ZEz47TWgUrv01HenyQBQa8io06MWsp0L"),
			"path":                    nil,
			"redis": map[string]any{
				"cluster_addresses":        nil,
				"cluster_max_redirections": float64(5),
				"cluster_nodes":            nil,
				"connect_timeout":          float64(2000),
				"connection_is_proxied":    bool(false),
				"database":                 float64(0),
				"host":                     string("127.0.0.5"),
				"keepalive_backlog":        nil,
				"keepalive_pool_size":      float64(256),
				"password":                 nil,
				"port":                     float64(6380),
				"read_timeout":             float64(2000),
				"redis_proxy_type":         nil,
				"send_timeout":             float64(2000),
				"sentinel_addresses":       nil,
				"sentinel_master":          string("mymaster"),
				"sentinel_nodes":           nil,
				"sentinel_password":        nil,
				"sentinel_role":            string("master"),
				"sentinel_username":        nil,
				"server_name":              nil,
				"ssl":                      bool(false),
				"ssl_verify":               bool(false),
				"timeout":                  float64(2000),
				"username":                 nil,
			},
			"retry_after_jitter_max": float64(0),
			"strategy":               string("redis"),
			"sync_rate":              float64(10),
			"window_size":            []any{float64(60)},
			"window_type":            string("sliding"),
		}
	},
	OpenIDConnectConfiguration: func() kong.Configuration {
		return kong.Configuration{
			"anonymous":                      nil,
			"audience_claim":                 []any{string("aud")},
			"audience_required":              nil,
			"audience":                       nil,
			"authenticated_groups_claim":     nil,
			"authorization_cookie_http_only": true,
			"authorization_cookie_name":      string("authorization"),
			"authorization_cookie_path":      string("/"),
			"authorization_cookie_domain":    nil,
			"authorization_cookie_same_site": string("Default"),
			"authorization_cookie_secure":    nil,
			"authorization_endpoint":         nil,
			"auth_methods": []any{
				string("password"), string("client_credentials"), string("authorization_code"),
				string("bearer"),
				string("introspection"),
				string("userinfo"),
				string("kong_oauth2"),
				string("refresh_token"),
				string("session"),
			},
			"authorization_query_args_client": nil,
			"authorization_query_args_names":  nil,
			"authorization_query_args_values": nil,
			"authorization_rolling_timeout":   float64(600),
			"cache_introspection":             true,
			"cache_tokens":                    bool(true),
			"cache_tokens_salt":               nil,
			"cache_ttl":                       float64(3600),
			"cache_token_exchange":            true,
			"cache_ttl_max":                   nil,
			"cache_ttl_min":                   nil,
			"cache_ttl_neg":                   nil,
			"cache_ttl_resurrect":             nil,
			"cache_user_info":                 true,
			"claims_forbidden":                nil,
			"client_alg":                      nil,
			"client_arg":                      string("client_id"),
			"client_auth":                     nil,
			"client_credentials_param_type":   []any{string("header"), string("query"), string("body")},
			"client_id":                       nil,
			"client_jwk":                      nil,
			"client_secret":                   nil,
			"cluster_cache_strategy":          string("off"),
			"cluster_cache_redis": map[string]any{
				"username":                 nil,
				"sentinel_master":          nil,
				"sentinel_role":            nil,
				"connect_timeout":          float64(2000),
				"sentinel_nodes":           nil,
				"read_timeout":             float64(2000),
				"sentinel_password":        nil,
				"host":                     string("127.0.0.1"),
				"ssl":                      false,
				"cluster_addresses":        nil,
				"database":                 float64(0),
				"cluster_max_redirections": float64(5),
				"sentinel_addresses":       nil,
				"timeout":                  float64(2000),
				"connection_is_proxied":    false,
				"cluster_nodes":            nil,
				"sentinel_username":        nil,
				"keepalive_pool_size":      float64(256),
				"keepalive_backlog":        nil,
				"port":                     float64(6379),
				"server_name":              nil,
				"password":                 nil,
				"send_timeout":             float64(2000),
				"ssl_verify":               false,
			},
			"consumer_by":                                       []any{string("username"), string("custom_id")},
			"consumer_claim":                                    nil,
			"consumer_optional":                                 false,
			"credential_claim":                                  []any{string("sub")},
			"disable_session":                                   nil,
			"discovery_headers_names":                           nil,
			"discovery_headers_values":                          nil,
			"display_errors":                                    false,
			"domains":                                           nil,
			"downstream_access_token_header":                    nil,
			"downstream_access_token_jwk_header":                nil,
			"downstream_headers_claims":                         nil,
			"downstream_headers_names":                          nil,
			"downstream_id_token_header":                        nil,
			"downstream_id_token_jwk_header":                    nil,
			"downstream_introspection_header":                   nil,
			"downstream_introspection_jwt_header":               nil,
			"downstream_refresh_token_header":                   nil,
			"downstream_session_id_header":                      nil,
			"downstream_user_info_header":                       nil,
			"downstream_user_info_jwt_header":                   nil,
			"dpop_proof_lifetime":                               float64(300),
			"dpop_use_nonce":                                    bool(false),
			"enable_hs_signatures":                              false,
			"end_session_endpoint":                              nil,
			"expose_error_code":                                 true,
			"extra_jwks_uris":                                   nil,
			"forbidden_destroy_session":                         true,
			"forbidden_error_message":                           string("Forbidden"),
			"forbidden_redirect_uri":                            nil,
			"groups_claim":                                      []any{string("groups")},
			"groups_required":                                   nil,
			"hide_credentials":                                  bool(false),
			"http_proxy":                                        nil,
			"http_proxy_authorization":                          nil,
			"http_version":                                      float64(1.1),
			"https_proxy":                                       nil,
			"https_proxy_authorization":                         nil,
			"id_token_param_name":                               nil,
			"id_token_param_type":                               []any{string("header"), string("query"), string("body")},
			"ignore_signature":                                  []any{},
			"introspect_jwt_tokens":                             false,
			"introspection_accept":                              string("application/json"),
			"introspection_check_active":                        true,
			"introspection_endpoint_auth_method":                nil,
			"introspection_endpoint":                            nil,
			"introspection_headers_client":                      nil,
			"introspection_headers_names":                       nil,
			"introspection_headers_values":                      nil,
			"introspection_hint":                                string("access_token"),
			"introspection_post_args_client":                    nil,
			"introspection_post_args_client_headers":            nil,
			"introspection_post_args_names":                     nil,
			"introspection_post_args_values":                    nil,
			"introspection_token_param_name":                    string("token"),
			"issuer":                                            string("https://accounts.google.test/.well-known/openid-configuration"), //nolint:lll
			"issuers_allowed":                                   nil,
			"keepalive":                                         true,
			"jwt_session_claim":                                 string("sid"),
			"jwt_session_cookie":                                nil,
			"leeway":                                            float64(0),
			"login_action":                                      string("upstream"),
			"login_methods":                                     []any{string("authorization_code")},
			"login_redirect_uri":                                nil,
			"login_redirect_mode":                               string("fragment"),
			"login_tokens":                                      []any{string("id_token")},
			"logout_methods":                                    []any{string("POST"), string("DELETE")},
			"logout_post_arg":                                   nil,
			"logout_query_arg":                                  nil,
			"logout_redirect_uri":                               nil,
			"logout_revoke_refresh_token":                       true,
			"logout_revoke":                                     false,
			"logout_revoke_access_token":                        true,
			"logout_uri_suffix":                                 nil,
			"max_age":                                           nil,
			"mtls_introspection_endpoint":                       nil,
			"mtls_revocation_endpoint":                          nil,
			"mtls_token_endpoint":                               nil,
			"no_proxy":                                          nil,
			"password_param_type":                               []any{string("header"), string("query"), string("body")},
			"preserve_query_args":                               bool(false),
			"proof_of_possession_auth_methods_validation":       true,
			"proof_of_possession_dpop":                          string("off"),
			"proof_of_possession_mtls":                          string("off"),
			"pushed_authorization_request_endpoint_auth_method": nil,
			"pushed_authorization_request_endpoint":             nil,
			"redirect_uri":                                      nil,
			"redis": map[string]any{
				"cluster_addresses":        nil,
				"cluster_max_redirections": 5,
				"cluster_nodes":            nil,
				"connect_timeout":          float64(2000),
				"connection_is_proxied":    bool(false),
				"database":                 float64(0),
				"host":                     string("127.0.0.1"),
				"keepalive_backlog":        nil,
				"keepalive_pool_size":      float64(256),
				"password":                 nil,
				"port":                     float64(6379),
				"prefix":                   nil,
				"read_timeout":             float64(2000),
				"send_timeout":             float64(2000),
				"sentinel_addresses":       nil,
				"sentinel_master":          nil,
				"sentinel_nodes":           nil,
				"sentinel_password":        nil,
				"sentinel_role":            nil,
				"sentinel_username":        nil,
				"server_name":              nil,
				"socket":                   nil,
				"ssl":                      bool(false),
				"ssl_verify":               bool(false),
				"timeout":                  float64(2000),
				"username":                 nil,
			},
			"rediscovery_lifetime":                   float64(30),
			"refresh_token_param_name":               nil,
			"refresh_token_param_type":               []any{string("header"), string("query"), string("body")},
			"refresh_tokens":                         true,
			"require_proof_key_for_code_exchange":    nil,
			"require_pushed_authorization_requests":  nil,
			"require_signed_request_object":          nil,
			"resolve_distributed_claims":             false,
			"response_mode":                          string("query"),
			"response_type":                          []any{string("code")},
			"reverify":                               bool(false),
			"revocation_endpoint_auth_method":        nil,
			"revocation_endpoint":                    nil,
			"revocation_token_param_name":            string("token"),
			"roles_claim":                            []any{string("roles")},
			"roles_required":                         nil,
			"run_on_preflight":                       bool(true),
			"scopes":                                 []any{string("openid")},
			"scopes_claim":                           []any{string("scope")},
			"scopes_required":                        nil,
			"search_user_info":                       bool(false),
			"session_absolute_timeout":               float64(86400),
			"session_audience":                       string("default"),
			"session_cookie_domain":                  nil,
			"session_cookie_http_only":               true,
			"session_cookie_name":                    string("session"),
			"session_cookie_path":                    string("/"),
			"session_cookie_same_site":               string("Lax"),
			"session_cookie_secure":                  nil,
			"session_enforce_same_subject":           bool(false),
			"session_hash_storage_key":               bool(false),
			"session_hash_subject":                   bool(false),
			"session_idling_timeout":                 float64(900),
			"session_memcached_host":                 string("127.0.0.1"),
			"session_memcached_port":                 float64(11211),
			"session_memcached_prefix":               nil,
			"session_memcached_socket":               nil,
			"session_redis_cluster_max_redirections": 5,
			"session_redis_cluster_nodes":            nil,
			"session_redis_connect_timeout":          float64(2000),
			"session_redis_host":                     string("127.0.0.1"),
			"session_redis_port":                     float64(6379),
			"session_redis_prefix":                   nil,
			"session_redis_read_timeout":             float64(2000),
			"session_redis_send_timeout":             float64(2000),
			"session_redis_server_name":              nil,
			"session_redis_socket":                   nil,
			"session_redis_ssl_verify":               false,
			"session_redis_ssl":                      false,
			"session_redis_username":                 nil,
			"session_redis_password":                 nil,
			"session_remember_absolute_timeout":      float64(2592000),
			"session_remember_cookie_name":           string("remember"),
			"session_remember_rolling_timeout":       float64(604800),
			"bearer_token_cookie_name":               nil,
			"bearer_token_param_type":                []any{string("header"), string("query"), string("body")},
			"by_username_ignore_case":                bool(false),
			"session_remember":                       false,
			"session_request_headers":                nil,
			"session_response_headers":               nil,
			"session_rolling_timeout":                float64(3600),
			"session_secret":                         nil,
			"session_storage":                        string("cookie"),
			"session_store_metadata":                 bool(false),
			"ssl_verify":                             false,
			"timeout":                                float64(10000),
			"tls_client_auth_cert_id":                nil,
			"tls_client_auth_ssl_verify":             true,
			"token_cache_key_include_scope":          bool(false),
			"token_endpoint":                         nil,
			"token_endpoint_auth_method":             nil,
			"token_exchange_endpoint":                nil,
			"token_headers_client":                   nil,
			"token_headers_grants":                   nil,
			"token_headers_names":                    nil,
			"token_headers_prefix":                   nil,
			"token_headers_replay":                   nil,
			"token_headers_values":                   nil,
			"token_post_args_client":                 nil,
			"token_post_args_names":                  nil,
			"token_post_args_values":                 nil,
			"unauthorized_destroy_session":           true,
			"unauthorized_error_message":             string("Unauthorized"),
			"unauthorized_redirect_uri":              nil,
			"unexpected_redirect_uri":                nil,
			"upstream_access_token_header":           string("authorization:bearer"),
			"upstream_access_token_jwk_header":       nil,
			"upstream_headers_claims":                nil,
			"upstream_headers_names":                 nil,
			"upstream_id_token_header":               nil,
			"upstream_id_token_jwk_header":           nil,
			"upstream_introspection_header":          nil,
			"upstream_introspection_jwt_header":      nil,
			"upstream_refresh_token_header":          nil,
			"upstream_session_id_header":             nil,
			"upstream_user_info_header":              nil,
			"upstream_user_info_jwt_header":          nil,
			"userinfo_accept":                        string("application/json"),
			"userinfo_endpoint":                      nil,
			"userinfo_headers_client":                nil,
			"userinfo_headers_names":                 nil,
			"userinfo_headers_values":                nil,
			"userinfo_query_args_client":             nil,
			"userinfo_query_args_names":              nil,
			"userinfo_query_args_values":             nil,
			"using_pseudo_issuer":                    bool(false),
			"verify_claims":                          bool(true),
			"verify_nonce":                           true,
			"verify_parameters":                      false,
			"verify_signature":                       true,
		}
	},
}
