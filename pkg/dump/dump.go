package dump

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/kong/go-kong/kong"
	"github.com/kong/go-kong/kong/custom"
	"golang.org/x/sync/errgroup"
)

// Config can be used to skip exporting certain entities
type Config struct {
	// If true, only RBAC resources are exported.
	// SkipConsumers and SelectorTags should be falsy when this is set.
	RBACResourcesOnly bool

	// If true, consumers and any plugins associated with it
	// are not exported.
	SkipConsumers bool

	// If true, CA certificates are not exported.
	SkipCACerts bool

	// If true, licenses are exported.
	IncludeLicenses bool

	// CustomEntityTypes lists types of custom entities to list.
	CustomEntityTypes []string

	// SelectorTags can be used to export entities tagged with only specific
	// tags.
	SelectorTags []string

	// LookUpSelectorTags* can be used to ensure state lookup for entities using
	// these tags. This functionality is essential when using a plugin that references
	// consumers or routes associated with tags different from those in the sync command.
	LookUpSelectorTagsConsumerGroups []string
	LookUpSelectorTagsConsumers      []string
	LookUpSelectorTagsRoutes         []string
	LookUpSelectorTagsServices       []string

	// KonnectControlPlane
	KonnectControlPlane string

	// IsConsumerGroupScopedPluginSupported
	IsConsumerGroupScopedPluginSupported bool
}

func deduplicate(stringSlice []string) []string {
	existing := map[string]struct{}{}
	result := []string{}

	for _, s := range stringSlice {
		if _, exists := existing[s]; !exists {
			existing[s] = struct{}{}
			result = append(result, s)
		}
	}

	return result
}

func newOpt(tags []string) *kong.ListOpt {
	opt := new(kong.ListOpt)
	opt.Size = 1000
	opt.Tags = kong.StringSlice(deduplicate(tags)...)
	opt.MatchAllTags = true
	return opt
}

func validateConfig(config Config) error {
	if config.RBACResourcesOnly {
		if config.SkipConsumers {
			return fmt.Errorf("dump: config: SkipConsumer cannot be set when RBACResourcesOnly is set")
		}
		if len(config.SelectorTags) != 0 {
			return fmt.Errorf("dump: config: SelectorTags cannot be set when RBACResourcesOnly is set")
		}
	}
	return nil
}

func getConsumerGroupsConfiguration(ctx context.Context, group *errgroup.Group,
	client *kong.Client, config Config, state *utils.KongRawState,
) {
	group.Go(func() error {
		consumerGroups, err := GetAllConsumerGroups(ctx, client, config.SelectorTags)
		if err != nil {
			if kong.IsNotFoundErr(err) || kong.IsForbiddenErr(err) {
				return nil
			}
			return fmt.Errorf("consumer_groups: %w", err)
		}
		if config.LookUpSelectorTagsConsumerGroups != nil {
			globalConsumerGroups, err := GetAllConsumerGroups(ctx, client, config.LookUpSelectorTagsConsumerGroups)
			if err != nil {
				return fmt.Errorf("error retrieving global consumer groups: %w", err)
			}
			// if globalConsumers are not present, add them.
			for _, globalConsumerGroup := range globalConsumerGroups {
				found := false
				for _, consumerGroup := range consumerGroups {
					if *globalConsumerGroup.ConsumerGroup.ID == *consumerGroup.ConsumerGroup.ID {
						found = true
						break
					}
				}
				if !found {
					consumerGroups = append(consumerGroups, globalConsumerGroup)
				}
			}
		}
		state.ConsumerGroups = consumerGroups
		return nil
	})
}

