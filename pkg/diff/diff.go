package diff

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/kong/go-database-reconciler/pkg/cprint"
	"github.com/kong/go-database-reconciler/pkg/crud"
	"github.com/kong/go-database-reconciler/pkg/konnect"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-database-reconciler/pkg/types"
	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/kong/go-kong/kong"
)

// ------------------------------------------------------
// Old types used by the direct output diff engine
// ------------------------------------------------------

// TODO https://github.com/Kong/go-database-reconciler/issues/22 Body is an any type field. It is set here
// but apparently never used. It only ever contains the "old"/"new" map with the old and new object from
// the event above. We use the event directly in generateDiffString, so it's not clear what its intended
// purpose was. We should probably do a breaking change to either remove it or change it to a more
// structured type. The latter makes sense if we want downstream to be able to calculate its own diffs
// from structs for whatever reason, e.g. to print a partial diff rather than the complete diff string.

type EntityState struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Body any    `json:"body"`
}

type Summary struct {
	Creating int32 `json:"creating"`
	Updating int32 `json:"updating"`
	Deleting int32 `json:"deleting"`
	Total    int32 `json:"total"`
}

type JSONOutputObject struct {
	Changes  EntityChanges `json:"changes"`
	Summary  Summary       `json:"summary"`
	Warnings []string      `json:"warnings"`
	Errors   []string      `json:"errors"`
}

type EntityChanges struct {
	Creating []EntityState `json:"creating"`
	Updating []EntityState `json:"updating"`
	Deleting []EntityState `json:"deleting"`
}

// ------------------------------------------------------
// New types used by the no output diff engine
// ------------------------------------------------------

// ReconcileAction is an action taken by the diff engine.
type ReconcileAction string

const (
	// CreateAction is the ReconcileAction used when a target object did not exist in the current state and was created.
	CreateAction = ReconcileAction("create")
	// UpdateAction is the ReconcileAction used when a target object exists in the current state and was updated.
	UpdateAction = ReconcileAction("update")
	// DeleteAction is the ReconcileAction used when a current object exists in the target state and was deleted.
	DeleteAction = ReconcileAction("delete")

	// eventBuffer is the number of events to buffer in the various syncer channels.
	eventBuffer = 10
)

// Entity is an entity processed by the diff engine.
type Entity struct {
	// Name is the name of the entity.
	Name string `json:"name"`
	// Kind is the type of entity.
	Kind string `json:"kind"`
	// Old is the original entity in the current state, if any.
	Old any `json:"old,omitempty"`
	// New is the new entity in the target state, if any.
	New any `json:"new,omitempty"`
}

// EntityAction describes an entity processed by the diff engine and the action taken on it.
type EntityAction struct {
	// Action is the ReconcileAction taken on the entity.
	Action ReconcileAction `json:"action"`
	// Entity holds the processed entity.
	Entity Entity `json:"entity"`
	// Diff is diff string describing the modifications made to an entity.
	Diff string `json:"-"`
	// Error is the error encountered processing and entity, if any.
	Error error `json:"error,omitempty"`
}

var errEnqueueFailed = errors.New("failed to queue event")

func defaultBackOff() backoff.BackOff {
	// For various reasons, Kong can temporarily fail to process
	// a valid request (e.g. when the database is under heavy load).
	// We retry each request up to 3 times on failure, after around
	// 1 second, 3 seconds, and 9 seconds (randomized exponential backoff).
	exponentialBackoff := backoff.NewExponentialBackOff()
	exponentialBackoff.InitialInterval = 1 * time.Second
	exponentialBackoff.Multiplier = 3
	return backoff.WithMaxRetries(exponentialBackoff, 4)
}

