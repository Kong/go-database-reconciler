package state

import (
	"errors"
	"fmt"

	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/kong/go-kong/kong"
)

// Get builds a KongState from a raw representation of Kong.
func Get(raw *utils.KongRawState) (*KongState, error) {
	kongState, err := NewKongState()
	if err != nil {
		return nil, fmt.Errorf("creating new in-memory state of Kong: %w", err)
	}
	err = buildKong(kongState, raw)
	if err != nil {
		return nil, err
	}
	return kongState, nil
}

func ensureService(kongState *KongState, serviceID string) (bool, *kong.Service, error) {
	s, err := kongState.Services.Get(serviceID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("looking up service %q: %w", serviceID, err)

	}
	return true, utils.GetServiceReference(s.Service), nil
}

func ensureRoute(kongState *KongState, routeID string) (bool, *kong.Route, error) {
	r, err := kongState.Routes.Get(routeID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("looking up route %q: %w", routeID, err)

	}
	return true, utils.GetRouteReference(r.Route), nil
}

func ensureConsumer(kongState *KongState, consumerID string) (bool, *kong.Consumer, error) {
	c, err := kongState.Consumers.GetByIDOrUsername(consumerID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("looking up consumer %q: %w", consumerID, err)

	}
	return true, utils.GetConsumerReference(c.Consumer), nil
}

func ensureConsumerGroup(kongState *KongState, consumerGroupID string) (bool, *kong.ConsumerGroup, error) {
	c, err := kongState.ConsumerGroups.Get(consumerGroupID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("looking up consumer-group %q: %w", consumerGroupID, err)

	}
	return true, utils.GetConsumerGroupReference(c.ConsumerGroup), nil
}

func ensurePartial(kongState *KongState, partialID string) (bool, *kong.Partial, error) {
	p, err := kongState.Partials.Get(partialID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("looking up partial %q: %w", partialID, err)

	}
	return true, utils.GetPartialReference(p.Partial), nil
}

