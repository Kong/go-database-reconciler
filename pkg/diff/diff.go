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
	"github.com/kong/go-database-reconciler/pkg/crud"
	"github.com/kong/go-database-reconciler/pkg/konnect"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-database-reconciler/pkg/types"
	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/kong/go-kong/kong"
)

type EntityState struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Body any    `json:"body"`
	Diff string `json:"-"`
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

	eventChan chan crud.Event
	errChan   chan error
	stopChan  chan struct{}

	inFlightOps int32

	silenceWarnings bool
	stageDelaySec   int

	kongClient    *kong.Client
	konnectClient *konnect.Client

	entityDiffers map[types.EntityType]types.Differ

	noMaskValues bool

	isKonnect bool

	eventLog      EntityChanges
	eventLogMutex sync.Mutex
}

type SyncerOpts struct {
	CurrentState *state.KongState
	TargetState  *state.KongState

	KongClient    *kong.Client
	KonnectClient *konnect.Client

	SilenceWarnings bool
	StageDelaySec   int

	NoMaskValues bool

	IsKonnect bool
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

		isKonnect: opts.IsKonnect,
	}

	err := s.init()
	if err != nil {
		return nil, err
	}

	return s, nil
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

		types.RBACRole, types.RBACEndpointPermission,

		types.ServicePackage, types.ServiceVersion, types.Document,
	}
	sc.entityDiffers = map[types.EntityType]types.Differ{}
	for _, entityType := range entities {
		entity, err := types.NewEntity(entityType, opts)
		if err != nil {
			return err
		}
		sc.postProcessor.MustRegister(crud.Kind(entityType), entity.PostProcessActions())
		sc.processor.MustRegister(crud.Kind(entityType), entity.CRUDActions())
		sc.entityDiffers[entityType] = entity.Differ()
	}

	sc.eventLog = EntityChanges{
		Creating: []EntityState{},
		Updating: []EntityState{},
		Deleting: []EntityState{},
	}

	return nil
}

func (sc *Syncer) diff() error {
	for _, operation := range []func() error{
		sc.deleteDuplicates,
		sc.createUpdate,
		sc.delete,
	} {
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
	for _, types := range reverseOrder() {
		for _, entityType := range types {
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
	for _, types := range order() {
		for _, entityType := range types {
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

// LogCreate adds a create action to the event log.
func (sc *Syncer) LogCreate(state EntityState) {
	sc.eventLogMutex.Lock()
	defer sc.eventLogMutex.Unlock()
	sc.eventLog.Creating = append(sc.eventLog.Creating, state)
}

// LogUpdate adds an update action to the event log.
func (sc *Syncer) LogUpdate(state EntityState) {
	sc.eventLogMutex.Lock()
	defer sc.eventLogMutex.Unlock()
	sc.eventLog.Updating = append(sc.eventLog.Updating, state)
}

// LogDelete adds a delete action to the event log.
func (sc *Syncer) LogDelete(state EntityState) {
	sc.eventLogMutex.Lock()
	defer sc.eventLogMutex.Unlock()
	sc.eventLog.Deleting = append(sc.eventLog.Deleting, state)
}

// GetEventLog returns the syncer event log.
func (sc *Syncer) GetEventLog() EntityChanges {
	sc.eventLogMutex.Lock()
	defer sc.eventLogMutex.Unlock()
	return sc.eventLog
}

// Run starts a diff and invokes d for every diff.
func (sc *Syncer) Run(ctx context.Context, parallelism int, action Do) []error {
	if parallelism < 1 {
		return append([]error{}, fmt.Errorf("parallelism can not be negative"))
	}

	var wg sync.WaitGroup
	const eventBuffer = 10

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

	// close the error chan once all done
	go func() {
		wg.Wait()
		close(sc.errChan)
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

// NOTE TRC part of the return path
func (sc *Syncer) eventLoop(ctx context.Context, action Do) error {
	for event := range sc.eventChan {
		// Stop if program is terminated
		select {
		case <-sc.stopChan:
			return nil
		default:
		}

		err := sc.handleEvent(ctx, action, event)
		sc.eventCompleted()
		if err != nil {
			return err
		}
	}
	return nil
}

// NOTE TRC part of the return path. this actually runs the
func (sc *Syncer) handleEvent(ctx context.Context, action Do, event crud.Event) error {
	err := backoff.Retry(func() error {
		res, err := action(event)
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

// NOTE TRC Solve is the entry point for both the local and konnect sync commands
// those command functions already output other text fwiw
// although they can iterate over a returned op set and print info for each, this does
// introduce a delay in output. Solve() currently prints each action as it takes them,
// whereas the returned set would be printed in a batch at the end. unsure if this should
// matter in practice, but probably not. we could introduce an event channel and separate
// goroutine to handle synch output, but probably not worth it

// Solve generates a diff and walks the graph.
func (sc *Syncer) Solve(ctx context.Context, parallelism int, dry bool, isJSONOut bool) (Stats,
	[]error, EntityChanges,
) {
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
			stats.DeleteOps.Increment(1)
		}
	}

	// NOTE TRC the length makes it confusing to read, but the code below _isn't being run here_, it's an anon func
	// arg to Run(), which parallelizes it. However, because it's defined in Solve()'s scope, the output created above
	// is available in aggregate and contains most of the content we need already
	errs := sc.Run(ctx, parallelism, func(e crud.Event) (crud.Arg, error) {
		var err error
		var result crud.Arg

		c := e.Obj.(state.ConsoleString)
		objDiff := map[string]interface{}{
			"old": e.OldObj,
			"new": e.Obj,
		}
		item := EntityState{
			Body: objDiff,
			// NOTE TRC currently used directly
			Name: c.Console(),
			// NOTE TRC current prints use the kind directly, but it doesn't matter, it's just a string alias anyway
			Kind: string(e.Kind),
		}
		// NOTE TRC currently we emit lines here, need to collect objects instead
		switch e.Op {
		case crud.Create:
			sc.LogCreate(item)
		case crud.Update:
			// TODO TRC this is not currently available in the item EntityState
			diffString, err := generateDiffString(e, false, sc.noMaskValues)
			if err != nil {
				return nil, err
			}
			item.Diff = diffString
			sc.LogUpdate(item)
		case crud.Delete:
			sc.LogDelete(item)
		default:
			panic("unknown operation " + e.Op.String())
		}

		if !dry {
			// sync mode
			// fire the request to Kong
			result, err = sc.processor.Do(ctx, e.Kind, e.Op, e)
			if err != nil {
				return nil, fmt.Errorf("%v %v %v failed: %w", e.Op, e.Kind, c.Console(), err)
			}
		} else {
			// diff mode
			// return the new obj as is but with timestamps zeroed out
			utils.ZeroOutTimestamps(e.Obj)
			utils.ZeroOutTimestamps(e.OldObj)
			result = e.Obj
		}
		// record operation in both: diff and sync commands
		recordOp(e.Op)

		// TODO TRC our existing return is a complete object and error. probably need to return some sort of processed
		// event struct
		return result, nil
	})
	return stats, errs, sc.GetEventLog()
}