func getConsumerConfiguration(ctx context.Context, group *errgroup.Group,
	client *kong.Client, config Config, state *utils.KongRawState,
) {
	group.Go(func() error {
		consumers, err := GetAllConsumers(ctx, client, config.SelectorTags)
		if err != nil {
			return fmt.Errorf("consumers: %w", err)
		}
		if config.LookUpSelectorTagsConsumers != nil {
			globalConsumers, err := GetAllConsumers(ctx, client, config.LookUpSelectorTagsConsumers)
			if err != nil {
				return fmt.Errorf("error retrieving global consumers: %w", err)
			}
			// if globalConsumers are not present, add them.
			for _, globalConsumer := range globalConsumers {
				found := false
				for _, consumer := range consumers {
					if *globalConsumer.ID == *consumer.ID {
						found = true
						break
					}
				}
				if !found {
					consumers = append(consumers, globalConsumer)
				}
			}
		}
		state.Consumers = consumers
		return nil
	})

	group.Go(func() error {
		keyAuths, err := GetAllKeyAuths(ctx, client, config.SelectorTags)
		if err != nil {
			return fmt.Errorf("key-auths: %w", err)
		}
		state.KeyAuths = keyAuths
		return nil
	})

	group.Go(func() error {
		hmacAuths, err := GetAllHMACAuths(ctx, client, config.SelectorTags)
		if err != nil {
			return fmt.Errorf("hmac-auths: %w", err)
		}
		state.HMACAuths = hmacAuths
		return nil
	})

	group.Go(func() error {
		jwtAuths, err := GetAllJWTAuths(ctx, client, config.SelectorTags)
		if err != nil {
			return fmt.Errorf("jwts: %w", err)
		}
		state.JWTAuths = jwtAuths
		return nil
	})

	group.Go(func() error {
		basicAuths, err := GetAllBasicAuths(ctx, client, config.SelectorTags)
		if err != nil {
			return fmt.Errorf("basic-auths: %w", err)
		}
		state.BasicAuths = basicAuths
		return nil
	})

	// OAuth2 credentials are not supported in Konnect.
	if config.KonnectControlPlane == "" {
		group.Go(func() error {
			oauth2Creds, err := GetAllOauth2Creds(ctx, client, config.SelectorTags)
			if err != nil {
				return fmt.Errorf("oauth2: %w", err)
			}
			state.Oauth2Creds = oauth2Creds
			return nil
		})
	}

	group.Go(func() error {
		aclGroups, err := GetAllACLGroups(ctx, client, config.SelectorTags)
		if err != nil {
			return fmt.Errorf("acls: %w", err)
		}
		state.ACLGroups = aclGroups
		return nil
	})

	group.Go(func() error {
		// XXX Select-tags based filtering is not performed for mTLS-auth credentials
		// because of the following problems:
		// - We currently do not already tag these credentials, filtering these
		// credentials with tags will break any existing user
		// - this is not a big issue since only mTLS-auth credentials for tagged
		// consumers are exported anyway
		// This feature would only benefit a user who uses tagged consumers but
		// then managed mtls-auth credentials out-of-band. We expect such users
		// to be rare or non-existent.
		mtlsAuths, err := GetAllMTLSAuths(ctx, client, nil)
		if err != nil {
			return fmt.Errorf("mtls-auths: %w", err)
		}
		state.MTLSAuths = mtlsAuths
		return nil
	})
}

