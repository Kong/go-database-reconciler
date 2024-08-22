//nolint:deadcode
package integration

import (
	"context"
	"io"
	"os"
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
		cmpopts.IgnoreFields(kong.ConsumerGroup{}, "CreatedAt", "ID"),
		cmpopts.IgnoreFields(kong.ConsumerGroupPlugin{}, "CreatedAt", "ID"),
		cmpopts.IgnoreFields(kong.KeyAuth{}, "ID", "CreatedAt"),
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
	args := []string{"reset", "--force"}
	if len(opts) > 0 {
		args = append(args, opts...)
	}
	deckCmd.SetArgs(args)
	if err := deckCmd.Execute(); err != nil {
		t.Fatalf(err.Error(), "failed to reset Kong's state")
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
	args := []string{"sync", "-s", kongFile}
	if len(opts) > 0 {
		args = append(args, opts...)
	}
	deckCmd.SetArgs(args)
	return deckCmd.ExecuteContext(context.Background())
}

func diff(kongFile string, opts ...string) (string, error) {
	deckCmd := cmd.NewRootCmd()
	args := []string{"diff", "-s", kongFile}
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
	args := []string{"dump"}
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

func lint(opts ...string) (string, error) {
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
	require.NoErrorf(t, err, "failed to parse semantic version from version string", versionStr)
	return semver.Version{
		Major: kv.Major(),
		Minor: kv.Minor(),
		Patch: kv.Patch(),
	}
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
