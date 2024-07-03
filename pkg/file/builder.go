package file

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/kong/go-database-reconciler/pkg/konnect"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/kong/go-kong/kong"
)

const ratelimitingAdvancedPluginName = "rate-limiting-advanced"

type stateBuilder struct {
	targetContent   *Content
	rawState        *utils.KongRawState
	konnectRawState *utils.KonnectRawState
	currentState    *state.KongState
	defaulter       *utils.Defaulter
	kongVersion     semver.Version

	selectTags          []string
	lookupTagsConsumers []string
	lookupTagsRoutes    []string
	skipCACerts         bool
	includeLicenses     bool
	intermediate        *state.KongState

	client *kong.Client
	ctx    context.Context

	schemasCache map[string]map[string]interface{}

	disableDynamicDefaults bool

	isKonnect bool

	checkRoutePaths bool

	isConsumerGroupScopedPluginSupported bool

	err error
}

// uuid generates a UUID string and returns a pointer to it.
// It is a variable for testing purpose, to override and supply
// a deterministic UUID generator.
var uuid = func() *string {
	return kong.String(utils.UUID())
}

var ErrWorkspaceNotFound = fmt.Errorf("workspace not found")

func (b *stateBuilder) build() (*utils.KongRawState, *utils.KonnectRawState, error) {
	// setup
	var err error
	b.rawState = &utils.KongRawState{}
	b.konnectRawState = &utils.KonnectRawState{}
	b.schemasCache = make(map[string]map[string]interface{})

	b.intermediate, err = state.NewKongState()
	if err != nil {
		return nil, nil, err
	}

	defaulter, err := defaulter(b.ctx, b.client, b.targetContent, b.disableDynamicDefaults, b.isKonnect)
	if err != nil {
		return nil, nil, err
	}
	b.defaulter = defaulter

	if utils.Kong300Version.LTE(b.kongVersion) {
		b.checkRoutePaths = true
	}

	if utils.Kong340Version.LTE(b.kongVersion) || b.isKonnect {
		b.isConsumerGroupScopedPluginSupported = true
	}

	// build
	b.certificates()
	if !b.skipCACerts {
		b.caCertificates()
	}
	b.services()
	b.routes()
	b.upstreams()
	b.consumerGroups()
	b.consumers()
	b.plugins()
	b.filterChains()
	b.enterprise()

	// konnect
	b.konnect()

	// result
	if b.err != nil {
		return nil, nil, b.err
	}
	return b.rawState, b.konnectRawState, nil
}

func (b *stateBuilder) ingestConsumerGroupScopedPlugins(cg FConsumerGroupObject) error {
	var plugins []FPlugin
	for _, plugin := range cg.Plugins {
		plugin.ConsumerGroup = utils.GetConsumerGroupReference(cg.ConsumerGroup)
		plugins = append(plugins, FPlugin{
			Plugin: kong.Plugin{
				ID:     plugin.ID,
				Name:   plugin.Name,
				Config: plugin.Config,
				ConsumerGroup: &kong.ConsumerGroup{
					ID: cg.ID,
				},
			},
			ConfigSource: plugin.ConfigSource,
		})
	}
	return b.ingestPlugins(plugins)
}

func (b *stateBuilder) addConsumerGroupPlugins(
	cg FConsumerGroupObject, cgo *kong.ConsumerGroupObject,
) error {
	for _, plugin := range cg.Plugins {
		if utils.Empty(plugin.ID) {
			current, err := b.currentState.ConsumerGroupPlugins.Get(
				*plugin.Name, *cg.ConsumerGroup.ID,
			)
			if errors.Is(err, state.ErrNotFound) {
				plugin.ID = uuid()
			} else if err != nil {
				return err
			} else {
				plugin.ID = kong.String(*current.ID)
			}
		}
		b.defaulter.MustSet(plugin)
		cgo.Plugins = append(cgo.Plugins, plugin)
	}
	return nil
}

func (b *stateBuilder) consumerGroups() {
	if b.err != nil {
		return
	}

	for _, cg := range b.targetContent.ConsumerGroups {
		cg := cg
		current, err := b.currentState.ConsumerGroups.Get(*cg.Name)
		if utils.Empty(cg.ID) {
			if errors.Is(err, state.ErrNotFound) {
				cg.ID = uuid()
			} else if err != nil {
				b.err = err
				return
			} else {
				cg.ID = kong.String(*current.ID)
			}
		}
		utils.MustMergeTags(&cg.ConsumerGroup, b.selectTags)

		cgo := kong.ConsumerGroupObject{
			ConsumerGroup: &cg.ConsumerGroup,
		}

		err = b.intermediate.ConsumerGroups.Add(state.ConsumerGroup{ConsumerGroup: cg.ConsumerGroup})
		if err != nil {
			b.err = err
			return
		}

		// Plugins and Consumer Groups can be handled in two ways:
		//   1. directly in the ConsumerGroup object
		//   2. by scoping the plugin to the ConsumerGroup (Kong >= 3.4.0)
		//
		// The first method is deprecated and will be removed in the future, but
		// we still need to support it for now. The isConsumerGroupScopedPluginSupported
		// flag is used to determine which method to use based on the Kong version.
		if b.isConsumerGroupScopedPluginSupported {
			if err := b.ingestConsumerGroupScopedPlugins(cg); err != nil {
				b.err = err
				return
			}
		} else {
			if err := b.addConsumerGroupPlugins(cg, &cgo); err != nil {
				b.err = err
				return
			}
		}
		if current != nil {
			cgo.ConsumerGroup.CreatedAt = current.CreatedAt
		}

		for _, consumer := range cg.Consumers {
			if consumer != nil {
				c, err := b.ingestConsumerGroupConsumer(cg.ID, &FConsumer{
					Consumer: *consumer,
				})
				if err != nil {
					b.err = err
					return
				}
				cgo.Consumers = append(cgo.Consumers, c)
			}
		}
		b.rawState.ConsumerGroups = append(b.rawState.ConsumerGroups, &cgo)
	}
}