func buildKong(kongState *KongState, raw *utils.KongRawState) error {
	for _, s := range raw.Services {
		err := kongState.Services.Add(Service{Service: *s})
		if err != nil {
			return fmt.Errorf("inserting service into state: %w", err)
		}
	}
	for _, r := range raw.Routes {
		if r.Service != nil && !utils.Empty(r.Service.ID) {
			ok, s, err := ensureService(kongState, *r.Service.ID)
			if err != nil {
				return err
			}
			if ok {
				r.Service = s
			}
		}
		err := kongState.Routes.Add(Route{Route: *r})
		if err != nil {
			return fmt.Errorf("inserting route into state: %w", err)
		}
	}
	for _, c := range raw.Consumers {
		err := kongState.Consumers.Add(Consumer{Consumer: *c})
		if err != nil {
			return fmt.Errorf("inserting consumer into state: %w", err)
		}
	}
	for _, cg := range raw.ConsumerGroups {
		err := kongState.ConsumerGroups.Add(ConsumerGroup{ConsumerGroup: *cg.ConsumerGroup})
		if err != nil {
			return fmt.Errorf("inserting consumer group into state: %w", err)
		}
		utils.ZeroOutTimestamps(cg.ConsumerGroup)
		for _, c := range cg.Consumers {
			err := kongState.ConsumerGroupConsumers.Add(
				ConsumerGroupConsumer{
					ConsumerGroupConsumer: kong.ConsumerGroupConsumer{
						Consumer: c, ConsumerGroup: cg.ConsumerGroup,
					},
				},
			)
			if err != nil {
				return fmt.Errorf("inserting consumer group consumer into state: %w", err)
			}
		}

		for _, p := range cg.Plugins {
			err := kongState.ConsumerGroupPlugins.Add(
				ConsumerGroupPlugin{
					ConsumerGroupPlugin: kong.ConsumerGroupPlugin{
						ID:            p.ID,
						Name:          p.Name,
						Config:        p.Config,
						ConsumerGroup: cg.ConsumerGroup,
						ConfigSource:  p.ConfigSource,
						Tags:          p.Tags,
					},
				},
			)
			if err != nil {
				return fmt.Errorf("inserting consumer group plugin into state: %w", err)
			}
		}
	}
	for _, cred := range raw.KeyAuths {
		ok, c, err := ensureConsumer(kongState, *cred.Consumer.ID)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		cred.Consumer = c
		err = kongState.KeyAuths.Add(KeyAuth{KeyAuth: *cred})
		if err != nil {
			return fmt.Errorf("inserting key-auth into state: %w", err)
		}
	}
	for _, cred := range raw.HMACAuths {
		ok, c, err := ensureConsumer(kongState, *cred.Consumer.ID)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		cred.Consumer = c
		err = kongState.HMACAuths.Add(HMACAuth{HMACAuth: *cred})
		if err != nil {
			return fmt.Errorf("inserting hmac-auth into state: %w", err)
		}
	}
	for _, cred := range raw.JWTAuths {
		ok, c, err := ensureConsumer(kongState, *cred.Consumer.ID)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		cred.Consumer = c
		err = kongState.JWTAuths.Add(JWTAuth{JWTAuth: *cred})
		if err != nil {
			return fmt.Errorf("inserting jwt into state: %w", err)
		}
	}
	for _, cred := range raw.BasicAuths {
		ok, c, err := ensureConsumer(kongState, *cred.Consumer.ID)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		cred.Consumer = c
		err = kongState.BasicAuths.Add(BasicAuth{BasicAuth: *cred})
		if err != nil {
			return fmt.Errorf("inserting basic-auth into state: %w", err)
		}
	}
	for _, cred := range raw.Oauth2Creds {
		ok, c, err := ensureConsumer(kongState, *cred.Consumer.ID)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		cred.Consumer = c
		err = kongState.Oauth2Creds.Add(Oauth2Credential{Oauth2Credential: *cred})
		if err != nil {
			return fmt.Errorf("inserting oauth2-cred into state: %w", err)
		}
	}
	for _, cred := range raw.ACLGroups {
		ok, c, err := ensureConsumer(kongState, *cred.Consumer.ID)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		cred.Consumer = c
		err = kongState.ACLGroups.Add(ACLGroup{ACLGroup: *cred})
		if err != nil {
			return fmt.Errorf("inserting basic-auth into state: %w", err)
		}
	}
	for _, cred := range raw.MTLSAuths {
		ok, c, err := ensureConsumer(kongState, *cred.Consumer.ID)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		cred.Consumer = c
		err = kongState.MTLSAuths.Add(MTLSAuth{MTLSAuth: *cred})
		if err != nil {
			return fmt.Errorf("inserting mtls-auth into state: %w", err)
		}
	}
	for _, u := range raw.Upstreams {
		err := kongState.Upstreams.Add(Upstream{Upstream: *u})
		if err != nil {
			return fmt.Errorf("inserting upstream into state: %w", err)
		}
	}
	for _, t := range raw.Targets {
		err := kongState.Targets.Add(Target{Target: *t})
		if err != nil {
			return fmt.Errorf("inserting target into state: %w", err)
		}
	}

	for _, c := range raw.Certificates {
		err := kongState.Certificates.Add(Certificate{Certificate: *c})
		if err != nil {
			return fmt.Errorf("inserting certificate into state: %w", err)
		}
	}

	for _, s := range raw.SNIs {
		err := kongState.SNIs.Add(SNI{SNI: *s})
		if err != nil {
			return fmt.Errorf("inserting sni into state: %w", err)
		}
	}

	for _, c := range raw.CACertificates {
		err := kongState.CACertificates.Add(CACertificate{
			CACertificate: *c,
		})
		if err != nil {
			return fmt.Errorf("inserting ca_certificate into state: %w", err)
		}
	}

	for _, p := range raw.Partials {
		utils.ZeroOutTimestamps(p)
		err := kongState.Partials.Add(Partial{Partial: *p})
		if err != nil {
			return fmt.Errorf("inserting partial into state: %w", err)
		}
	}

	for _, p := range raw.Plugins {
		if p.Service != nil && !utils.Empty(p.Service.ID) {
			ok, s, err := ensureService(kongState, *p.Service.ID)
			if err != nil {
				return err
			}
			if ok {
				p.Service = s
			}
		}
		if p.Route != nil && !utils.Empty(p.Route.ID) {
			ok, r, err := ensureRoute(kongState, *p.Route.ID)
			if err != nil {
				return err
			}
			if ok {
				p.Route = r
			}
		}
		if p.Consumer != nil && !utils.Empty(p.Consumer.ID) {
			ok, c, err := ensureConsumer(kongState, *p.Consumer.ID)
			if err != nil {
				return err
			}
			if ok {
				p.Consumer = c
			}
		}
		if p.ConsumerGroup != nil && !utils.Empty(p.ConsumerGroup.ID) {
			ok, cg, err := ensureConsumerGroup(kongState, *p.ConsumerGroup.ID)
			if err != nil {
				return err
			}
			if ok {
				p.ConsumerGroup = cg
			}
		}
		if p.Partials != nil {
			var pluginPartials []*kong.PartialLink
			for _, partial := range p.Partials {
				if partial.Partial != nil && !utils.Empty(partial.Partial.ID) {
					ok, pt, err := ensurePartial(kongState, *partial.Partial.ID)
					if err != nil {
						return err
					}
					if ok {
						pluginPartials = append(pluginPartials, &kong.PartialLink{
							Partial: pt,
							Path:    partial.Path,
						})
					}
				}
			}
			p.Partials = pluginPartials
		}
		err := kongState.Plugins.Add(Plugin{Plugin: *p})
		if err != nil {
			return fmt.Errorf("inserting plugins into state: %w", err)
		}
	}

	for _, f := range raw.FilterChains {
		if f.Service != nil && !utils.Empty(f.Service.ID) {
			ok, s, err := ensureService(kongState, *f.Service.ID)
			if err != nil {
				return err
			}
			if ok {
				f.Service = s
			}
		}
		if f.Route != nil && !utils.Empty(f.Route.ID) {
			ok, r, err := ensureRoute(kongState, *f.Route.ID)
			if err != nil {
				return err
			}
			if ok {
				f.Route = r
			}
		}
		err := kongState.FilterChains.Add(FilterChain{FilterChain: *f})
		if err != nil {
			return fmt.Errorf("inserting filter chains into state: %w", err)
		}
	}

	for _, r := range raw.RBACRoles {
		err := kongState.RBACRoles.Add(RBACRole{RBACRole: *r})
		if err != nil {
			return fmt.Errorf("inserting rbac roles into state: %w", err)
		}
	}
	for _, r := range raw.RBACEndpointPermissions {
		err := kongState.RBACEndpointPermissions.Add(RBACEndpointPermission{RBACEndpointPermission: *r})
		if err != nil {
			return fmt.Errorf("inserting rbac endpoint permissions into state: %w", err)
		}
	}
	for _, v := range raw.Vaults {
		err := kongState.Vaults.Add(Vault{Vault: *v})
		if err != nil {
			return fmt.Errorf("inserting vault into state: %w", err)
		}
	}
	for _, l := range raw.Licenses {
		err := kongState.Licenses.Add(License{License: *l})
		if err != nil {
			return fmt.Errorf("inserting license into state: %w", err)
		}
	}

	for _, d := range raw.DegraphqlRoutes {
		if d.Service != nil && !utils.Empty(d.Service.ID) {
			ok, s, err := ensureService(kongState, *d.Service.ID)
			if err != nil {
				return err
			}
			if ok {
				d.Service = s
			}
		}
		err := kongState.DegraphqlRoutes.Add(DegraphqlRoute{DegraphqlRoute: *d})
		if err != nil {
			return fmt.Errorf("inserting degraphql route into state: %w", err)
		}
	}

	for _, c := range raw.CustomEntities {
		if c.Type() == "degraphql_routes" {
			entity := c.Object()

			degraphqlRoute, err := buildDegraphqlRouteFromCustomEntity(kongState, entity)
			if err != nil {
				return fmt.Errorf("building degraphql route from custom entity: %w", err)
			}

			err = kongState.DegraphqlRoutes.Add(degraphqlRoute)
			if err != nil {
				return fmt.Errorf("inserting degraphql route into state: %w", err)
			}
		}
	}

	for _, k := range raw.Keys {
		err := kongState.Keys.Add(Key{Key: *k})
		if err != nil {
			return fmt.Errorf("inserting key into state: %w", err)
		}
	}

	for _, s := range raw.KeySets {
		err := kongState.KeySets.Add(KeySet{KeySet: *s})
		if err != nil {
			return fmt.Errorf("inserting key-set into state: %w", err)
		}
	}

	return nil
}