// Syncer takes in a current and target state of Kong,
// diffs them, generating a Graph to get Kong from current
// to target state.
type Syncer struct {
	currentState *state.KongState
	targetState  *state.KongState

	processor     crud.Registry
	postProcessor crud.Registry

	eventChan  chan crud.Event
	errChan    chan error
	stopChan   chan struct{}
	resultChan chan EntityAction

	inFlightOps int32

	silenceWarnings bool
	stageDelaySec   int

	createPrintln func(a ...interface{})
	updatePrintln func(a ...interface{})
	deletePrintln func(a ...interface{})

	kongClient    *kong.Client
	konnectClient *konnect.Client

	entityDiffers map[types.EntityType]types.Differ

	noMaskValues    bool
	includeLicenses bool

	isKonnect bool

	// enableEntityActions enables entity actions and disables direct output prints. If set to true, clients must
	// consume the Syncer.resultChan channel or Syncer.Solve() will block.
	enableEntityActions bool

	// Prevents the Syncer from performing any Delete operations. Default is false (will delete).
	noDeletes bool

	// schema caching helps in reducing the number of GET calls to Kong Gateway
	pluginSchemasCache  map[string]map[string]interface{}
	partialSchemasCache map[string]map[string]interface{}

	cacheMutex sync.Mutex
}

type SyncerOpts struct {
	CurrentState *state.KongState
	TargetState  *state.KongState

	KongClient    *kong.Client
	KonnectClient *konnect.Client

	SilenceWarnings bool
	StageDelaySec   int

	NoMaskValues    bool
	IncludeLicenses bool

	IsKonnect bool

	CreatePrintln func(a ...interface{})
	UpdatePrintln func(a ...interface{})
	DeletePrintln func(a ...interface{})

	// EnableEntityActions instructs the Syncer to send EntityActions to its resultChan. If enabled, clients must
	// consume the Syncer.resultChan channel or Syncer.Solve() will block.
	EnableEntityActions bool

	// Prevents the Syncer from performing any Delete operations. Default is false (will delete).
	NoDeletes bool
}

// NewSyncer constructs a Syncer.
func NewSyncer(opts SyncerOpts) (*Syncer, error) {
	s := &Syncer{
		currentState: opts.CurrentState,
		targetState:  opts.TargetState,

		kongClient:    opts.KongClient,
		konnectClient: opts.KonnectClient,

		silenceWarnings: opts.SilenceWarnings,
		stageDelaySec:   opts.StageDelaySec,

		noMaskValues: opts.NoMaskValues,

		createPrintln:   opts.CreatePrintln,
		updatePrintln:   opts.UpdatePrintln,
		deletePrintln:   opts.DeletePrintln,
		includeLicenses: opts.IncludeLicenses,
		isKonnect:       opts.IsKonnect,

		enableEntityActions: opts.EnableEntityActions,
		noDeletes:           opts.NoDeletes,
	}

	if opts.IsKonnect {
		s.includeLicenses = false
	}

	if s.createPrintln == nil {
		s.createPrintln = cprint.CreatePrintln
	}
	if s.updatePrintln == nil {
		s.updatePrintln = cprint.UpdatePrintln
	}
	if s.deletePrintln == nil {
		s.deletePrintln = cprint.DeletePrintln
	}

	err := s.init()
	if err != nil {
		return nil, err
	}
	s.resultChan = make(chan EntityAction, eventBuffer)

	s.pluginSchemasCache = make(map[string]map[string]interface{})
	s.partialSchemasCache = make(map[string]map[string]interface{})

	s.cacheMutex = sync.Mutex{}

	return s, nil
}

// GetResultChan returns the Syncer's result channel.
func (sc *Syncer) GetResultChan() chan EntityAction {
	return sc.resultChan
}