func (b *stateBuilder) certificates() {
	if b.err != nil {
		return
	}

	for i := range b.targetContent.Certificates {
		c := b.targetContent.Certificates[i]
		if utils.Empty(c.ID) {
			cert, err := b.currentState.Certificates.GetByCertKey(*c.Cert,
				*c.Key)
			if errors.Is(err, state.ErrNotFound) {
				c.ID = uuid()
			} else if err != nil {
				b.err = err
				return
			} else {
				c.ID = kong.String(*cert.ID)
			}
		}
		utils.MustMergeTags(&c, b.selectTags)

		snisFromCert := c.SNIs

		kongCert := kong.Certificate{
			ID:        c.ID,
			Key:       c.Key,
			Cert:      c.Cert,
			Tags:      c.Tags,
			CreatedAt: c.CreatedAt,
		}
		b.rawState.Certificates = append(b.rawState.Certificates, &kongCert)

		// snis associated with the certificate
		var snis []kong.SNI
		for _, sni := range snisFromCert {
			sni.Certificate = &kong.Certificate{ID: kong.String(*c.ID)}
			snis = append(snis, sni)
		}
		if err := b.ingestSNIs(snis); err != nil {
			b.err = err
			return
		}
	}
}

func (b *stateBuilder) ingestSNIs(snis []kong.SNI) error {
	for _, sni := range snis {
		sni := sni
		currentSNI, err := b.currentState.SNIs.Get(*sni.Name)
		if utils.Empty(sni.ID) {
			if errors.Is(err, state.ErrNotFound) {
				sni.ID = uuid()
			} else if err != nil {
				return err
			} else {
				sni.ID = kong.String(*currentSNI.ID)
			}
		}
		utils.MustMergeTags(&sni, b.selectTags)
		if currentSNI != nil {
			sni.CreatedAt = currentSNI.CreatedAt
		}
		b.rawState.SNIs = append(b.rawState.SNIs, &sni)
	}
	return nil
}

func (b *stateBuilder) caCertificates() {
	if b.err != nil {
		return
	}

	for _, c := range b.targetContent.CACertificates {
		c := c
		cert, err := b.currentState.CACertificates.Get(*c.Cert)
		if utils.Empty(c.ID) {
			if errors.Is(err, state.ErrNotFound) {
				c.ID = uuid()
			} else if err != nil {
				b.err = err
				return
			} else {
				c.ID = kong.String(*cert.ID)
			}
		}
		utils.MustMergeTags(&c.CACertificate, b.selectTags)
		if cert != nil {
			c.CACertificate.CreatedAt = cert.CreatedAt
		}

		b.rawState.CACertificates = append(b.rawState.CACertificates,
			&c.CACertificate)
	}
}

func (b *stateBuilder) ingestConsumerGroupConsumer(cgID *string, c *FConsumer) (*kong.Consumer, error) {
	var (
		consumer *state.Consumer
		err      error
	)

	// if the consumer is already present in the target state because it is pulled from
	// upstream via the lookup tags, we don't want to create a new consumer.
	for _, tc := range b.targetContent.Consumers {
		stringTCTags := make([]string, len(tc.Tags))
		for i, tag := range tc.Tags {
			if tag != nil {
				stringTCTags[i] = *tag
			}
		}
		sort.Strings(stringTCTags)
		if reflect.DeepEqual(stringTCTags, b.lookupTagsConsumers) && !utils.Empty(tc.ID) {
			if (tc.Username != nil && c.Username != nil && *tc.Username == *c.Username) ||
				(tc.CustomID != nil && c.CustomID != nil && *tc.CustomID == *c.CustomID) {
				return &kong.Consumer{
					ID:       tc.ID,
					Username: tc.Username,
					CustomID: tc.CustomID,
					Tags:     tc.Tags,
				}, nil
			}
		}
	}

	if c.Username != nil {
		consumer, err = b.currentState.Consumers.GetByIDOrUsername(*c.Username)
	}
	if errors.Is(err, state.ErrNotFound) || consumer == nil {
		if c.CustomID != nil {
			consumer, err = b.currentState.Consumers.GetByCustomID(*c.CustomID)
		}
	}
	if utils.Empty(c.ID) {
		if errors.Is(err, state.ErrNotFound) {
			c.ID = uuid()
		} else if err != nil {
			return nil, err
		} else {
			c.ID = kong.String(*consumer.ID)
		}
	}
	utils.MustMergeTags(&c.Consumer, b.selectTags)
	if consumer != nil {
		c.Consumer.CreatedAt = consumer.CreatedAt
	}

	b.rawState.Consumers = append(b.rawState.Consumers, &c.Consumer)
	err = b.intermediate.Consumers.Add(state.Consumer{Consumer: c.Consumer})
	if err != nil {
		return nil, err
	}
	err = b.intermediate.ConsumerGroupConsumers.Add(state.ConsumerGroupConsumer{
		ConsumerGroupConsumer: kong.ConsumerGroupConsumer{
			ConsumerGroup: &kong.ConsumerGroup{ID: cgID},
			Consumer:      &c.Consumer,
		},
	})
	if err != nil {
		return nil, err
	}
	return &c.Consumer, nil
}