func buildKonnect(kongState *KongState, raw *utils.KonnectRawState) error {
	for _, s := range raw.ServicePackages {
		servicePackage := s.DeepCopy()
		servicePackage.Versions = nil
		err := kongState.ServicePackages.Add(ServicePackage{
			ServicePackage: *servicePackage,
		})
		if err != nil {
			return fmt.Errorf("inserting service-package into state: %w", err)
		}

		for _, v := range s.Versions {
			v = *v.DeepCopy()
			v.ServicePackage = servicePackage.DeepCopy()
			err := kongState.ServiceVersions.Add(ServiceVersion{
				ServiceVersion: v,
			})
			if err != nil {
				return fmt.Errorf("inserting service-version into state: %w", err)
			}
		}
	}
	for _, d := range raw.Documents {
		document := d.ShallowCopy()
		err := kongState.Documents.Add(Document{
			Document: *document,
		})
		if err != nil {
			return fmt.Errorf("inserting document into state: %w", err)
		}
	}
	return nil
}

func GetKonnectState(rawKong *utils.KongRawState,
	rawKonnect *utils.KonnectRawState,
) (*KongState, error) {
	kongState, err := NewKongState()
	if err != nil {
		return nil, fmt.Errorf("creating new in-memory state of Kong: %w", err)
	}

	err = buildKong(kongState, rawKong)
	if err != nil {
		return nil, err
	}

	err = buildKonnect(kongState, rawKonnect)
	if err != nil {
		return nil, err
	}
	return kongState, nil
}