func (sc *Syncer) init() error {
	opts := types.EntityOpts{
		CurrentState: sc.currentState,
		TargetState:  sc.targetState,

		KongClient:    sc.kongClient,
		KonnectClient: sc.konnectClient,

		IsKonnect: sc.isKonnect,
	}

	entities := []types.EntityType{
		types.Service, types.Route, types.Plugin,

		types.Certificate, types.SNI, types.CACertificate,

		types.Upstream, types.Target,

		types.Consumer,
		types.ConsumerGroup, types.ConsumerGroupConsumer, types.ConsumerGroupPlugin,
		types.ACLGroup, types.BasicAuth, types.KeyAuth,
		types.HMACAuth, types.JWTAuth, types.OAuth2Cred,
		types.MTLSAuth,

		types.Vault,
		types.License,

		types.RBACRole, types.RBACEndpointPermission,

		types.ServicePackage, types.ServiceVersion, types.Document,

		types.FilterChain,

		types.DegraphqlRoute,

		types.Partial,

		types.Key, types.KeySet,
	}

	sc.entityDiffers = map[types.EntityType]types.Differ{}
	for _, entityType := range entities {
		// Skip licenses if includeLicenses is disabled.
		if !sc.includeLicenses && entityType == types.License {
			continue
		}
		entity, err := types.NewEntity(entityType, opts)
		if err != nil {
			return err
		}
		sc.postProcessor.MustRegister(crud.Kind(entityType), entity.PostProcessActions())
		sc.processor.MustRegister(crud.Kind(entityType), entity.CRUDActions())
		sc.entityDiffers[entityType] = entity.Differ()
	}
	return nil
}

func (sc *Syncer) diff() error {
	var operations []func() error

	// If the syncer is configured to skip deletes, then don't add those functions at all to the list of diff operations.
	if !sc.noDeletes {
		operations = append(operations, sc.deleteDuplicates)
	}

	operations = append(operations, sc.createUpdate)

	if !sc.noDeletes {
		operations = append(operations, sc.delete)
	}

	for _, operation := range operations {
		err := operation()
		if err != nil {
			return err
		}
	}
	return nil
}

func (sc *Syncer) deleteDuplicates() error {
	var events []crud.Event
	for _, ts := range reverseOrder() {
		for _, entityType := range ts {
			entityDiffer, ok := sc.entityDiffers[entityType].(types.DuplicatesDeleter)
			if !ok {
				continue
			}
			entityEvents, err := entityDiffer.DuplicatesDeletes()
			if err != nil {
				return err
			}
			events = append(events, entityEvents...)
		}
	}

	return sc.processDeleteDuplicates(eventsInOrder(events, reverseOrder()))
}

func (sc *Syncer) processDeleteDuplicates(eventsByLevel [][]crud.Event) error {
	// All entities implement this interface. We'll use it to index delete events by (kind, identifier) tuple to prevent
	// deleting a single object twice.
	type identifier interface {
		Identifier() string
	}
	var (
		alreadyDeleted = map[string]struct{}{}
		keyForEvent    = func(event crud.Event) (string, error) {
			obj, ok := event.Obj.(identifier)
			if !ok {
				return "", fmt.Errorf("unexpected type %T in event", event.Obj)
			}
			return fmt.Sprintf("%s-%s", event.Kind, obj.Identifier()), nil
		}
	)

	for _, events := range eventsByLevel {
		for _, event := range events {
			key, err := keyForEvent(event)
			if err != nil {
				return err
			}
			if _, ok := alreadyDeleted[key]; ok {
				continue
			}
			if err := sc.queueEvent(event); err != nil {
				return err
			}
			alreadyDeleted[key] = struct{}{}
		}

		// Wait for all the deletes to finish before moving to the next level to avoid conflicts.
		sc.wait()
	}

	return nil
}

func (sc *Syncer) delete() error {
	for _, typeSet := range reverseOrder() {
		for _, entityType := range typeSet {
			// Skip licenses if includeLicenses is disabled.
			if !sc.includeLicenses && entityType == types.License {
				continue
			}
			err := sc.entityDiffers[entityType].Deletes(sc.queueEvent)
			if err != nil {
				return err
			}
			sc.wait()
		}
	}
	return nil
}