func (b *stateBuilder) consumers() {
	if b.err != nil {
		return
	}

	for _, c := range b.targetContent.Consumers {
		c := c

		var (
			consumer *state.Consumer
			err      error
		)
		if c.Username != nil {
			consumer, err = b.currentState.Consumers.GetByIDOrUsername(*c.Username)
		}
		if errors.Is(err, state.ErrNotFound) || consumer == nil {
			if c.CustomID != nil {
				consumer, err = b.currentState.Consumers.GetByCustomID(*c.CustomID)
			}
		}

		if utils.Empty(c.ID) {
			if errors.Is(err, state.ErrNotFound) {
				c.ID = uuid()
			} else if err != nil {
				b.err = err
				return
			} else {
				c.ID = kong.String(*consumer.ID)
			}
		}

		stringTags := make([]string, len(c.Tags))
		for i, tag := range c.Tags {
			if tag != nil {
				stringTags[i] = *tag
			}
		}
		sort.Strings(stringTags)
		sort.Strings(b.lookupTagsConsumers)
		// if the consumer tags and the lookup tags are the same, it means
		// that the consumer is a global consumer retrieved from upstream,
		// therefore we don't want to merge its tags with the selected tags.
		if !reflect.DeepEqual(stringTags, b.lookupTagsConsumers) {
			utils.MustMergeTags(&c.Consumer, b.selectTags)
		}

		if consumer != nil {
			c.Consumer.CreatedAt = consumer.CreatedAt
		}

		// check if consumer was already added in the consumer groups section.
		// if it was, we don't want to add it again.
		consumerAlreadyAdded := false
		consumerGroupConsumers, err := b.intermediate.ConsumerGroupConsumers.GetAll()
		if err != nil {
			b.err = err
			return
		}
		for _, cgc := range consumerGroupConsumers {
			if cgc.Consumer != nil &&
				(c.Username != nil && cgc.Consumer.Username != nil && *cgc.Consumer.Username == *c.Username ||
					c.CustomID != nil && cgc.Consumer.CustomID != nil && *cgc.Consumer.CustomID == *c.CustomID) {
				c.ID = cgc.Consumer.ID
				consumerAlreadyAdded = true
				break
			}
		}
		if !consumerAlreadyAdded {
			b.rawState.Consumers = append(b.rawState.Consumers, &c.Consumer)
			err = b.intermediate.Consumers.Add(state.Consumer{Consumer: c.Consumer})
			if err != nil {
				b.err = err
				return
			}
			// ingest consumer into consumer group
			if err := b.ingestIntoConsumerGroup(c); err != nil {
				b.err = err
				return
			}
		}

		// plugins for the Consumer
		var plugins []FPlugin
		for _, p := range c.Plugins {
			p.Consumer = utils.GetConsumerReference(c.Consumer)
			plugins = append(plugins, *p)
		}
		if err := b.ingestPlugins(plugins); err != nil {
			b.err = err
			return
		}

		var keyAuths []kong.KeyAuth
		for _, cred := range c.KeyAuths {
			cred.Consumer = utils.GetConsumerReference(c.Consumer)
			keyAuths = append(keyAuths, *cred)
		}
		if err := b.ingestKeyAuths(keyAuths); err != nil {
			b.err = err
			return
		}

		var basicAuths []kong.BasicAuth
		for _, cred := range c.BasicAuths {
			cred.Consumer = utils.GetConsumerReference(c.Consumer)
			basicAuths = append(basicAuths, *cred)
		}
		if err := b.ingestBasicAuths(basicAuths); err != nil {
			b.err = err
			return
		}

		var hmacAuths []kong.HMACAuth
		for _, cred := range c.HMACAuths {
			cred.Consumer = utils.GetConsumerReference(c.Consumer)
			hmacAuths = append(hmacAuths, *cred)
		}
		if err := b.ingestHMACAuths(hmacAuths); err != nil {
			b.err = err
			return
		}

		var jwtAuths []kong.JWTAuth
		for _, cred := range c.JWTAuths {
			cred.Consumer = utils.GetConsumerReference(c.Consumer)
			jwtAuths = append(jwtAuths, *cred)
		}
		if err := b.ingestJWTAuths(jwtAuths); err != nil {
			b.err = err
			return
		}

		var oauth2Creds []kong.Oauth2Credential
		for _, cred := range c.Oauth2Creds {
			cred.Consumer = utils.GetConsumerReference(c.Consumer)
			oauth2Creds = append(oauth2Creds, *cred)
		}
		if err := b.ingestOauth2Creds(oauth2Creds); err != nil {
			b.err = err
			return
		}

		var aclGroups []kong.ACLGroup
		for _, cred := range c.ACLGroups {
			cred.Consumer = utils.GetConsumerReference(c.Consumer)
			aclGroups = append(aclGroups, *cred)
		}
		if err := b.ingestACLGroups(aclGroups); err != nil {
			b.err = err
			return
		}

		var mtlsAuths []kong.MTLSAuth
		for _, cred := range c.MTLSAuths {
			cred.Consumer = utils.GetConsumerReference(c.Consumer)
			mtlsAuths = append(mtlsAuths, *cred)
		}

		b.ingestMTLSAuths(mtlsAuths)
	}
}

func (b *stateBuilder) ingestIntoConsumerGroup(consumer FConsumer) error {
	for _, group := range consumer.Groups {
		found := false
		for _, cg := range b.rawState.ConsumerGroups {
			if group.ID != nil && *cg.ConsumerGroup.ID == *group.ID {
				cg.Consumers = append(cg.Consumers, &consumer.Consumer)
				found = true
				break

			}
			if group.Name != nil && *cg.ConsumerGroup.Name == *group.Name {
				cg.Consumers = append(cg.Consumers, &consumer.Consumer)
				found = true
				break
			}
		}
		if !found {
			var groupIdentifier string
			if group.Name != nil {
				groupIdentifier = *group.Name
			} else {
				groupIdentifier = *group.ID
			}
			return fmt.Errorf(
				"consumer-group '%s' not found for consumer '%s'", groupIdentifier, *consumer.ID,
			)
		}
	}
	return nil
}

func (b *stateBuilder) ingestKeyAuths(creds []kong.KeyAuth) error {
	for _, cred := range creds {
		cred := cred
		existingCred, err := b.currentState.KeyAuths.Get(*cred.Key)
		if utils.Empty(cred.ID) {
			if errors.Is(err, state.ErrNotFound) {
				cred.ID = uuid()
			} else if err != nil {
				return err
			} else {
				cred.ID = kong.String(*existingCred.ID)
			}
		}
		if b.kongVersion.GTE(utils.Kong140Version) {
			utils.MustMergeTags(&cred, b.selectTags)
		}
		if existingCred != nil {
			cred.CreatedAt = existingCred.CreatedAt
		}
		b.rawState.KeyAuths = append(b.rawState.KeyAuths, &cred)
	}
	return nil
}

func (b *stateBuilder) ingestBasicAuths(creds []kong.BasicAuth) error {
	for _, cred := range creds {
		cred := cred
		existingCred, err := b.currentState.BasicAuths.Get(*cred.Username)
		if utils.Empty(cred.ID) {
			if errors.Is(err, state.ErrNotFound) {
				cred.ID = uuid()
			} else if err != nil {
				return err
			} else {
				cred.ID = kong.String(*existingCred.ID)
			}
		}
		if b.kongVersion.GTE(utils.Kong140Version) {
			utils.MustMergeTags(&cred, b.selectTags)
		}
		if existingCred != nil {
			cred.CreatedAt = existingCred.CreatedAt
		}
		b.rawState.BasicAuths = append(b.rawState.BasicAuths, &cred)
	}
	return nil
}

