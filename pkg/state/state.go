package state

import (
	"fmt"

	memdb "github.com/hashicorp/go-memdb"
)

type collection struct {
	db *memdb.MemDB
}

// KongState is an in-memory database representation
// of Kong's configuration.
type KongState struct {
	common                 collection
	Services               *ServicesCollection
	Routes                 *RoutesCollection
	Upstreams              *UpstreamsCollection
	Targets                *TargetsCollection
	Certificates           *CertificatesCollection
	SNIs                   *SNIsCollection
	CACertificates         *CACertificatesCollection
	Plugins                *PluginsCollection
	FilterChains           *FilterChainsCollection
	Consumers              *ConsumersCollection
	Vaults                 *VaultsCollection
	Licenses               *LicensesCollection
	ConsumerGroups         *ConsumerGroupsCollection
	ConsumerGroupConsumers *ConsumerGroupConsumersCollection
	ConsumerGroupPlugins   *ConsumerGroupPluginsCollection
	Partials               *PartialsCollection
	Keys                   *KeysCollection
	KeySets                *KeySetsCollection

	KeyAuths                *KeyAuthsCollection
	HMACAuths               *HMACAuthsCollection
	JWTAuths                *JWTAuthsCollection
	BasicAuths              *BasicAuthsCollection
	ACLGroups               *ACLGroupsCollection
	Oauth2Creds             *Oauth2CredsCollection
	MTLSAuths               *MTLSAuthsCollection
	DegraphqlRoutes         *DegraphqlRoutesCollection
	RBACRoles               *RBACRolesCollection
	RBACEndpointPermissions *RBACEndpointPermissionsCollection

	// konnect-specific entities
	ServicePackages *ServicePackagesCollection
	ServiceVersions *ServiceVersionsCollection
	Documents       *DocumentsCollection
}

// NewKongState creates a new in-memory KongState.
func NewKongState() (*KongState, error) {
	// TODO FIXME clean up the mess
	keyAuthTemp := newKeyAuthsCollection(collection{})
	hmacAuthTemp := newHMACAuthsCollection(collection{})
	basicAuthTemp := newBasicAuthsCollection(collection{})
	jwtAuthTemp := newJWTAuthsCollection(collection{})
	oauth2CredsTemp := newOauth2CredsCollection(collection{})
	mtlsAuthTemp := newMTLSAuthsCollection(collection{})
	degraphqlRouteTemp := newDegraphqlRoutesCollection(collection{})

	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			serviceTableName:                serviceTableSchema,
			routeTableName:                  routeTableSchema,
			upstreamTableName:               upstreamTableSchema,
			targetTableName:                 targetTableSchema,
			certificateTableName:            certificateTableSchema,
			sniTableName:                    sniTableSchema,
			caCertTableName:                 caCertTableSchema,
			pluginTableName:                 pluginTableSchema,
			filterChainTableName:            filterChainTableSchema,
			consumerTableName:               consumerTableSchema,
			consumerGroupTableName:          consumerGroupTableSchema,
			consumerGroupConsumerTableName:  consumerGroupConsumerTableSchema,
			consumerGroupPluginTableName:    consumerGroupPluginTableSchema,
			rbacRoleTableName:               rbacRoleTableSchema,
			rbacEndpointPermissionTableName: rbacEndpointPermissionTableSchema,
			vaultTableName:                  vaultTableSchema,
			licenseTableName:                licenseTableSchema,
			partialTableName:                partialTableSchema,
			keyTableName:                    keyTableSchema,
			keySetTableName:                 keySetTableSchema,

			degraphqlRouteTemp.TableName(): degraphqlRouteTemp.Schema(),

			keyAuthTemp.TableName():     keyAuthTemp.Schema(),
			hmacAuthTemp.TableName():    hmacAuthTemp.Schema(),
			basicAuthTemp.TableName():   basicAuthTemp.Schema(),
			jwtAuthTemp.TableName():     jwtAuthTemp.Schema(),
			oauth2CredsTemp.TableName(): oauth2CredsTemp.Schema(),
			mtlsAuthTemp.TableName():    mtlsAuthTemp.Schema(),

			aclGroupTableName: aclGroupTableSchema,

			// konnect-specific entities
			servicePackageTableName: servicePackageTableSchema,
			serviceVersionTableName: serviceVersionTableSchema,
			documentTableName:       documentTableSchema,
		},
	}

	memDB, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, fmt.Errorf("creating new ServiceCollection: %w", err)
	}
	var state KongState
	state.common = collection{
		db: memDB,
	}

	state.Services = (*ServicesCollection)(&state.common)
	state.Routes = (*RoutesCollection)(&state.common)
	state.Upstreams = (*UpstreamsCollection)(&state.common)
	state.Targets = (*TargetsCollection)(&state.common)
	state.Certificates = (*CertificatesCollection)(&state.common)
	state.SNIs = (*SNIsCollection)(&state.common)
	state.CACertificates = (*CACertificatesCollection)(&state.common)
	state.Plugins = (*PluginsCollection)(&state.common)
	state.FilterChains = (*FilterChainsCollection)(&state.common)
	state.Consumers = (*ConsumersCollection)(&state.common)
	state.ConsumerGroups = (*ConsumerGroupsCollection)(&state.common)
	state.ConsumerGroupConsumers = (*ConsumerGroupConsumersCollection)(&state.common)
	state.ConsumerGroupPlugins = (*ConsumerGroupPluginsCollection)(&state.common)
	state.RBACRoles = (*RBACRolesCollection)(&state.common)
	state.RBACEndpointPermissions = (*RBACEndpointPermissionsCollection)(&state.common)
	state.Vaults = (*VaultsCollection)(&state.common)
	state.Licenses = (*LicensesCollection)(&state.common)
	state.Partials = (*PartialsCollection)(&state.common)
	state.Keys = (*KeysCollection)(&state.common)
	state.KeySets = (*KeySetsCollection)(&state.common)

	state.DegraphqlRoutes = newDegraphqlRoutesCollection(state.common)

	state.KeyAuths = newKeyAuthsCollection(state.common)
	state.HMACAuths = newHMACAuthsCollection(state.common)
	state.BasicAuths = newBasicAuthsCollection(state.common)
	state.JWTAuths = newJWTAuthsCollection(state.common)
	state.Oauth2Creds = newOauth2CredsCollection(state.common)
	state.MTLSAuths = newMTLSAuthsCollection(state.common)

	state.ACLGroups = (*ACLGroupsCollection)(&state.common)

	// konnect-specific entities
	state.ServicePackages = (*ServicePackagesCollection)(&state.common)
	state.ServiceVersions = (*ServiceVersionsCollection)(&state.common)
	state.Documents = (*DocumentsCollection)(&state.common)

	return &state, nil
}