func (sc *Syncer) createUpdate() error {
	for _, typeSet := range order() {
		for _, entityType := range typeSet {
			// Skip licenses if includeLicenses is disabled.
			if !sc.includeLicenses && entityType == types.License {
				continue
			}
			err := sc.entityDiffers[entityType].CreateAndUpdates(sc.queueEvent)
			if err != nil {
				return err
			}
			sc.wait()
		}
	}
	return nil
}

func (sc *Syncer) queueEvent(e crud.Event) error {
	atomic.AddInt32(&sc.inFlightOps, 1)
	select {
	case sc.eventChan <- e:
		return nil
	case <-sc.stopChan:
		atomic.AddInt32(&sc.inFlightOps, -1)
		return errEnqueueFailed
	}
}

func (sc *Syncer) eventCompleted() {
	atomic.AddInt32(&sc.inFlightOps, -1)
}

func (sc *Syncer) wait() {
	time.Sleep(time.Duration(sc.stageDelaySec) * time.Second)
	for atomic.LoadInt32(&sc.inFlightOps) != 0 {
		select {
		case <-sc.stopChan:
			return
		default:
			time.Sleep(1 * time.Millisecond)
		}
	}
}

// Run starts a diff and invokes action for every diff.
func (sc *Syncer) Run(ctx context.Context, parallelism int, action Do) []error {
	if parallelism < 1 {
		return append([]error{}, fmt.Errorf("parallelism can not be negative"))
	}

	var wg sync.WaitGroup

	sc.eventChan = make(chan crud.Event, eventBuffer)
	sc.stopChan = make(chan struct{})
	sc.errChan = make(chan error)

	// run rabbit run
	// start the consumers
	wg.Add(parallelism)
	for i := 0; i < parallelism; i++ {
		go func() {
			err := sc.eventLoop(ctx, action)
			if err != nil {
				sc.errChan <- err
			}
			wg.Done()
		}()
	}

	// start the producer
	wg.Add(1)
	go func() {
		err := sc.diff()
		if err != nil {
			sc.errChan <- err
		}
		close(sc.eventChan)
		wg.Done()
	}()

	// close the error and result chan once all done
	go func() {
		wg.Wait()
		close(sc.errChan)
		close(sc.resultChan)
	}()

	var errs []error
	select {
	case <-ctx.Done():
		errs = append(errs, fmt.Errorf("failed to sync all entities: %w", ctx.Err()))
	case err, ok := <-sc.errChan:
		if ok && err != nil {
			if !errors.Is(err, errEnqueueFailed) {
				errs = append(errs, err)
			}
		}
	}

	// stop the producer
	close(sc.stopChan)

	// collect errors
	for err := range sc.errChan {
		if !errors.Is(err, errEnqueueFailed) {
			errs = append(errs, err)
		}
	}

	return errs
}

// Do is the worker function to sync the diff
type Do func(a crud.Event) (crud.Arg, error)

func (sc *Syncer) eventLoop(ctx context.Context, d Do) error {
	for event := range sc.eventChan {
		// Stop if program is terminated
		select {
		case <-sc.stopChan:
			return nil
		default:
		}

		err := sc.handleEvent(ctx, d, event)
		sc.eventCompleted()
		if err != nil {
			return err
		}
	}
	return nil
}

func (sc *Syncer) handleEvent(ctx context.Context, d Do, event crud.Event) error {
	err := backoff.Retry(func() error {
		res, err := d(event)
		if err != nil {
			err = fmt.Errorf("while processing event: %w", err)

			var kongAPIError *kong.APIError
			if errors.As(err, &kongAPIError) &&
				kongAPIError.Code() == http.StatusInternalServerError {
				// Only retry if the request to Kong returned a 500 status code
				return err
			}

			// Do not retry on other status codes
			return backoff.Permanent(err)
		}
		if res == nil {
			// Do not retry empty responses
			return backoff.Permanent(fmt.Errorf("result of event is nil"))
		}
		_, err = sc.postProcessor.Do(ctx, event.Kind, event.Op, res)
		if err != nil {
			// Do not retry program errors
			return backoff.Permanent(fmt.Errorf("while post processing event: %w", err))
		}
		return nil
	}, defaultBackOff())

	return err
}