func (b *stateBuilder) ingestHMACAuths(creds []kong.HMACAuth) error {
	for _, cred := range creds {
		cred := cred
		existingCred, err := b.currentState.HMACAuths.Get(*cred.Username)
		if utils.Empty(cred.ID) {
			if errors.Is(err, state.ErrNotFound) {
				cred.ID = uuid()
			} else if err != nil {
				return err
			} else {
				cred.ID = kong.String(*existingCred.ID)
			}
		}
		if b.kongVersion.GTE(utils.Kong140Version) {
			utils.MustMergeTags(&cred, b.selectTags)
		}
		if existingCred != nil {
			cred.CreatedAt = existingCred.CreatedAt
		}
		b.rawState.HMACAuths = append(b.rawState.HMACAuths, &cred)
	}
	return nil
}

func (b *stateBuilder) ingestJWTAuths(creds []kong.JWTAuth) error {
	for _, cred := range creds {
		cred := cred
		existingCred, err := b.currentState.JWTAuths.Get(*cred.Key)
		if utils.Empty(cred.ID) {
			if errors.Is(err, state.ErrNotFound) {
				cred.ID = uuid()
			} else if err != nil {
				return err
			} else {
				cred.ID = kong.String(*existingCred.ID)
			}
		}
		if b.kongVersion.GTE(utils.Kong140Version) {
			utils.MustMergeTags(&cred, b.selectTags)
		}
		if existingCred != nil {
			cred.CreatedAt = existingCred.CreatedAt
		}
		b.rawState.JWTAuths = append(b.rawState.JWTAuths, &cred)
	}
	return nil
}

func (b *stateBuilder) ingestOauth2Creds(creds []kong.Oauth2Credential) error {
	for _, cred := range creds {
		cred := cred
		existingCred, err := b.currentState.Oauth2Creds.Get(*cred.ClientID)
		if utils.Empty(cred.ID) {
			if errors.Is(err, state.ErrNotFound) {
				cred.ID = uuid()
			} else if err != nil {
				return err
			} else {
				cred.ID = kong.String(*existingCred.ID)
			}
		}
		if b.kongVersion.GTE(utils.Kong140Version) {
			utils.MustMergeTags(&cred, b.selectTags)
		}
		if existingCred != nil {
			cred.CreatedAt = existingCred.CreatedAt
		}
		b.rawState.Oauth2Creds = append(b.rawState.Oauth2Creds, &cred)
	}
	return nil
}

func (b *stateBuilder) ingestACLGroups(creds []kong.ACLGroup) error {
	for _, cred := range creds {
		cred := cred
		if utils.Empty(cred.ID) {
			existingCred, err := b.currentState.ACLGroups.Get(
				*cred.Consumer.ID,
				*cred.Group)
			if errors.Is(err, state.ErrNotFound) {
				cred.ID = uuid()
			} else if err != nil {
				return err
			} else {
				cred.ID = kong.String(*existingCred.ID)
			}
		}
		if b.kongVersion.GTE(utils.Kong140Version) {
			utils.MustMergeTags(&cred, b.selectTags)
		}
		b.rawState.ACLGroups = append(b.rawState.ACLGroups, &cred)
	}
	return nil
}

func (b *stateBuilder) ingestMTLSAuths(creds []kong.MTLSAuth) {
	kong230Version := semver.MustParse("2.3.0")
	for _, cred := range creds {
		cred := cred
		// normally, we'd want to look up existing resources in this case
		// however, this is impossible here: mtls-auth simply has no unique fields other than ID,
		// so we don't--schema validation requires the ID
		// there's nothing more to do here

		if b.kongVersion.GTE(kong230Version) {
			utils.MustMergeTags(&cred, b.selectTags)
		}
		b.rawState.MTLSAuths = append(b.rawState.MTLSAuths, &cred)
	}
}

func (b *stateBuilder) konnect() {
	if b.err != nil {
		return
	}

	for i := range b.targetContent.ServicePackages {
		targetSP := b.targetContent.ServicePackages[i]
		if utils.Empty(targetSP.ID) {
			currentSP, err := b.currentState.ServicePackages.Get(*targetSP.Name)
			if errors.Is(err, state.ErrNotFound) {
				targetSP.ID = uuid()
			} else if err != nil {
				b.err = err
				return
			} else {
				targetSP.ID = kong.String(*currentSP.ID)
			}
		}

		targetKonnectSP := konnect.ServicePackage{
			ID:          targetSP.ID,
			Name:        targetSP.Name,
			Description: targetSP.Description,
		}

		if targetSP.Document != nil {
			targetKonnectDoc := konnect.Document{
				ID:        targetSP.Document.ID,
				Path:      targetSP.Document.Path,
				Published: targetSP.Document.Published,
				Content:   targetSP.Document.Content,
				Parent:    &targetKonnectSP,
			}
			if utils.Empty(targetKonnectDoc.ID) {
				currentDoc, err := b.currentState.Documents.GetByParent(&targetKonnectSP, *targetKonnectDoc.Path)
				if errors.Is(err, state.ErrNotFound) {
					targetKonnectDoc.ID = uuid()
				} else if err != nil {
					b.err = err
					return
				} else {
					targetKonnectDoc.ID = kong.String(*currentDoc.ID)
				}
			}
			b.konnectRawState.Documents = append(b.konnectRawState.Documents, &targetKonnectDoc)
		}

		// versions associated with the package
		for _, targetSV := range targetSP.Versions {
			targetKonnectSV := konnect.ServiceVersion{
				ID:      targetSV.ID,
				Version: targetSV.Version,
			}
			targetRelationID := ""
			if utils.Empty(targetKonnectSV.ID) {
				currentSV, err := b.currentState.ServiceVersions.Get(*targetKonnectSP.ID, *targetKonnectSV.Version)
				if errors.Is(err, state.ErrNotFound) {
					targetKonnectSV.ID = uuid()
				} else if err != nil {
					b.err = err
					return
				} else {
					targetKonnectSV.ID = kong.String(*currentSV.ID)
					if currentSV.ControlPlaneServiceRelation != nil {
						targetRelationID = *currentSV.ControlPlaneServiceRelation.ID
					}
				}
			}
			if targetSV.Implementation != nil &&
				targetSV.Implementation.Kong != nil {
				err := b.ingestService(targetSV.Implementation.Kong.Service)
				if err != nil {
					b.err = err
					return
				}
				targetKonnectSV.ControlPlaneServiceRelation = &konnect.ControlPlaneServiceRelation{
					ControlPlaneEntityID: targetSV.Implementation.Kong.Service.ID,
				}
				if targetRelationID != "" {
					targetKonnectSV.ControlPlaneServiceRelation.ID = &targetRelationID
				}
			}
			if targetSV.Document != nil {
				targetKonnectDoc := konnect.Document{
					ID:        targetSV.Document.ID,
					Path:      targetSV.Document.Path,
					Published: targetSV.Document.Published,
					Content:   targetSV.Document.Content,
					Parent:    &targetKonnectSV,
				}
				if utils.Empty(targetKonnectDoc.ID) {
					currentDoc, err := b.currentState.Documents.GetByParent(&targetKonnectSV, *targetKonnectDoc.Path)
					if errors.Is(err, state.ErrNotFound) {
						targetKonnectDoc.ID = uuid()
					} else if err != nil {
						b.err = err
						return
					} else {
						targetKonnectDoc.ID = kong.String(*currentDoc.ID)
					}
				}
				b.konnectRawState.Documents = append(b.konnectRawState.Documents, &targetKonnectDoc)
			}
			targetKonnectSP.Versions = append(targetKonnectSP.Versions, targetKonnectSV)
		}

		b.konnectRawState.ServicePackages = append(b.konnectRawState.ServicePackages,
			&targetKonnectSP)
	}
}