func buildDegraphqlRouteFromCustomEntity(kongState *KongState, entity map[string]interface{}) (DegraphqlRoute, error) {
	var degraphqlRoute DegraphqlRoute

	if entity["id"] != nil {
		id, ok := entity["id"].(string)
		if !ok {
			return DegraphqlRoute{}, fmt.Errorf("id must be of type string")
		}
		degraphqlRoute.ID = kong.String(id)
	}

	if entity["service"] != nil {
		service, ok := entity["service"].(map[string]interface{})
		if !ok {
			return DegraphqlRoute{}, fmt.Errorf("service must be of type object")
		}

		serviceID, ok := service["id"].(string)
		if !ok {
			return DegraphqlRoute{}, fmt.Errorf("service must be of type object with a valid string id or name")
		}

		ok, s, err := ensureService(kongState, serviceID)
		if err != nil {
			return DegraphqlRoute{}, err
		}
		if !ok {
			return DegraphqlRoute{}, fmt.Errorf("service must be of type object with a valid string id or name")
		}
		degraphqlRoute.Service = s
	}

	if entity["uri"] != nil {
		uri, ok := entity["uri"].(string)
		if !ok {
			return DegraphqlRoute{}, fmt.Errorf("uri must be of type string")
		}
		degraphqlRoute.URI = kong.String(uri)
	}

	if entity["query"] != nil {
		query, ok := entity["query"].(string)
		if !ok {
			return DegraphqlRoute{}, fmt.Errorf("query must be of type string")
		}
		degraphqlRoute.Query = kong.String(query)
	}

	if entity["methods"] != nil {
		methodSlice, ok := entity["methods"].([]interface{})
		if !ok {
			return DegraphqlRoute{}, fmt.Errorf("methods must be an array of strings")
		}

		methods := make([]string, len(methodSlice))
		for i, v := range methodSlice {
			method, ok := v.(string)
			if !ok {
				return DegraphqlRoute{}, fmt.Errorf("methods must be an array of strings")
			}
			methods[i] = method
		}
		degraphqlRoute.Methods = kong.StringSlice(methods...)
	}

	return degraphqlRoute, nil
}