// Stats holds the stats related to a Solve.
type Stats struct {
	CreateOps *utils.AtomicInt32Counter
	UpdateOps *utils.AtomicInt32Counter
	DeleteOps *utils.AtomicInt32Counter
}

// Generete Diff output for 'sync' and 'diff' commands
func generateDiffString(e crud.Event, isDelete bool, noMaskValues bool) (string, error) {
	var diffString string
	var err error
	if oldObj, ok := e.OldObj.(*state.Document); ok {
		if !isDelete {
			diffString, err = getDocumentDiff(oldObj, e.Obj.(*state.Document))
		} else {
			diffString, err = getDocumentDiff(e.Obj.(*state.Document), oldObj)
		}
	} else {
		if !isDelete {
			diffString, err = getDiff(e.OldObj, e.Obj)
		} else {
			diffString, err = getDiff(e.Obj, e.OldObj)
		}
	}
	if err != nil {
		return "", err
	}
	if !noMaskValues {
		diffString = MaskEnvVarValue(diffString)
	}
	return diffString, err
}

func (sc *Syncer) getPluginSchema(ctx context.Context, pluginName string) (map[string]interface{}, error) {
	var schema map[string]interface{}

	sc.cacheMutex.Lock()
	defer sc.cacheMutex.Unlock()
	if schema, ok := sc.pluginSchemasCache[pluginName]; ok {
		return schema, nil
	}

	schema, err := sc.kongClient.Plugins.GetFullSchema(ctx, &pluginName)
	if err != nil {
		return schema, err
	}
	sc.pluginSchemasCache[pluginName] = schema
	return schema, nil
}

func (sc *Syncer) getPartialSchema(ctx context.Context, partialName string) (map[string]interface{}, error) {
	var schema map[string]interface{}

	sc.cacheMutex.Lock()
	defer sc.cacheMutex.Unlock()
	if schema, ok := sc.partialSchemasCache[partialName]; ok {
		return schema, nil
	}

	schema, err := sc.kongClient.Partials.GetFullSchema(ctx, &partialName)
	if err != nil {
		return schema, err
	}
	sc.partialSchemasCache[partialName] = schema
	return schema, nil
}