func (b *stateBuilder) services() {
	if b.err != nil {
		return
	}

	for _, s := range b.targetContent.Services {
		s := s
		err := b.ingestService(&s)
		if err != nil {
			b.err = err
			return
		}
	}
}

func (b *stateBuilder) ingestService(s *FService) error {
	var (
		svc *state.Service
		err error
	)
	if !utils.Empty(s.Name) {
		svc, err = b.currentState.Services.Get(*s.Name)
	}
	if utils.Empty(s.ID) {
		if errors.Is(err, state.ErrNotFound) {
			s.ID = uuid()
		} else if err != nil {
			return err
		} else {
			s.ID = kong.String(*svc.ID)
		}
	}
	utils.MustMergeTags(&s.Service, b.selectTags)
	b.defaulter.MustSet(&s.Service)
	if svc != nil {
		s.Service.CreatedAt = svc.CreatedAt
	}
	b.rawState.Services = append(b.rawState.Services, &s.Service)
	err = b.intermediate.Services.Add(state.Service{Service: s.Service})
	if err != nil {
		return err
	}

	// plugins for the service
	var plugins []FPlugin
	for _, p := range s.Plugins {
		p.Service = utils.GetServiceReference(s.Service)
		plugins = append(plugins, *p)
	}
	if err := b.ingestPlugins(plugins); err != nil {
		return err
	}

	// filter chains for the service
	var filterChains []FFilterChain
	for _, f := range s.FilterChains {
		f.Service = utils.GetServiceReference(s.Service)
		filterChains = append(filterChains, *f)
	}
	if err := b.ingestFilterChains(filterChains); err != nil {
		return err
	}

	// routes for the service
	for _, r := range s.Routes {
		r := r
		r.Service = utils.GetServiceReference(s.Service)
		if err := b.ingestRoute(*r); err != nil {
			return err
		}
	}
	return nil
}

func (b *stateBuilder) routes() {
	if b.err != nil {
		return
	}

	for _, r := range b.targetContent.Routes {
		r := r
		if err := b.ingestRoute(r); err != nil {
			b.err = err
			return
		}
	}

	// check routes' paths format
	if b.checkRoutePaths {
		unsupportedRoutes := []string{}
		allRoutes, err := b.intermediate.Routes.GetAll()
		if err != nil {
			b.err = err
			return
		}
		for _, r := range allRoutes {
			if utils.HasPathsWithRegex300AndAbove(r.Route) {
				unsupportedRoutes = append(unsupportedRoutes, *r.Route.ID)
			}
		}
		if len(unsupportedRoutes) > 0 {
			utils.PrintRouteRegexWarning(unsupportedRoutes)
		}
	}
}

func (b *stateBuilder) enterprise() {
	b.rbacRoles()
	b.vaults()
	// In Konnect, licenses are managed by Konnect cloud,
	// so licenses should not be included running against Konnect when building Kong state from files.
	if b.includeLicenses && !b.isKonnect {
		b.licenses()
	}
}

func (b *stateBuilder) vaults() {
	if b.err != nil {
		return
	}

	for _, v := range b.targetContent.Vaults {
		v := v
		vault, err := b.currentState.Vaults.Get(*v.Prefix)
		if utils.Empty(v.ID) {
			if errors.Is(err, state.ErrNotFound) {
				v.ID = uuid()
			} else if err != nil {
				b.err = err
				return
			} else {
				v.ID = kong.String(*vault.ID)
			}
		}
		utils.MustMergeTags(&v.Vault, b.selectTags)
		if vault != nil {
			v.Vault.CreatedAt = vault.CreatedAt
		}

		b.rawState.Vaults = append(b.rawState.Vaults, &v.Vault)
	}
}

func (b *stateBuilder) licenses() {
	if b.err != nil {
		return
	}

	for _, l := range b.targetContent.Licenses {
		l := l
		// Fill with a random ID if the ID is not given.
		// If ID is not given in the file to sync from, a NEW license will be created.
		if utils.Empty(l.ID) {
			l.ID = uuid()
		}

		b.rawState.Licenses = append(b.rawState.Licenses, &l.License)
	}
}