func getProxyConfiguration(ctx context.Context, group *errgroup.Group,
	client *kong.Client, config Config, state *utils.KongRawState,
) {
	group.Go(func() error {
		services, err := GetAllServices(ctx, client, config.SelectorTags)
		if err != nil {
			return fmt.Errorf("services: %w", err)
		}
		if config.LookUpSelectorTagsServices != nil {
			globalServices, err := GetAllServices(ctx, client, config.LookUpSelectorTagsServices)
			if err != nil {
				return fmt.Errorf("error retrieving global services: %w", err)
			}
			// if globalServices are not present, add them.
			for _, globalService := range globalServices {
				found := false
				for _, service := range services {
					if *globalService.ID == *service.ID {
						found = true
						break
					}
				}
				if !found {
					services = append(services, globalService)
				}
			}
		}
		state.Services = services
		return nil
	})

	group.Go(func() error {
		routes, err := GetAllRoutes(ctx, client, config.SelectorTags)
		if err != nil {
			return fmt.Errorf("routes: %w", err)
		}
		if config.LookUpSelectorTagsRoutes != nil {
			globalRoutes, err := GetAllRoutes(ctx, client, config.LookUpSelectorTagsRoutes)
			if err != nil {
				return fmt.Errorf("error retrieving global routes: %w", err)
			}
			// if globalRoutes are not present, add them.
			for _, globalRoute := range globalRoutes {
				found := false
				for _, route := range routes {
					if *globalRoute.ID == *route.ID {
						found = true
						break
					}
				}
				if !found {
					routes = append(routes, globalRoute)
				}
			}
		}
		state.Routes = routes
		return nil
	})

	group.Go(func() error {
		plugins, err := GetAllPlugins(ctx, client, config.SelectorTags)
		if err != nil {
			return fmt.Errorf("plugins: %w", err)
		}

		plugins = excludeKonnectManagedPlugins(plugins)

		if config.SkipConsumers {
			plugins = excludeConsumersPlugins(plugins)
			plugins = excludeConsumerGroupsPlugins(plugins)
		}
		state.Plugins = plugins
		return nil
	})

	group.Go(func() error {
		state.FilterChains = make([]*kong.FilterChain, 0)
		filterChains, err := GetAllFilterChains(ctx, client, config.SelectorTags)
		if err != nil {
			var kongErr *kong.APIError
			if errors.As(err, &kongErr) {
				// GET /filter-chains returns:
				//   -> 200 on success
				//   -> 404 if Kong version < 3.4
				//   -> 400 if Kong version >= 3.4 but wasm is not enabled
				if kongErr.Code() == http.StatusNotFound || kongErr.Code() == http.StatusBadRequest {
					return nil
				}
			}
			return fmt.Errorf("filter chains: %w", err)
		}
		state.FilterChains = filterChains
		return nil
	})

	group.Go(func() error {
		certificates, err := GetAllCertificates(ctx, client, config.SelectorTags)
		if err != nil {
			return fmt.Errorf("certificates: %w", err)
		}
		state.Certificates = certificates
		return nil
	})

	if !config.SkipCACerts {
		group.Go(func() error {
			caCerts, err := GetAllCACertificates(ctx, client, config.SelectorTags)
			if err != nil {
				return fmt.Errorf("ca-certificates: %w", err)
			}
			state.CACertificates = caCerts
			return nil
		})
	}

	group.Go(func() error {
		snis, err := GetAllSNIs(ctx, client, config.SelectorTags)
		if err != nil {
			return fmt.Errorf("snis: %w", err)
		}
		state.SNIs = snis
		return nil
	})

	group.Go(func() error {
		upstreams, err := GetAllUpstreams(ctx, client, config.SelectorTags)
		if err != nil {
			return fmt.Errorf("upstreams: %w", err)
		}
		state.Upstreams = upstreams
		targets, err := GetAllTargets(ctx, client, upstreams, config.SelectorTags)
		if err != nil {
			return fmt.Errorf("targets: %w", err)
		}
		state.Targets = targets
		return nil
	})

	group.Go(func() error {
		vaults, err := GetAllVaults(ctx, client, config.SelectorTags)
		if err != nil {
			return fmt.Errorf("vaults: %w", err)
		}
		state.Vaults = vaults
		return nil
	})

	if config.IncludeLicenses {
		group.Go(func() error {
			licenses, err := GetAllLicenses(ctx, client, config.SelectorTags)
			if err != nil {
				return fmt.Errorf("licenses: %w", err)
			}
			state.Licenses = licenses
			return nil
		})
	}

	if len(config.CustomEntityTypes) > 0 {
		// Get custom entities with types given in config.CustomEntityTypes.
		customEntityLock := sync.Mutex{}
		for _, entityType := range config.CustomEntityTypes {
			t := entityType
			group.Go(func() error {
				// Register entity type.
				// Because client writes an unprotected map to register entity types, we need to use mutex to protect it.
				customEntityLock.Lock()
				err := tryRegisterEntityType(client, custom.Type(t))
				customEntityLock.Unlock()
				if err != nil {
					return fmt.Errorf("custom entity %s: %w", t, err)
				}
				// Fetch all entities with the given type.
				entities, err := GetAllCustomEntitiesWithType(ctx, client, t)
				if err != nil {
					return fmt.Errorf("custom entity %s: %w", t, err)
				}
				// Add custom entities to rawstate.
				customEntityLock.Lock()
				state.CustomEntities = append(state.CustomEntities, entities...)
				customEntityLock.Unlock()
				return nil
			})
		}
	}
}

func tryRegisterEntityType(client *kong.Client, typ custom.Type) error {
	if client.Lookup(typ) != nil {
		return nil
	}
	return client.Register(typ, &custom.EntityCRUDDefinition{
		Name:       typ,
		CRUDPath:   "/" + string(typ),
		PrimaryKey: "id",
	})
}

func getEnterpriseRBACConfiguration(ctx context.Context, group *errgroup.Group,
	client *kong.Client, state *utils.KongRawState,
) {
	group.Go(func() error {
		roles, err := GetAllRBACRoles(ctx, client)
		if err != nil {
			return fmt.Errorf("roles: %w", err)
		}
		state.RBACRoles = roles
		return nil
	})

	group.Go(func() error {
		eps, err := GetAllRBACREndpointPermissions(ctx, client)
		if err != nil {
			return fmt.Errorf("eps: %w", err)
		}
		state.RBACEndpointPermissions = eps
		return nil
	})
}