// Solve generates a diff and walks the graph.
func (sc *Syncer) Solve(ctx context.Context, parallelism int, dry bool, isJSONOut bool) (Stats,
	[]error, EntityChanges,
) {
	// TODO https://github.com/Kong/go-database-reconciler/issues/22/
	// this can probably be extracted to clients (only deck uses it) by having clients count events through the result
	// channel, rather than returning them from Solve.
	stats := Stats{
		CreateOps: &utils.AtomicInt32Counter{},
		UpdateOps: &utils.AtomicInt32Counter{},
		DeleteOps: &utils.AtomicInt32Counter{},
	}
	recordOp := func(op crud.Op) {
		switch op {
		case crud.Create:
			stats.CreateOps.Increment(1)
		case crud.Update:
			stats.UpdateOps.Increment(1)
		case crud.Delete:
			if !sc.noDeletes {
				stats.DeleteOps.Increment(1)
			}
		}
	}

	output := EntityChanges{
		Creating: []EntityState{},
		Updating: []EntityState{},
		Deleting: []EntityState{},
	}

	// The length makes it confusing to read, but the code below _isn't being run here_, it's an anon func
	// arg to Run(), which parallelizes it. However, because it's defined in Solve()'s scope, the output created above
	// is available in aggregate and contains most of the content we need already.
	errs := sc.Run(ctx, parallelism, func(e crud.Event) (crud.Arg, error) {
		var err error
		var result crud.Arg

		// This variable holds the original event with the unchanged configuration
		// Below the configuration in `e` may be modified. This is done solely for
		// the purpose of displaying a correct diff and should not affect the
		// configuration that is sent to Kong.
		eventForKong := e

		workspaceExists, err := utils.WorkspaceExists(ctx, sc.kongClient)
		if err != nil {
			return nil, err
		}

		// If the event is for a plugin, inject defaults in the plugin's config
		// that will be used for the diff. This is needed to avoid highlighting
		// default values that were populated by Kong as differences.
		if plugin, ok := e.Obj.(*state.Plugin); ok {
			pluginCopy := &state.Plugin{Plugin: *plugin.DeepCopy()}
			e.Obj = pluginCopy

			if workspaceExists {
				schema, err := sc.getPluginSchema(ctx, *pluginCopy.Plugin.Name)
				if err != nil {
					return nil, err
				}

				linkedPartialConfig, err := utils.FindLinkedPartials(ctx, sc.kongClient, &pluginCopy.Plugin)
				if err != nil {
					return nil, err
				}

				err = kong.FillPluginsDefaultsWithPartials(&pluginCopy.Plugin, schema, linkedPartialConfig)
				if err != nil {
					return nil, fmt.Errorf("failed processing auto fields: %w", err)
				}

				// only fill auto fields for the configuration sent to Kong
				// this is done because we want to avoid Kong to auto generate fields, which
				// would make decK's configuration no longer fully "declarative"
				if err := kong.FillPluginsDefaultsWithOpts(&plugin.Plugin, schema, kong.FillRecordOptions{
					FillDefaults: false,
					FillAuto:     true,
				}); err != nil {
					return nil, fmt.Errorf("failed processing auto fields: %w", err)
				}

				// `oldPlugin` contains both new and deprecated fields.
				// If `plugin` (the new plugin) contains only deprecated fields,
				// we need to remove the new fields from `oldPlugin` to ensure both configurations align correctly.
				if oldPlugin, ok := e.OldObj.(*state.Plugin); ok {
					oldPluginCopy := &state.Plugin{Plugin: *oldPlugin.DeepCopy()}
					e.OldObj = oldPluginCopy
					linkedPartialConfig, err := utils.FindLinkedPartials(ctx, sc.kongClient, &oldPluginCopy.Plugin)
					if err != nil {
						return nil, err
					}

					err = kong.FillPluginsDefaultsWithPartials(&oldPluginCopy.Plugin, schema, linkedPartialConfig)
					if err != nil {
						return nil, fmt.Errorf("failed processing auto fields: %w", err)
					}

					if err := kong.ClearUnmatchingDeprecations(&pluginCopy.Plugin, &oldPluginCopy.Plugin, schema); err != nil {
						return nil, fmt.Errorf("failed processing auto fields: %w", err)
					}
				}
			}
		}

		// If the event is for a partial, inject defaults in the partial's config
		// that will be used for the diff. This is needed to avoid highlighting
		// default values that were populated by Kong as differences.
		if partial, ok := e.Obj.(*state.Partial); ok {
			partialCopy := &state.Partial{Partial: *partial.DeepCopy()}
			e.Obj = partialCopy

			if workspaceExists {
				schema, err := sc.getPartialSchema(ctx, *partialCopy.Partial.Type)
				if err != nil {
					return nil, err
				}

				// fill defaults fields for the configuration that will be used for the diff
				if err := kong.FillPartialDefaults(&partialCopy.Partial, schema); err != nil {
					return nil, fmt.Errorf("failed processing fields for partial: %w", err)
				}
			}
		}

		c := e.Obj.(state.ConsoleString)
		objDiff := map[string]interface{}{
			"old": e.OldObj,
			"new": e.Obj,
		}
		item := EntityState{
			// TODO https://github.com/Kong/go-database-reconciler/issues/22 this is the current (only) place Body is
			// set in an EntityState.
			Body: objDiff,
			Name: c.Console(),
			Kind: string(e.Kind),
		}

		actionResult := EntityAction{
			Entity: Entity{
				Name: c.Console(),
				Kind: string(e.Kind),
				Old:  e.OldObj,
				New:  e.Obj,
			},
		}

		switch e.Op {
		case crud.Create:
			// TODO https://github.com/Kong/go-database-reconciler/issues/22 this currently supports either the entity
			// actions channel or direct console outputs to allow a phased transition to the channel only. Existing console
			// prints and JSON blob building will be moved to the deck client.
			if sc.enableEntityActions {
				actionResult.Action = CreateAction
			} else {
				if isJSONOut {
					output.Creating = append(output.Creating, item)
				} else {
					sc.createPrintln("creating", e.Kind, c.Console())
				}
			}
		case crud.Update:
			diffString, err := generateDiffString(e, false, sc.noMaskValues)
			// TODO https://github.com/Kong/go-database-reconciler/issues/22 this currently supports either the entity
			// actions channel or direct console outputs to allow a phased transition to the channel only. Existing console
			// prints and JSON blob building will be moved to the deck client.
			if sc.enableEntityActions {
				actionResult.Action = UpdateAction
				if err != nil {
					actionResult.Error = err
					select {
					case sc.resultChan <- actionResult:
					case <-ctx.Done():
					}
					return nil, err
				}
			} else {
				if err != nil {
					return nil, err
				}
				if isJSONOut {
					output.Updating = append(output.Updating, item)
				} else {
					sc.updatePrintln("updating", e.Kind, c.Console(), diffString)
				}
			}
		case crud.Delete:
			if !sc.noDeletes {
				// TODO https://github.com/Kong/go-database-reconciler/issues/22 this currently supports either the entity
				// actions channel or direct console outputs to allow a phased transition to the channel only. Existing console
				// prints and JSON blob building will be moved to the deck client.
				if sc.enableEntityActions {
					actionResult.Action = DeleteAction
				} else {
					if isJSONOut {
						output.Deleting = append(output.Deleting, item)
					} else {
						sc.deletePrintln("deleting", e.Kind, c.Console())
					}
				}
			}
		default:
			panic("unknown operation " + e.Op.String())
		}

		if !dry {
			// sync mode
			// fire the request to Kong
			result, err = sc.processor.Do(ctx, eventForKong.Kind, eventForKong.Op, eventForKong)
			// TODO https://github.com/Kong/go-database-reconciler/issues/22 this does not print, but is switched on
			// sc.enableEntityActions because the existing behavior returns a result from the anon Run function.
			// Refactoring should use only the channel and simplify the return, probably to just an error (all the other
			// data will have been sent through the result channel).
			if sc.enableEntityActions {
				actionResult.Error = err
				select {
				case sc.resultChan <- actionResult:
				case <-ctx.Done():
				}
			}
			if err != nil {
				return nil, &crud.ActionError{
					OperationType: e.Op,
					Kind:          e.Kind,
					Name:          c.Console(),
					Err:           err,
				}
			}
		} else {
			// diff mode
			// return the new obj as is but with timestamps zeroed out
			utils.ZeroOutTimestamps(e.Obj)
			utils.ZeroOutTimestamps(e.OldObj)
			// TODO https://github.com/Kong/go-database-reconciler/issues/22 this does not print, but is switched on
			// sc.enableEntityActions because the existing behavior returns a result from the anon Run function.
			// Refactoring should use only the channel and simplify the return, probably to just an error (all the other
			// data will have been sent through the result channel).
			if sc.enableEntityActions {
				select {
				case sc.resultChan <- actionResult:
				case <-ctx.Done():
				}
			}
			result = e.Obj
		}
		// record operation in both: diff and sync commands
		recordOp(e.Op)

		return result, nil
	})
	return stats, errs, output
}