func (b *stateBuilder) rbacRoles() {
	if b.err != nil {
		return
	}

	for _, r := range b.targetContent.RBACRoles {
		r := r
		role, err := b.currentState.RBACRoles.Get(*r.Name)
		if utils.Empty(r.ID) {
			if errors.Is(err, state.ErrNotFound) {
				r.ID = uuid()
			} else if err != nil {
				b.err = err
				return
			} else {
				r.ID = kong.String(*role.ID)
			}
		}
		if role != nil {
			r.RBACRole.CreatedAt = role.CreatedAt
		}
		b.rawState.RBACRoles = append(b.rawState.RBACRoles, &r.RBACRole)
		// rbac endpoint permissions for the role
		for _, ep := range r.EndpointPermissions {
			ep := ep
			ep.Role = &kong.RBACRole{ID: kong.String(*r.ID)}
			b.rawState.RBACEndpointPermissions = append(b.rawState.RBACEndpointPermissions, &ep.RBACEndpointPermission)
		}
	}
}

var (
	IPv6HasPortPattern    = regexp.MustCompile(`\]\:\d+$`)
	IPv6HasBracketPattern = regexp.MustCompile(`\[\S+\]$`)
)

func isIPv6(hostname string) bool {
	parts := strings.Split(hostname, ":")
	return len(parts) > 2
}

// expandIPv6 decompress an ipv6 address into its 'long' format.
// for example:
//
// from ::1 to 0000:0000:0000:0000:0000:0000:0000:0001.
func expandIPv6(address string) string {
	addr, err := netip.ParseAddr(address)
	if err != nil {
		return ""
	}
	addr.StringExpanded()
	return addr.StringExpanded()
}

// normalizeIPv6 normalizes an ipv6 address to the format [address]:port.
// for example:
// from ::1 to [0000:0000:0000:0000:0000:0000:0000:0001]:8000.
func normalizeIPv6(target string) (string, error) {
	ip := target
	port := "8000"
	match := IPv6HasPortPattern.FindStringSubmatch(target)
	if len(match) > 0 {
		// has [address]:port pattern
		port = strings.ReplaceAll(match[0], "]:", "")
		ip = strings.ReplaceAll(target, match[0], "")
		ip = removeBrackets(ip)
	} else {
		match = IPv6HasBracketPattern.FindStringSubmatch(target)
		if len(match) > 0 {
			ip = removeBrackets(match[0])
		}
		if net.ParseIP(ip).To16() == nil {
			return "", fmt.Errorf("invalid ipv6 address %s", target)
		}
	}
	expandedIPv6 := expandIPv6(ip)
	if expandedIPv6 == "" {
		return "", fmt.Errorf("invalid ipv6 address %s", target)
	}
	return fmt.Sprintf("[%s]:%s", expandedIPv6, port), nil
}

func removeBrackets(ip string) string {
	ip = strings.ReplaceAll(ip, "[", "")
	return strings.ReplaceAll(ip, "]", "")
}

func (b *stateBuilder) upstreams() {
	if b.err != nil {
		return
	}

	for _, u := range b.targetContent.Upstreams {
		u := u
		ups, err := b.currentState.Upstreams.Get(*u.Name)
		if utils.Empty(u.ID) {
			if errors.Is(err, state.ErrNotFound) {
				u.ID = uuid()
			} else if err != nil {
				b.err = err
				return
			} else {
				u.ID = kong.String(*ups.ID)
			}
		}
		utils.MustMergeTags(&u.Upstream, b.selectTags)
		b.defaulter.MustSet(&u.Upstream)
		if ups != nil {
			u.Upstream.CreatedAt = ups.CreatedAt
		}

		b.rawState.Upstreams = append(b.rawState.Upstreams, &u.Upstream)

		// targets for the upstream
		var targets []kong.Target
		for _, t := range u.Targets {
			t.Upstream = &kong.Upstream{ID: kong.String(*u.ID)}
			targets = append(targets, t.Target)
		}
		if err := b.ingestTargets(targets); err != nil {
			b.err = err
			return
		}
	}
}

func (b *stateBuilder) ingestTargets(targets []kong.Target) error {
	for _, t := range targets {
		t := t

		if t.Target != nil && isIPv6(*t.Target) {
			normalizedTarget, err := normalizeIPv6(*t.Target)
			if err != nil {
				return err
			}
			t.Target = kong.String(normalizedTarget)
		}

		if utils.Empty(t.ID) {
			target, err := b.currentState.Targets.Get(*t.Upstream.ID, *t.Target)
			if errors.Is(err, state.ErrNotFound) {
				t.ID = uuid()
			} else if err != nil {
				return err
			} else {
				t.ID = kong.String(*target.ID)
			}
		}
		utils.MustMergeTags(&t, b.selectTags)
		b.defaulter.MustSet(&t)
		b.rawState.Targets = append(b.rawState.Targets, &t)
	}
	return nil
}

func (b *stateBuilder) plugins() {
	if b.err != nil {
		return
	}

	var plugins []FPlugin
	for _, p := range b.targetContent.Plugins {
		p := p
		if p.Consumer != nil && !utils.Empty(p.Consumer.ID) {
			c, err := b.intermediate.Consumers.GetByIDOrUsername(*p.Consumer.ID)
			if errors.Is(err, state.ErrNotFound) {
				b.err = fmt.Errorf("consumer %v for plugin %v: %w",
					p.Consumer.FriendlyName(), *p.Name, err)

				return
			} else if err != nil {
				b.err = err
				return
			}
			p.Consumer = utils.GetConsumerReference(c.Consumer)
		}
		if p.Service != nil && !utils.Empty(p.Service.ID) {
			s, err := b.intermediate.Services.Get(*p.Service.ID)
			if errors.Is(err, state.ErrNotFound) {
				b.err = fmt.Errorf("service %v for plugin %v: %w",
					p.Service.FriendlyName(), *p.Name, err)

				return
			} else if err != nil {
				b.err = err
				return
			}
			p.Service = utils.GetServiceReference(s.Service)
		}
		if p.Route != nil && !utils.Empty(p.Route.ID) {
			r, err := b.intermediate.Routes.Get(*p.Route.ID)
			if errors.Is(err, state.ErrNotFound) {
				b.err = fmt.Errorf("route %v for plugin %v: %w",
					p.Route.FriendlyName(), *p.Name, err)

				return
			} else if err != nil {
				b.err = err
				return
			}
			p.Route = utils.GetRouteReference(r.Route)
		}
		if p.ConsumerGroup != nil && !utils.Empty(p.ConsumerGroup.ID) {
			cg, err := b.intermediate.ConsumerGroups.Get(*p.ConsumerGroup.ID)
			if errors.Is(err, state.ErrNotFound) {
				b.err = fmt.Errorf("consumer-group %v for plugin %v: %w",
					p.ConsumerGroup.FriendlyName(), *p.Name, err)
				return
			} else if err != nil {
				b.err = err
				return
			}
			p.ConsumerGroup = utils.GetConsumerGroupReference(cg.ConsumerGroup)
		}

		if err := b.validatePlugin(p); err != nil {
			b.err = err
			return
		}
		plugins = append(plugins, p)
	}
	if err := b.ingestPlugins(plugins); err != nil {
		b.err = err
		return
	}
}