// Get queries all the entities using client and returns
// all the entities in KongRawState.
func Get(ctx context.Context, client *kong.Client, config Config) (*utils.KongRawState, error) {
	var state utils.KongRawState

	if err := validateConfig(config); err != nil {
		return nil, err
	}

	group, ctx := errgroup.WithContext(ctx)

	// dump only rbac resources
	if config.RBACResourcesOnly {
		getEnterpriseRBACConfiguration(ctx, group, client, &state)
	} else {
		// regular case
		getProxyConfiguration(ctx, group, client, config, &state)

		if !config.SkipConsumers {
			getConsumerGroupsConfiguration(ctx, group, client, config, &state)
			getConsumerConfiguration(ctx, group, client, config, &state)
		}
	}

	err := group.Wait()
	if err != nil {
		return nil, err
	}

	return &state, nil
}

// GetAllServices queries Kong for all the services using client.
func GetAllServices(ctx context.Context, client *kong.Client,
	tags []string,
) ([]*kong.Service, error) {
	var services []*kong.Service
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.Services.List(ctx, opt)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		services = append(services, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return services, nil
}

// GetAllRoutes queries Kong for all the routes using client.
func GetAllRoutes(ctx context.Context, client *kong.Client,
	tags []string,
) ([]*kong.Route, error) {
	var routes []*kong.Route
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.Routes.List(ctx, opt)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		routes = append(routes, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return routes, nil
}

// GetAllPlugins queries Kong for all the plugins using client.
func GetAllPlugins(ctx context.Context,
	client *kong.Client, tags []string,
) ([]*kong.Plugin, error) {
	var plugins []*kong.Plugin
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.Plugins.List(ctx, opt)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		plugins = append(plugins, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return plugins, nil
}

// GetAllFilterChains queries Kong for all the filter chains using client.
func GetAllFilterChains(ctx context.Context,
	client *kong.Client, tags []string,
) ([]*kong.FilterChain, error) {
	var filterChains []*kong.FilterChain
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.FilterChains.List(ctx, opt)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		filterChains = append(filterChains, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return filterChains, nil
}

// GetAllCertificates queries Kong for all the certificates using client.
func GetAllCertificates(ctx context.Context, client *kong.Client,
	tags []string,
) ([]*kong.Certificate, error) {
	var certificates []*kong.Certificate
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.Certificates.List(ctx, opt)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, cert := range s {
			c := cert
			c.SNIs = nil
			certificates = append(certificates, cert)
		}
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return certificates, nil
}

// GetAllCACertificates queries Kong for all the CACertificates using client.
func GetAllCACertificates(ctx context.Context,
	client *kong.Client,
	tags []string,
) ([]*kong.CACertificate, error) {
	var caCertificates []*kong.CACertificate
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.CACertificates.List(ctx, opt)
		// Compatibility for Kong < 1.3
		// This core entitiy was not present in the past
		// and the Admin API request will error with 404 Not Found
		// If we do get the error, we return back an empty array of
		// CACertificates, effectively disabling the entity for versions
		// which don't have it.
		// A better solution would be to have a version check, and based
		// on the version, the entities are loaded and synced.
		if err != nil {
			if kong.IsNotFoundErr(err) {
				return caCertificates, nil
			}
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		caCertificates = append(caCertificates, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return caCertificates, nil
}

// GetAllSNIs queries Kong for all the SNIs using client.
func GetAllSNIs(ctx context.Context,
	client *kong.Client, tags []string,
) ([]*kong.SNI, error) {
	var snis []*kong.SNI
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.SNIs.List(ctx, opt)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		snis = append(snis, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return snis, nil
}

// GetAllConsumers queries Kong for all the consumers using client.
// Please use this method with caution if you have a lot of consumers.
func GetAllConsumers(ctx context.Context,
	client *kong.Client, tags []string,
) ([]*kong.Consumer, error) {
	var consumers []*kong.Consumer
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.Consumers.List(ctx, opt)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		consumers = append(consumers, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return consumers, nil
}

// GetAllUpstreams queries Kong for all the Upstreams using client.
func GetAllUpstreams(ctx context.Context,
	client *kong.Client, tags []string,
) ([]*kong.Upstream, error) {
	var upstreams []*kong.Upstream
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.Upstreams.List(ctx, opt)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		upstreams = append(upstreams, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return upstreams, nil
}

// GetAllConsumerGroups queries Kong for all the ConsumerGroups using client.
func GetAllConsumerGroups(ctx context.Context,
	client *kong.Client, tags []string,
) ([]*kong.ConsumerGroupObject, error) {
	var consumerGroupObjects []*kong.ConsumerGroupObject
	opt := newOpt(tags)

	for {
		cgs, nextopt, err := client.ConsumerGroups.List(ctx, opt)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		for _, cg := range cgs {
			r, err := client.ConsumerGroups.Get(ctx, cg.Name)
			if err != nil {
				return nil, err
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			group := &kong.ConsumerGroupObject{
				ConsumerGroup: r.ConsumerGroup,
				Plugins:       r.Plugins,
			}
			consumers := []*kong.Consumer{}
			for _, c := range r.Consumers {
				// if tags are set and if the consumer is not tagged, skip it
				if len(tags) > 0 && !utils.HasTags(c, tags) {
					continue
				}
				consumers = append(consumers, c)
			}
			group.Consumers = consumers
			consumerGroupObjects = append(consumerGroupObjects, group)
		}
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return consumerGroupObjects, nil
}

// GetAllTargets queries Kong for all the Targets of upstreams using client.
// Targets are queries per upstream as there exists no endpoint in Kong
// to list all targets of all upstreams.
func GetAllTargets(ctx context.Context, client *kong.Client,
	upstreams []*kong.Upstream, tags []string,
) ([]*kong.Target, error) {
	var targets []*kong.Target
	opt := newOpt(tags)

	for _, upstream := range upstreams {
		for {
			t, nextopt, err := client.Targets.List(ctx, upstream.ID, opt)
			if err != nil {
				return nil, err
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			targets = append(targets, t...)
			if nextopt == nil {
				break
			}
			opt = nextopt
		}
	}

	return targets, nil
}

// GetAllVaults queries Kong for all the Vaults using client.
func GetAllVaults(
	ctx context.Context, client *kong.Client, tags []string,
) ([]*kong.Vault, error) {
	var vaults []*kong.Vault
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.Vaults.List(ctx, opt)
		if kong.IsNotFoundErr(err) || kong.IsForbiddenErr(err) {
			return vaults, nil
		}
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		vaults = append(vaults, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}

	return vaults, nil
}

// GetAllKeyAuths queries Kong for all key-auth credentials using client.
func GetAllKeyAuths(ctx context.Context,
	client *kong.Client, tags []string,
) ([]*kong.KeyAuth, error) {
	var keyAuths []*kong.KeyAuth
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.KeyAuths.List(ctx, opt)
		if kong.IsNotFoundErr(err) {
			return keyAuths, nil
		}
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		keyAuths = append(keyAuths, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return keyAuths, nil
}

// GetAllHMACAuths queries Kong for all hmac-auth credentials using client.
func GetAllHMACAuths(ctx context.Context,
	client *kong.Client, tags []string,
) ([]*kong.HMACAuth, error) {
	var hmacAuths []*kong.HMACAuth
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.HMACAuths.List(ctx, opt)
		if kong.IsNotFoundErr(err) {
			return hmacAuths, nil
		}
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		hmacAuths = append(hmacAuths, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return hmacAuths, nil
}

// GetAllJWTAuths queries Kong for all jwt credentials using client.
func GetAllJWTAuths(ctx context.Context,
	client *kong.Client, tags []string,
) ([]*kong.JWTAuth, error) {
	var jwtAuths []*kong.JWTAuth
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.JWTAuths.List(ctx, opt)
		if kong.IsNotFoundErr(err) {
			return jwtAuths, nil
		}
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		jwtAuths = append(jwtAuths, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return jwtAuths, nil
}

// GetAllBasicAuths queries Kong for all basic-auth credentials using client.
func GetAllBasicAuths(ctx context.Context,
	client *kong.Client, tags []string,
) ([]*kong.BasicAuth, error) {
	var basicAuths []*kong.BasicAuth
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.BasicAuths.List(ctx, opt)
		if kong.IsNotFoundErr(err) {
			return basicAuths, nil
		}
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		basicAuths = append(basicAuths, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return basicAuths, nil
}

// GetAllOauth2Creds queries Kong for all oauth2 credentials using client.
func GetAllOauth2Creds(ctx context.Context, client *kong.Client,
	tags []string,
) ([]*kong.Oauth2Credential, error) {
	var oauth2Creds []*kong.Oauth2Credential
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.Oauth2Credentials.List(ctx, opt)
		if kong.IsNotFoundErr(err) {
			return oauth2Creds, nil
		}
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		oauth2Creds = append(oauth2Creds, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return oauth2Creds, nil
}

// GetAllACLGroups queries Kong for all ACL groups using client.
func GetAllACLGroups(ctx context.Context,
	client *kong.Client, tags []string,
) ([]*kong.ACLGroup, error) {
	var aclGroups []*kong.ACLGroup
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.ACLs.List(ctx, opt)
		if kong.IsNotFoundErr(err) {
			return aclGroups, nil
		}
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		aclGroups = append(aclGroups, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return aclGroups, nil
}

// GetAllMTLSAuths queries Kong for all basic-auth credentials using client.
func GetAllMTLSAuths(ctx context.Context,
	client *kong.Client, tags []string,
) ([]*kong.MTLSAuth, error) {
	var mtlsAuths []*kong.MTLSAuth
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.MTLSAuths.List(ctx, opt)
		if kong.IsNotFoundErr(err) {
			return mtlsAuths, nil
		}
		if err != nil {
			// TODO figure out a better way to handle unauthorized endpoints
			// per https://github.com/Kong/deck/issues/274 we can't dump these resources
			// from an Enterprise instance running in free mode, and the 403 results in a
			// fatal error when running "deck dump". We don't want to just treat 403s the
			// same as 404s because Kong also uses them to indicate missing RBAC permissions,
			// but this is currently necessary for compatibility. We need a better approach
			// before adding other Enterprise resources that decK handles by default (versus,
			// for example, RBAC roles, which require the --rbac-resources-only flag).
			var kongErr *kong.APIError
			if errors.As(err, &kongErr) {
				if kongErr.Code() == http.StatusForbidden {
					return mtlsAuths, nil
				}
			}
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		mtlsAuths = append(mtlsAuths, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}
	return mtlsAuths, nil
}

// GetAllRBACRoles queries Kong for all the RBACRoles using client.
func GetAllRBACRoles(ctx context.Context,
	client *kong.Client,
) ([]*kong.RBACRole, error) {
	roles, err := client.RBACRoles.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return roles, nil
}

func GetAllRBACREndpointPermissions(ctx context.Context,
	client *kong.Client,
) ([]*kong.RBACEndpointPermission, error) {
	eps := []*kong.RBACEndpointPermission{}
	roles, err := client.RBACRoles.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// retrieve all permissions for the role
	for _, r := range roles {
		reps, err := client.RBACEndpointPermissions.ListAllForRole(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		eps = append(eps, reps...)
	}

	return eps, nil
}

// GetAllLicenses queries Kong for all the Licenses using client.
func GetAllLicenses(
	ctx context.Context, client *kong.Client, tags []string,
) ([]*kong.License, error) {
	var licenses []*kong.License
	opt := newOpt(tags)

	for {
		s, nextopt, err := client.Licenses.List(ctx, opt)
		if kong.IsNotFoundErr(err) {
			return licenses, nil
		}
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		licenses = append(licenses, s...)
		if nextopt == nil {
			break
		}
		opt = nextopt
	}

	return licenses, nil
}

// GetAllCustomEntitiesWithType quries Kong for all Custom entities with the given type.
func GetAllCustomEntitiesWithType(
	ctx context.Context, client *kong.Client, entityType string,
) ([]custom.Entity, error) {
	entities := []custom.Entity{}
	opt := newOpt(nil)
	e := custom.NewEntityObject(custom.Type(entityType))
	for {
		s, nextOpt, err := client.CustomEntities.List(ctx, opt, e)
		if kong.IsNotFoundErr(err) {
			return entities, nil
		}
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		entities = append(entities, s...)
		if nextOpt == nil {
			break
		}
		opt = nextOpt
	}
	return entities, nil
}

// excludeConsumersPlugins filter out consumer plugins
func excludeConsumersPlugins(plugins []*kong.Plugin) []*kong.Plugin {
	var filtered []*kong.Plugin
	for _, p := range plugins {
		if p.Consumer != nil && !utils.Empty(p.Consumer.ID) {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered
}

// excludeConsumerGroupsPlugins filter out consumer-groups plugins
func excludeConsumerGroupsPlugins(plugins []*kong.Plugin) []*kong.Plugin {
	var filtered []*kong.Plugin
	for _, p := range plugins {
		if p.ConsumerGroup != nil && !utils.Empty(p.ConsumerGroup.ID) {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered
}