func (b *stateBuilder) filterChains() {
	if b.err != nil {
		return
	}

	var filterChains []FFilterChain
	for _, f := range b.targetContent.FilterChains {
		f := f
		if f.Service != nil && !utils.Empty(f.Service.ID) {
			s, err := b.intermediate.Services.Get(*f.Service.ID)
			if errors.Is(err, state.ErrNotFound) {
				b.err = fmt.Errorf("service %v for filterChain %v: %w",
					f.Service.FriendlyName(), *f.Name, err)

				return
			} else if err != nil {
				b.err = err
				return
			}
			f.Service = utils.GetServiceReference(s.Service)
		}
		if f.Route != nil && !utils.Empty(f.Route.ID) {
			r, err := b.intermediate.Routes.Get(*f.Route.ID)
			if errors.Is(err, state.ErrNotFound) {
				b.err = fmt.Errorf("route %v for filterChain %v: %w",
					f.Route.FriendlyName(), *f.Name, err)

				return
			} else if err != nil {
				b.err = err
				return
			}
			f.Route = utils.GetRouteReference(r.Route)
		}
		filterChains = append(filterChains, f)
	}
	if err := b.ingestFilterChains(filterChains); err != nil {
		b.err = err
		return
	}
}

func (b *stateBuilder) validatePlugin(p FPlugin) error {
	if b.isConsumerGroupScopedPluginSupported && *p.Name == ratelimitingAdvancedPluginName {
		// check if deprecated consumer-groups configuration is present in the config
		var consumerGroupsFound bool
		if groups, ok := p.Config["consumer_groups"]; ok {
			// if groups is an array of length > 0, then consumer_groups is set
			if groupsArray, ok := groups.([]interface{}); ok && len(groupsArray) > 0 {
				consumerGroupsFound = true
			}
		}
		var enforceConsumerGroupsFound bool
		if enforceConsumerGroups, ok := p.Config["enforce_consumer_groups"]; ok {
			if enforceConsumerGroupsBool, ok := enforceConsumerGroups.(bool); ok && enforceConsumerGroupsBool {
				enforceConsumerGroupsFound = true
			}
		}
		if consumerGroupsFound || enforceConsumerGroupsFound {
			return utils.ErrorConsumerGroupUpgrade
		}
	}
	return nil
}

// strip_path schema default value is 'true', but it cannot be set when
// protocols include 'grpc' and/or 'grpcs'. When users explicitly set
// strip_path to 'true' with grpc/s protocols, deck returns a schema violation error.
// When strip_path is not set and protocols include grpc/s, deck sets strip_path to 'false',
// despite its default value would be 'true' under normal circumstances.
func getStripPathBasedOnProtocols(route kong.Route) (*bool, error) {
	for _, p := range route.Protocols {
		if *p == "grpc" || *p == "grpcs" {
			if route.StripPath != nil && *route.StripPath {
				return nil, fmt.Errorf("schema violation (strip_path: cannot set " +
					"'strip_path' when 'protocols' is 'grpc' or 'grpcs')")
			}
			return kong.Bool(false), nil
		}
	}
	return route.StripPath, nil
}

func (b *stateBuilder) ingestRoute(r FRoute) error {
	var (
		route *state.Route
		err   error
	)
	if !utils.Empty(r.Name) {
		route, err = b.currentState.Routes.Get(*r.Name)
	}
	if utils.Empty(r.ID) {
		if errors.Is(err, state.ErrNotFound) {
			r.ID = uuid()
		} else if err != nil {
			return err
		} else {
			r.ID = kong.String(*route.ID)
		}
	}

	stringTags := make([]string, len(r.Tags))
	for i, tag := range r.Tags {
		if tag != nil {
			stringTags[i] = *tag
		}
	}
	sort.Strings(stringTags)
	sort.Strings(b.lookupTagsRoutes)
	// if the consumer tags and the lookup tags are the same, it means
	// that the route is a global route retrieved from upstream,
	// therefore we don't want to merge its tags with the selected tags.
	if !reflect.DeepEqual(stringTags, b.lookupTagsRoutes) {
		utils.MustMergeTags(&r.Route, b.selectTags)
	}

	utils.MustMergeTags(&r, b.selectTags)
	stripPath, err := getStripPathBasedOnProtocols(r.Route)
	if err != nil {
		return err
	}
	r.Route.StripPath = stripPath

	hasExpression := r.Route.Expression != nil
	hasRegexPriority := r.Route.RegexPriority != nil
	hasPathHandling := r.Route.PathHandling != nil

	b.defaulter.MustSet(&r.Route)
	if route != nil {
		r.Route.CreatedAt = route.CreatedAt
	}

	// Kong Gateway supports different schemas for different router versions.
	// On the other hand, Konnect can support only one schema including all
	// fields from 'traditional' and 'expressions' router schemas.
	// This may be problematic when it comes to defaults injection, because
	// the defaults for the 'traditiona' router schema can be wrongly injected
	// into the 'expressions' route configuration.
	//
	// Here we make sure that only the fields that are supported for a given
	// router version are set in the route configuration.
	if b.isKonnect && hasExpression && !(hasRegexPriority || hasPathHandling) {
		if r.Route.PathHandling != nil {
			r.Route.PathHandling = nil
		}
		if r.Route.RegexPriority != nil {
			r.Route.RegexPriority = nil
		}
		if r.Route.Priority == nil {
			r.Route.Priority = kong.Uint64(0)
		}
	}

	b.rawState.Routes = append(b.rawState.Routes, &r.Route)
	err = b.intermediate.Routes.Add(state.Route{Route: r.Route})
	if err != nil {
		return err
	}

	// filter chains for the route
	var filterChains []FFilterChain
	for _, f := range r.FilterChains {
		f.Route = utils.GetRouteReference(r.Route)
		filterChains = append(filterChains, *f)
	}
	if err := b.ingestFilterChains(filterChains); err != nil {
		return err
	}

	// plugins for the route
	var plugins []FPlugin
	for _, p := range r.Plugins {
		p.Route = utils.GetRouteReference(r.Route)
		plugins = append(plugins, *p)
	}
	if err := b.ingestPlugins(plugins); err != nil {
		return err
	}
	if r.Service != nil && utils.Empty(r.Service.ID) && !utils.Empty(r.Service.Name) {
		s, err := b.intermediate.Services.Get(*r.Service.Name)
		if err != nil {
			return fmt.Errorf("retrieve intermediate services (%s): %w", *r.Service.Name, err)
		}
		r.Service.ID = s.ID
		r.Service.Name = nil
	}
	return nil
}

func (b *stateBuilder) getPluginSchema(pluginName string) (map[string]interface{}, error) {
	var schema map[string]interface{}

	// lookup in cache
	if schema, ok := b.schemasCache[pluginName]; ok {
		return schema, nil
	}

	exists, err := utils.WorkspaceExists(b.ctx, b.client)
	if err != nil {
		return nil, fmt.Errorf("ensure workspace exists: %w", err)
	}
	if !exists {
		return schema, ErrWorkspaceNotFound
	}

	schema, err = b.client.Plugins.GetFullSchema(b.ctx, &pluginName)
	if err != nil {
		return schema, err
	}
	b.schemasCache[pluginName] = schema
	return schema, nil
}

func (b *stateBuilder) addPluginDefaults(plugin *FPlugin) error {
	if b.client == nil {
		return nil
	}
	schema, err := b.getPluginSchema(*plugin.Name)
	if err != nil {
		if errors.Is(err, ErrWorkspaceNotFound) {
			return nil
		}
		return fmt.Errorf("retrieve schema for %v from Kong: %w", *plugin.Name, err)
	}
	return kong.FillPluginsDefaults(&plugin.Plugin, schema)
}

func (b *stateBuilder) ingestPlugins(plugins []FPlugin) error {
	for _, p := range plugins {
		p := p
		cID, rID, sID, cgID := pluginRelations(&p.Plugin)
		plugin, err := b.currentState.Plugins.GetByProp(*p.Name,
			sID, rID, cID, cgID)
		if utils.Empty(p.ID) {
			if errors.Is(err, state.ErrNotFound) {
				p.ID = uuid()
			} else if err != nil {
				return err
			} else {
				p.ID = kong.String(*plugin.ID)
			}
		}
		if p.Config == nil {
			p.Config = make(map[string]interface{})
		}
		p.Config = ensureJSON(p.Config)
		err = b.fillPluginConfig(&p)
		if err != nil {
			return err
		}
		if err := b.addPluginDefaults(&p); err != nil {
			return fmt.Errorf("add defaults to plugin '%v': %w", *p.Name, err)
		}
		utils.MustMergeTags(&p, b.selectTags)
		if plugin != nil {
			p.Plugin.CreatedAt = plugin.CreatedAt
		}
		b.rawState.Plugins = append(b.rawState.Plugins, &p.Plugin)
	}
	return nil
}

func (b *stateBuilder) fillPluginConfig(plugin *FPlugin) error {
	if plugin == nil {
		return fmt.Errorf("plugin is nil")
	}
	if !utils.Empty(plugin.ConfigSource) {
		conf, ok := b.targetContent.PluginConfigs[*plugin.ConfigSource]
		if !ok {
			return fmt.Errorf("_plugin_config %q not found",
				*plugin.ConfigSource)
		}
		for k, v := range conf {
			if _, ok := plugin.Config[k]; !ok {
				plugin.Config[k] = v
			}
		}
	}
	return nil
}

func pluginRelations(plugin *kong.Plugin) (cID, rID, sID, cgID string) {
	if plugin.Consumer != nil && !utils.Empty(plugin.Consumer.ID) {
		cID = *plugin.Consumer.ID
	}
	if plugin.Route != nil && !utils.Empty(plugin.Route.ID) {
		rID = *plugin.Route.ID
	}
	if plugin.Service != nil && !utils.Empty(plugin.Service.ID) {
		sID = *plugin.Service.ID
	}
	if plugin.ConsumerGroup != nil && !utils.Empty(plugin.ConsumerGroup.ID) {
		cgID = *plugin.ConsumerGroup.ID
	}
	return
}

func (b *stateBuilder) ingestFilterChains(filterChains []FFilterChain) error {
	for _, f := range filterChains {
		f := f
		rID, sID := filterChainRelations(&f.FilterChain)
		filterChain, err := b.currentState.FilterChains.GetByProp(sID, rID)
		if utils.Empty(f.ID) {
			if errors.Is(err, state.ErrNotFound) {
				f.ID = uuid()
			} else if err != nil {
				return err
			} else {
				f.ID = kong.String(*filterChain.ID)
			}
		}
		if filterChain != nil {
			f.FilterChain.CreatedAt = filterChain.CreatedAt
		}
		utils.MustMergeTags(&f, b.selectTags)
		b.rawState.FilterChains = append(b.rawState.FilterChains, &f.FilterChain)
	}
	return nil
}

func filterChainRelations(filterChain *kong.FilterChain) (rID, sID string) {
	if filterChain.Route != nil && !utils.Empty(filterChain.Route.ID) {
		rID = *filterChain.Route.ID
	}
	if filterChain.Service != nil && !utils.Empty(filterChain.Service.ID) {
		sID = *filterChain.Service.ID
	}
	return
}

func defaulter(
	ctx context.Context, client *kong.Client, fileContent *Content, disableDynamicDefaults, isKonnect bool,
) (*utils.Defaulter, error) {
	var kongDefaults KongDefaults
	if fileContent.Info != nil {
		kongDefaults = fileContent.Info.Defaults
	}
	opts := utils.DefaulterOpts{
		Client:                 client,
		KongDefaults:           kongDefaults,
		DisableDynamicDefaults: disableDynamicDefaults,
		IsKonnect:              isKonnect,
	}
	defaulter, err := utils.GetDefaulter(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("creating defaulter: %w", err)
	}
	return defaulter, nil
}
