package file

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kong/go-database-reconciler/pkg/cprint"
	"github.com/kong/go-database-reconciler/pkg/state"
	"github.com/kong/go-database-reconciler/pkg/utils"
	"github.com/kong/go-kong/kong"
	"sigs.k8s.io/yaml"
)

// WriteConfig holds settings to use to write the state file.
type WriteConfig struct {
	Workspace                        string
	SelectTags                       []string
	Filename                         string
	FileFormat                       Format
	WithID                           bool
	ControlPlaneName                 string
	ControlPlaneID                   string
	KongVersion                      string
	IsConsumerGroupPolicyOverrideSet bool
	SanitizeContent                  bool
}

func compareOrder(obj1, obj2 sortable) bool {
	return strings.Compare(obj1.sortKey(), obj2.sortKey()) < 0
}

// compareCredential compares two credentials by primary key, falling back to ID.
// Both primary and id parameters are pointers that may be nil.
func compareCredential(primary1, primary2, id1, id2 *string) bool {
	p1, p2 := "", ""
	if primary1 != nil {
		p1 = *primary1
	}
	if primary2 != nil {
		p2 = *primary2
	}
	if p1 != p2 {
		return strings.Compare(p1, p2) < 0
	}
	// Fall back to ID comparison
	i1, i2 := "", ""
	if id1 != nil {
		i1 = *id1
	}
	if id2 != nil {
		i2 = *id2
	}
	return strings.Compare(i1, i2) < 0
}

// compareConsumerGroupPlugin compares two consumer group plugins by name, falling back to ID.
func compareConsumerGroupPlugin(p1, p2 *kong.ConsumerGroupPlugin) bool {
	n1, n2 := "", ""
	if p1.Name != nil {
		n1 = *p1.Name
	}
	if p2.Name != nil {
		n2 = *p2.Name
	}
	if n1 != n2 {
		return strings.Compare(n1, n2) < 0
	}
	// Fall back to ID comparison
	i1, i2 := "", ""
	if p1.ID != nil {
		i1 = *p1.ID
	}
	if p2.ID != nil {
		i2 = *p2.ID
	}
	return strings.Compare(i1, i2) < 0
}

func getFormatVersion(kongVersion string) (string, error) {
	parsedKongVersion, err := utils.ParseKongVersion(kongVersion)
	if err != nil {
		return "", fmt.Errorf("parsing Kong version: %w", err)
	}
	formatVersion := "1.1"
	if parsedKongVersion.GTE(utils.Kong300Version) {
		formatVersion = "3.0"
	}
	return formatVersion, nil
}

func exportIDsWithBasicAuth(kongState *state.KongState) bool {
	basicAuths, err := kongState.BasicAuths.GetAll()
	if err != nil {
		return false
	}

	exportIDs := len(basicAuths) > 0

	if exportIDs {
		const idsWarning = "Warning: basic-auth credentials detected, IDs will be exported"
		cprint.UpdatePrintlnStdErr(idsWarning)
	}

	return exportIDs
}

// KongStateToFile generates a state object to file.Content.
// It will omit timestamps and IDs while writing.
func KongStateToContent(kongState *state.KongState, config WriteConfig) (*Content, error) {
	file := &Content{}
	var err error

	file.Workspace = config.Workspace
	formatVersion, err := getFormatVersion(config.KongVersion)
	if err != nil {
		return nil, fmt.Errorf("get format version: %w", err)
	}
	file.FormatVersion = formatVersion
	if config.ControlPlaneName != "" || config.ControlPlaneID != "" {
		file.Konnect = &Konnect{
			ControlPlaneName: config.ControlPlaneName,
			ControlPlaneID:   config.ControlPlaneID,
		}
	}

	selectTags := config.SelectTags
	if len(selectTags) > 0 {
		file.Info = &Info{
			SelectorTags: selectTags,
		}
	}

	if config.IsConsumerGroupPolicyOverrideSet {
		if file.Info == nil {
			file.Info = &Info{
				ConsumerGroupPolicyOverrides: true,
			}
		} else {
			file.Info.ConsumerGroupPolicyOverrides = true
		}
	}

	if exportIDsWithBasicAuth(kongState) {
		config.WithID = true
	}

	err = populateServices(kongState, file, config)
	if err != nil {
		return nil, err
	}

	err = populateServicelessRoutes(kongState, file, config)
	if err != nil {
		return nil, err
	}

	err = populatePlugins(kongState, file, config)
	if err != nil {
		return nil, err
	}

	err = populateFilterChains(kongState, file, config)
	if err != nil {
		return nil, err
	}

	err = populateUpstreams(kongState, file, config)
	if err != nil {
		return nil, err
	}

	err = populateCertificates(kongState, file, config)
	if err != nil {
		return nil, err
	}

	err = populateCACertificates(kongState, file, config)
	if err != nil {
		return nil, err
	}

	err = populateConsumers(kongState, file, config)
	if err != nil {
		return nil, err
	}

	err = populateVaults(kongState, file, config)
	if err != nil {
		return nil, err
	}

	err = populateConsumerGroups(kongState, file, config)
	if err != nil {
		return nil, err
	}

	err = populateLicenses(kongState, file, config)
	if err != nil {
		return nil, err
	}

	err = populateDegraphqlRoutes(kongState, file)
	if err != nil {
		return nil, err
	}

	err = populatePartials(kongState, file)
	if err != nil {
		return nil, err
	}

	err = populateKeys(kongState, file, config)
	if err != nil {
		return nil, err
	}
	err = populateKeySets(kongState, file, config)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// KongStateToFile writes a state object to file with filename.
// See KongStateToContent for the State generation
func KongStateToFile(kongState *state.KongState, config WriteConfig) error {
	file, err := KongStateToContent(kongState, config)
	if err != nil {
		return err
	}
	return WriteContentToFile(file, config.Filename, config.FileFormat)
}

func KonnectStateToFile(kongState *state.KongState, config WriteConfig) error {
	file := &Content{}
	file.FormatVersion = "0.1"
	var err error

	err = populateServicePackages(kongState, file, config)
	if err != nil {
		return err
	}

	// do not populate service-less routes
	// we do not know if konnect supports these or not

	err = populatePlugins(kongState, file, config)
	if err != nil {
		return err
	}

	err = populateFilterChains(kongState, file, config)
	if err != nil {
		return err
	}

	err = populateUpstreams(kongState, file, config)
	if err != nil {
		return err
	}

	err = populateCertificates(kongState, file, config)
	if err != nil {
		return err
	}

	err = populateCACertificates(kongState, file, config)
	if err != nil {
		return err
	}

	err = populateConsumers(kongState, file, config)
	if err != nil {
		return err
	}

	return WriteContentToFile(file, config.Filename, config.FileFormat)
}

func populateServicePackages(kongState *state.KongState, file *Content,
	config WriteConfig,
) error {
	packages, err := kongState.ServicePackages.GetAll()
	if err != nil {
		return err
	}

	for _, sp := range packages {
		safePackageName := utils.NameToFilename(*sp.Name)
		p := FServicePackage{
			ID:          sp.ID,
			Name:        sp.Name,
			Description: sp.Description,
		}
		versions, err := kongState.ServiceVersions.GetAllByServicePackageID(*p.ID)
		if err != nil {
			return err
		}
		documents, err := kongState.Documents.GetAllByParent(sp)
		if err != nil {
			return err
		}

		for _, d := range documents {
			safeDocPath := utils.NameToFilename(*d.Path)
			fDocument := FDocument{
				ID:        d.ID,
				Path:      kong.String(filepath.Join(safePackageName, safeDocPath)),
				Published: d.Published,
				Content:   d.Content,
			}
			utils.ZeroOutID(&fDocument, fDocument.Path, config.WithID)
			p.Document = &fDocument
			// Although the documents API returns a list of documents and does support multiple documents,
			// we pretend there's only one because that's all the web UI allows.
			break
		}

		for _, v := range versions {
			safeVersionName := utils.NameToFilename(*v.Version)
			fVersion := FServiceVersion{
				ID:      v.ID,
				Version: v.Version,
			}
			if v.ControlPlaneServiceRelation != nil &&
				!utils.Empty(v.ControlPlaneServiceRelation.ControlPlaneEntityID) {
				kongServiceID := *v.ControlPlaneServiceRelation.ControlPlaneEntityID

				s, err := fetchService(kongServiceID, kongState, config)
				if err != nil {
					return err
				}
				fVersion.Implementation = &Implementation{
					Type: utils.ImplementationTypeKongGateway,
					Kong: &Kong{
						Service: s,
					},
				}
			}
			documents, err := kongState.Documents.GetAllByParent(v)
			if err != nil {
				return err
			}

			for _, d := range documents {
				safeDocPath := utils.NameToFilename(*d.Path)
				fDocument := FDocument{
					ID:        d.ID,
					Path:      kong.String(filepath.Join(safePackageName, safeVersionName, safeDocPath)),
					Published: d.Published,
					Content:   d.Content,
				}
				utils.ZeroOutID(&fDocument, fDocument.Path, config.WithID)
				fVersion.Document = &fDocument
				break
			}
			utils.ZeroOutID(&fVersion, fVersion.Version, config.WithID)
			p.Versions = append(p.Versions, fVersion)
		}
		sort.SliceStable(p.Versions, func(i, j int) bool {
			return compareOrder(p.Versions[i], p.Versions[j])
		})
		utils.ZeroOutID(&p, p.Name, config.WithID)
		file.ServicePackages = append(file.ServicePackages, p)
	}
	sort.SliceStable(file.ServicePackages, func(i, j int) bool {
		return compareOrder(file.ServicePackages[i], file.ServicePackages[j])
	})
	return nil
}

func populateServices(kongState *state.KongState, file *Content,
	config WriteConfig,
) error {
	services, err := kongState.Services.GetAll()
	if err != nil {
		return err
	}
	for _, s := range services {
		s, err := fetchService(*s.ID, kongState, config)
		if err != nil {
			return err
		}
		file.Services = append(file.Services, *s)
	}
	sort.SliceStable(file.Services, func(i, j int) bool {
		return compareOrder(file.Services[i], file.Services[j])
	})
	return nil
}

func getFRouteFromRoute(r *state.Route, kongState *state.KongState, config WriteConfig) (*FRoute, error) {
	plugins, err := kongState.Plugins.GetAllByRouteID(*r.ID)
	if err != nil {
		return nil, err
	}
	filterChains, err := kongState.FilterChains.GetAllByRouteID(*r.ID)
	if err != nil {
		return nil, err
	}
	utils.ZeroOutID(r, r.Name, config.WithID)
	utils.ZeroOutTimestamps(r)

	route := &FRoute{Route: r.Route}

	for _, p := range plugins {
		if p.Service != nil || p.Consumer != nil || p.ConsumerGroup != nil {
			continue
		}
		p.Route = nil
		utils.ZeroOutID(p, p.Name, config.WithID)
		utils.ZeroOutTimestamps(p)
		route.Plugins = append(route.Plugins, &FPlugin{Plugin: p.Plugin})
	}
	sort.SliceStable(route.Plugins, func(i, j int) bool {
		return compareOrder(route.Plugins[i], route.Plugins[j])
	})

	for _, f := range filterChains {
		f.Route = nil
		utils.ZeroOutID(f, f.Name, config.WithID)
		utils.ZeroOutTimestamps(f)
		route.FilterChains = append(route.FilterChains, &FFilterChain{FilterChain: f.FilterChain})
	}
	sort.SliceStable(route.FilterChains, func(i, j int) bool {
		return compareOrder(route.FilterChains[i], route.FilterChains[j])
	})

	return route, nil
}

func fetchService(id string, kongState *state.KongState, config WriteConfig) (*FService, error) {
	kongService, err := kongState.Services.Get(id)
	if err != nil {
		return nil, err
	}
	s := FService{Service: kongService.Service}
	routes, err := kongState.Routes.GetAllByServiceID(*s.ID)
	if err != nil {
		return nil, err
	}
	plugins, err := kongState.Plugins.GetAllByServiceID(*s.ID)
	if err != nil {
		return nil, err
	}
	filterChains, err := kongState.FilterChains.GetAllByServiceID(*s.ID)
	if err != nil {
		return nil, err
	}
	for _, p := range plugins {
		if p.Route != nil || p.Consumer != nil || p.ConsumerGroup != nil {
			continue
		}
		p.Service = nil
		utils.ZeroOutID(p, p.Name, config.WithID)
		utils.ZeroOutTimestamps(p)
		s.Plugins = append(s.Plugins, &FPlugin{Plugin: p.Plugin})
	}
	sort.SliceStable(s.Plugins, func(i, j int) bool {
		return compareOrder(s.Plugins[i], s.Plugins[j])
	})
	for _, r := range routes {
		r.Service = nil
		route, err := getFRouteFromRoute(r, kongState, config)
		if err != nil {
			return nil, err
		}
		s.Routes = append(s.Routes, route)
	}
	sort.SliceStable(s.Routes, func(i, j int) bool {
		return compareOrder(s.Routes[i], s.Routes[j])
	})
	for _, f := range filterChains {
		f.Service = nil
		utils.ZeroOutID(f, f.Name, config.WithID)
		utils.ZeroOutTimestamps(f)
		s.FilterChains = append(s.FilterChains, &FFilterChain{FilterChain: f.FilterChain})
	}
	sort.SliceStable(s.FilterChains, func(i, j int) bool {
		return compareOrder(s.FilterChains[i], s.FilterChains[j])
	})
	utils.ZeroOutID(&s, s.Name, config.WithID)
	utils.ZeroOutTimestamps(&s)
	return &s, nil
}

func populateServicelessRoutes(kongState *state.KongState, file *Content,
	config WriteConfig,
) error {
	routes, err := kongState.Routes.GetAll()
	if err != nil {
		return err
	}
	for _, r := range routes {
		if r.Service != nil {
			continue
		}
		route, err := getFRouteFromRoute(r, kongState, config)
		if err != nil {
			return err
		}
		file.Routes = append(file.Routes, *route)
	}
	sort.SliceStable(file.Routes, func(i, j int) bool {
		return compareOrder(file.Routes[i], file.Routes[j])
	})
	return nil
}

func populatePlugins(kongState *state.KongState, file *Content,
	config WriteConfig,
) error {
	plugins, err := kongState.Plugins.GetAll()
	if err != nil {
		return err
	}
	for _, p := range plugins {
		associations := 0
		if p.Consumer != nil {
			associations++
			cID := *p.Consumer.ID
			consumer, err := kongState.Consumers.GetByIDOrUsername(cID)
			if err != nil {
				return fmt.Errorf("unable to get consumer %s for plugin %s [%s]: %w", cID, *p.Name, *p.ID, err)
			}
			// If config.SanitizeContent is true, we wish to use IDs instead of usernames or names.
			// This is because Plugins use string references to Consumers, Services, etc. rather than objects.
			// Check file/types.go#foo (shadow type for Plugin) for how we embed string references.
			// Since the marshaling and unmarshaling of the Plugin uses "ID" field for embedding,
			// if we overwrite it with name/username, referential integrity will be lost.
			if !utils.Empty(consumer.Username) && !config.SanitizeContent {
				cID = *consumer.Username
			}
			p.Consumer.ID = &cID
		}
		if p.Service != nil {
			associations++
			sID := *p.Service.ID
			service, err := kongState.Services.Get(sID)
			if err != nil {
				return fmt.Errorf("unable to get service %s for plugin %s [%s]: %w", sID, *p.Name, *p.ID, err)
			}
			if !utils.Empty(service.Name) && !config.SanitizeContent {
				sID = *service.Name
			}
			p.Service.ID = &sID
		}
		if p.Route != nil {
			associations++
			rID := *p.Route.ID
			route, err := kongState.Routes.Get(rID)
			if err != nil {
				return fmt.Errorf("unable to get route %s for plugin %s [%s]: %w", rID, *p.Name, *p.ID, err)
			}
			if !utils.Empty(route.Name) && !config.SanitizeContent {
				rID = *route.Name
			}
			p.Route.ID = &rID
		}
		if p.ConsumerGroup != nil {
			associations++
			cgID := *p.ConsumerGroup.ID
			cg, err := kongState.ConsumerGroups.Get(cgID)
			if err != nil {
				return fmt.Errorf("unable to get consumer-group %s for plugin %s [%s]: %w", cgID, *p.Name, *p.ID, err)
			}
			if !utils.Empty(cg.Name) && !config.SanitizeContent {
				cgID = *cg.Name
			}
			p.ConsumerGroup.ID = &cgID
		}
		if associations == 0 || associations > 1 {
			utils.ZeroOutID(p, p.Name, config.WithID)
			utils.ZeroOutTimestamps(p)
			p := FPlugin{Plugin: p.Plugin}
			file.Plugins = append(file.Plugins, p)
		}
	}
	sort.SliceStable(file.Plugins, func(i, j int) bool {
		return compareOrder(file.Plugins[i], file.Plugins[j])
	})
	return nil
}

func populateFilterChains(kongState *state.KongState, file *Content,
	config WriteConfig,
) error {
	filterChains, err := kongState.FilterChains.GetAll()
	if err != nil {
		return err
	}

	for _, f := range filterChains {
		associations := 0
		if f.Service != nil {
			associations++
			sID := *f.Service.ID
			service, err := kongState.Services.Get(sID)
			if err != nil {
				return fmt.Errorf("unable to get service %s for filter chain %s [%s]: %w", sID, *f.Name, *f.ID, err)
			}
			// If config.SanitizeContent is true, we wish to use IDs instead of names.
			// This is because Plugins use string references to Services, Routes, etc. rather than objects.
			// Check file/types.go#SerializableFilterChain for how we embed string references.
			// Since the marshaling and unmarshaling of the Plugin uses "ID" field for embedding,
			// if we overwrite it with name/username, referential integrity will be lost.
			if !utils.Empty(service.Name) && !config.SanitizeContent {
				sID = *service.Name
			}
			f.Service.ID = &sID
		}
		if f.Route != nil {
			associations++
			rID := *f.Route.ID
			route, err := kongState.Routes.Get(rID)
			if err != nil {
				return fmt.Errorf("unable to get route %s for filter chain %s [%s]: %w", rID, *f.Name, *f.ID, err)
			}
			if !utils.Empty(route.Name) && !config.SanitizeContent {
				rID = *route.Name
			}
			f.Route.ID = &rID
		}
		if associations != 1 {
			return fmt.Errorf("unable to determine route or service entity associated with filter chain %s [%s]", *f.Name, *f.ID)
		}
	}
	sort.SliceStable(file.FilterChains, func(i, j int) bool {
		return compareOrder(file.FilterChains[i], file.FilterChains[j])
	})
	return nil
}

func populateUpstreams(kongState *state.KongState, file *Content,
	config WriteConfig,
) error {
	upstreams, err := kongState.Upstreams.GetAll()
	if err != nil {
		return err
	}
	for _, u := range upstreams {
		u := FUpstream{Upstream: u.Upstream}
		targets, err := kongState.Targets.GetAllByUpstreamID(*u.ID)
		if err != nil {
			return err
		}
		for _, t := range targets {
			t.Upstream = nil
			utils.ZeroOutID(t, t.Target.Target, config.WithID)
			utils.ZeroOutTimestamps(t)
			u.Targets = append(u.Targets, &FTarget{Target: t.Target})
		}
		sort.SliceStable(u.Targets, func(i, j int) bool {
			return compareOrder(u.Targets[i], u.Targets[j])
		})
		utils.ZeroOutID(&u, u.Name, config.WithID)
		utils.ZeroOutTimestamps(&u)
		file.Upstreams = append(file.Upstreams, u)
	}
	sort.SliceStable(file.Upstreams, func(i, j int) bool {
		return compareOrder(file.Upstreams[i], file.Upstreams[j])
	})
	return nil
}

func populateVaults(kongState *state.KongState, file *Content,
	config WriteConfig,
) error {
	vaults, err := kongState.Vaults.GetAll()
	if err != nil {
		return err
	}
	for _, v := range vaults {
		v := FVault{Vault: v.Vault}
		utils.ZeroOutID(&v, v.Prefix, config.WithID)
		utils.ZeroOutTimestamps(&v)
		file.Vaults = append(file.Vaults, v)
	}
	sort.SliceStable(file.Vaults, func(i, j int) bool {
		return compareOrder(file.Vaults[i], file.Vaults[j])
	})
	return nil
}

func populateCertificates(kongState *state.KongState, file *Content,
	config WriteConfig,
) error {
	certificates, err := kongState.Certificates.GetAll()
	if err != nil {
		return err
	}
	for _, c := range certificates {
		c := FCertificate{
			ID:   c.ID,
			Cert: c.Cert,
			Key:  c.Key,
			Tags: c.Tags,
		}
		snis, err := kongState.SNIs.GetAllByCertID(*c.ID)
		if err != nil {
			return err
		}
		for _, s := range snis {
			s.Certificate = nil
			utils.ZeroOutID(s, s.Name, config.WithID)
			utils.ZeroOutTimestamps(s)
			c.SNIs = append(c.SNIs, s.SNI)
		}
		sort.SliceStable(c.SNIs, func(i, j int) bool {
			return strings.Compare(*c.SNIs[i].Name, *c.SNIs[j].Name) < 0
		})
		utils.ZeroOutTimestamps(&c)
		file.Certificates = append(file.Certificates, c)
	}
	sort.SliceStable(file.Certificates, func(i, j int) bool {
		return compareOrder(file.Certificates[i], file.Certificates[j])
	})
	return nil
}

func populateCACertificates(kongState *state.KongState, file *Content,
	_ WriteConfig,
) error {
	caCertificates, err := kongState.CACertificates.GetAll()
	if err != nil {
		return err
	}
	for _, c := range caCertificates {
		c := FCACertificate{CACertificate: c.CACertificate}
		utils.ZeroOutTimestamps(&c)
		file.CACertificates = append(file.CACertificates, c)
	}
	sort.SliceStable(file.CACertificates, func(i, j int) bool {
		return compareOrder(file.CACertificates[i], file.CACertificates[j])
	})
	return nil
}

func populateConsumers(kongState *state.KongState, file *Content,
	config WriteConfig,
) error {
	consumers, err := kongState.Consumers.GetAll()
	if err != nil {
		return err
	}
	consumerGroups, err := kongState.ConsumerGroups.GetAll()
	if err != nil {
		return err
	}
	for _, c := range consumers {
		c := FConsumer{Consumer: c.Consumer}
		plugins, err := kongState.Plugins.GetAllByConsumerID(*c.ID)
		if err != nil {
			return err
		}
		for _, p := range plugins {
			if p.Service != nil || p.Route != nil || p.ConsumerGroup != nil {
				continue
			}
			utils.ZeroOutID(p, p.Name, config.WithID)
			utils.ZeroOutTimestamps(p)
			p.Consumer = nil
			c.Plugins = append(c.Plugins, &FPlugin{Plugin: p.Plugin})
		}
		sort.SliceStable(c.Plugins, func(i, j int) bool {
			return compareOrder(c.Plugins[i], c.Plugins[j])
		})
		// custom-entities associated with Consumer
		keyAuths, err := kongState.KeyAuths.GetAllByConsumerID(*c.ID)
		if err != nil {
			return err
		}
		for _, k := range keyAuths {
			utils.ZeroOutID(k, k.Key, config.WithID)
			utils.ZeroOutTimestamps(k)
			k.Consumer = nil
			c.KeyAuths = append(c.KeyAuths, &k.KeyAuth)
		}
		sort.SliceStable(c.KeyAuths, func(i, j int) bool {
			return compareCredential(c.KeyAuths[i].Key, c.KeyAuths[j].Key,
				c.KeyAuths[i].ID, c.KeyAuths[j].ID)
		})
		hmacAuth, err := kongState.HMACAuths.GetAllByConsumerID(*c.ID)
		if err != nil {
			return err
		}
		for _, k := range hmacAuth {
			k.Consumer = nil
			utils.ZeroOutID(k, k.Username, config.WithID)
			utils.ZeroOutTimestamps(k)
			c.HMACAuths = append(c.HMACAuths, &k.HMACAuth)
		}
		sort.SliceStable(c.HMACAuths, func(i, j int) bool {
			return compareCredential(c.HMACAuths[i].Username, c.HMACAuths[j].Username,
				c.HMACAuths[i].ID, c.HMACAuths[j].ID)
		})
		jwtSecrets, err := kongState.JWTAuths.GetAllByConsumerID(*c.ID)
		if err != nil {
			return err
		}
		for _, k := range jwtSecrets {
			k.Consumer = nil
			utils.ZeroOutID(k, k.Key, config.WithID)
			utils.ZeroOutTimestamps(k)
			c.JWTAuths = append(c.JWTAuths, &k.JWTAuth)
		}
		sort.SliceStable(c.JWTAuths, func(i, j int) bool {
			return compareCredential(c.JWTAuths[i].Key, c.JWTAuths[j].Key,
				c.JWTAuths[i].ID, c.JWTAuths[j].ID)
		})
		basicAuths, err := kongState.BasicAuths.GetAllByConsumerID(*c.ID)
		if err != nil {
			return err
		}
		for _, k := range basicAuths {
			k.Consumer = nil
			utils.ZeroOutID(k, k.Username, config.WithID)
			utils.ZeroOutTimestamps(k)
			c.BasicAuths = append(c.BasicAuths, &k.BasicAuth)
		}
		sort.SliceStable(c.BasicAuths, func(i, j int) bool {
			return compareCredential(c.BasicAuths[i].Username, c.BasicAuths[j].Username,
				c.BasicAuths[i].ID, c.BasicAuths[j].ID)
		})
		oauth2Creds, err := kongState.Oauth2Creds.GetAllByConsumerID(*c.ID)
		if err != nil {
			return err
		}
		for _, k := range oauth2Creds {
			k.Consumer = nil
			utils.ZeroOutID(k, k.ClientID, config.WithID)
			utils.ZeroOutTimestamps(k)
			c.Oauth2Creds = append(c.Oauth2Creds, &k.Oauth2Credential)
		}
		sort.SliceStable(c.Oauth2Creds, func(i, j int) bool {
			return compareCredential(c.Oauth2Creds[i].ClientID, c.Oauth2Creds[j].ClientID,
				c.Oauth2Creds[i].ID, c.Oauth2Creds[j].ID)
		})
		aclGroups, err := kongState.ACLGroups.GetAllByConsumerID(*c.ID)
		if err != nil {
			return err
		}
		for _, k := range aclGroups {
			k.Consumer = nil
			utils.ZeroOutID(k, k.Group, config.WithID)
			utils.ZeroOutTimestamps(k)
			c.ACLGroups = append(c.ACLGroups, &k.ACLGroup)
		}
		sort.SliceStable(c.ACLGroups, func(i, j int) bool {
			return compareCredential(c.ACLGroups[i].Group, c.ACLGroups[j].Group,
				c.ACLGroups[i].ID, c.ACLGroups[j].ID)
		})
		mtlsAuths, err := kongState.MTLSAuths.GetAllByConsumerID(*c.ID)
		if err != nil {
			return err
		}
		for _, k := range mtlsAuths {
			utils.ZeroOutTimestamps(k)
			k.Consumer = nil
			c.MTLSAuths = append(c.MTLSAuths, &k.MTLSAuth)
		}
		sort.SliceStable(c.MTLSAuths, func(i, j int) bool {
			return compareCredential(c.MTLSAuths[i].SubjectName, c.MTLSAuths[j].SubjectName,
				c.MTLSAuths[i].ID, c.MTLSAuths[j].ID)
		})
		// populate groups
		for _, cg := range consumerGroups {
			cg := *cg
			_, err := kongState.ConsumerGroupConsumers.Get(*c.ID, *cg.ID)
			if err != nil {
				if !errors.Is(err, state.ErrNotFound) {
					return err
				}
				continue
			}
			utils.ZeroOutID(&cg, cg.Name, config.WithID)
			utils.ZeroOutTimestamps(&cg)
			c.Groups = append(c.Groups, cg.DeepCopy())
		}
		sort.SliceStable(c.Groups, func(i, j int) bool {
			return compareCredential(c.Groups[i].Name, c.Groups[j].Name,
				c.Groups[i].ID, c.Groups[j].ID)
		})
		sort.SliceStable(c.Plugins, func(i, j int) bool {
			return compareOrder(c.Plugins[i], c.Plugins[j])
		})
		utils.ZeroOutID(&c, c.Username, config.WithID)
		utils.ZeroOutTimestamps(&c)
		file.Consumers = append(file.Consumers, c)
	}
	rbacRoles, err := kongState.RBACRoles.GetAll()
	if err != nil {
		return err
	}
	for _, r := range rbacRoles {
		r := FRBACRole{RBACRole: r.RBACRole}
		eps, err := kongState.RBACEndpointPermissions.GetAllByRoleID(*r.ID)
		if err != nil {
			return err
		}
		for _, ep := range eps {
			ep.Role = nil
			utils.ZeroOutTimestamps(ep)
			r.EndpointPermissions = append(
				r.EndpointPermissions, &FRBACEndpointPermission{RBACEndpointPermission: ep.RBACEndpointPermission})
		}
		sort.SliceStable(r.EndpointPermissions, func(i, j int) bool {
			e1, e2 := "", ""
			if r.EndpointPermissions[i].Endpoint != nil {
				e1 = *r.EndpointPermissions[i].Endpoint
			}
			if r.EndpointPermissions[j].Endpoint != nil {
				e2 = *r.EndpointPermissions[j].Endpoint
			}
			return strings.Compare(e1, e2) < 0
		})
		utils.ZeroOutID(&r, r.Name, config.WithID)
		utils.ZeroOutTimestamps(&r)
		file.RBACRoles = append(file.RBACRoles, r)
	}
	sort.SliceStable(file.Consumers, func(i, j int) bool {
		return compareOrder(file.Consumers[i], file.Consumers[j])
	})
	return nil
}

func populateConsumerGroups(kongState *state.KongState, file *Content,
	config WriteConfig,
) error {
	consumerGroups, err := kongState.ConsumerGroups.GetAll()
	if err != nil {
		return err
	}
	cgPlugins, err := kongState.ConsumerGroupPlugins.GetAll()
	if err != nil {
		return err
	}
	for _, cg := range consumerGroups {
		group := FConsumerGroupObject{ConsumerGroup: cg.ConsumerGroup}
		for _, plugin := range cgPlugins {
			if plugin.ID != nil && cg.ID != nil {
				plugin := plugin
				if plugin.ConsumerGroup != nil && *plugin.ConsumerGroup.ID == *cg.ID {
					utils.ZeroOutID(plugin, plugin.Name, config.WithID)
					utils.ZeroOutID(plugin.ConsumerGroup, plugin.ConsumerGroup.Name, config.WithID)
					utils.ZeroOutTimestamps(plugin.ConsumerGroupPlugin.ConsumerGroup)
					utils.ZeroOutField(&plugin.ConsumerGroupPlugin, "ConsumerGroup")
					group.Plugins = append(group.Plugins, &plugin.ConsumerGroupPlugin)
				}
			}
		}

		plugins, err := kongState.Plugins.GetAllByConsumerGroupID(*cg.ID)
		if err != nil {
			return err
		}
		for _, plugin := range plugins {
			if plugin.Service != nil || plugin.Consumer != nil || plugin.Route != nil {
				continue
			}
			if plugin.ID != nil && cg.ID != nil {
				if plugin.ConsumerGroup != nil && *plugin.ConsumerGroup.ID == *cg.ID {
					utils.ZeroOutID(plugin, plugin.Name, config.WithID)
					utils.ZeroOutID(plugin.ConsumerGroup, plugin.ConsumerGroup.Name, config.WithID)
					group.Plugins = append(group.Plugins, &kong.ConsumerGroupPlugin{
						ID:           plugin.ID,
						Name:         plugin.Name,
						InstanceName: plugin.InstanceName,
						Config:       plugin.Config,
						Tags:         plugin.Tags,
						Partials:     plugin.Partials,
					})
				}
			}
		}

		sort.SliceStable(group.Plugins, func(i, j int) bool {
			return compareConsumerGroupPlugin(group.Plugins[i], group.Plugins[j])
		})

		utils.ZeroOutID(&group, group.Name, config.WithID)
		utils.ZeroOutTimestamps(&group)
		file.ConsumerGroups = append(file.ConsumerGroups, group)
	}
	sort.SliceStable(file.ConsumerGroups, func(i, j int) bool {
		return compareOrder(file.ConsumerGroups[i], file.ConsumerGroups[j])
	})
	return nil
}

func populateLicenses(kongState *state.KongState, file *Content,
	config WriteConfig,
) error {
	licenses, err := kongState.Licenses.GetAll()
	if err != nil {
		return err
	}
	for _, l := range licenses {
		l := FLicense{License: l.License}
		utils.ZeroOutID(&l, l.Payload, config.WithID)
		utils.ZeroOutTimestamps(&l)
		file.Licenses = append(file.Licenses, l)
	}
	sort.SliceStable(file.Licenses, func(i, j int) bool {
		return compareOrder(file.Licenses[i], file.Licenses[j])
	})
	return nil
}

func populateDegraphqlRoutes(kongState *state.KongState, file *Content) error {
	degraphqlRoutes, err := kongState.DegraphqlRoutes.GetAll()
	if err != nil {
		return err
	}

	for _, d := range degraphqlRoutes {
		f := FCustomEntity{}

		err := copyFromDegraphqlRoute(DegraphqlRoute{
			DegraphqlRoute: d.DegraphqlRoute,
		}, &f)
		if err != nil {
			return err
		}
		utils.ZeroOutTimestamps(&f)

		file.CustomEntities = append(file.CustomEntities, f)
	}
	sort.SliceStable(file.CustomEntities, func(i, j int) bool {
		return compareOrder(file.CustomEntities[i], file.CustomEntities[j])
	})

	return nil
}

func populatePartials(kongState *state.KongState, file *Content) error {
	partials, err := kongState.Partials.GetAll()
	if err != nil {
		return err
	}
	for _, p := range partials {
		p := FPartial{Partial: p.Partial}
		utils.ZeroOutTimestamps(&p)
		file.Partials = append(file.Partials, p)
	}
	sort.SliceStable(file.Partials, func(i, j int) bool {
		return compareOrder(file.Partials[i], file.Partials[j])
	})
	return nil
}

func populateKeys(kongState *state.KongState, file *Content,
	config WriteConfig,
) error {
	keys, err := kongState.Keys.GetAll()
	if err != nil {
		return err
	}
	for _, k := range keys {
		k := FKey{Key: k.Key}
		if k.Set != nil {
			ks, err := kongState.KeySets.Get(*k.Set.ID)
			if err != nil {
				if !errors.Is(err, state.ErrNotFound) {
					return err
				}
			}
			utils.ZeroOutID(ks, ks.Name, config.WithID)
			utils.ZeroOutTimestamps(ks)
			k.Set = &ks.KeySet
		}
		utils.ZeroOutID(&k, k.Name, config.WithID)
		utils.ZeroOutTimestamps(&k)
		file.Keys = append(file.Keys, k)
	}
	sort.SliceStable(file.Keys, func(i, j int) bool {
		return compareOrder(file.Keys[i], file.Keys[j])
	})
	return nil
}

func populateKeySets(kongState *state.KongState, file *Content,
	config WriteConfig,
) error {
	sets, err := kongState.KeySets.GetAll()
	if err != nil {
		return err
	}
	for _, s := range sets {
		s := FKeySet{KeySet: s.KeySet}
		utils.ZeroOutID(&s, s.Name, config.WithID)
		utils.ZeroOutTimestamps(&s)
		file.KeySets = append(file.KeySets, s)
	}
	sort.SliceStable(file.KeySets, func(i, j int) bool {
		return compareOrder(file.KeySets[i], file.KeySets[j])
	})
	return nil
}

func WriteContentToFile(content *Content, filename string, format Format) error {
	var c []byte
	var err error
	switch format {
	case YAML:
		c, err = yaml.Marshal(content)
		if err != nil {
			return err
		}
	case JSON:
		c, err = json.MarshalIndent(content, "", "  ")
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown file format: %s", format)
	}

	if filename == "-" {
		if _, err := fmt.Print(string(c)); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}
	} else {
		filename = utils.AddExtToFilename(filename, strings.ToLower(string(format)))
		prefix, _ := filepath.Split(filename)
		if err := ioutil.WriteFile(filename, c, 0o600); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}
		for _, sp := range content.ServicePackages {
			if sp.Document != nil {
				if err := os.MkdirAll(filepath.Join(prefix, filepath.Dir(*sp.Document.Path)), 0o700); err != nil {
					return fmt.Errorf("creating document directory: %w", err)
				}
				if err := os.WriteFile(filepath.Join(prefix, *sp.Document.Path),
					[]byte(*sp.Document.Content), 0o600); err != nil {
					return fmt.Errorf("writing document file: %w", err)
				}
			}
			for _, v := range sp.Versions {
				if v.Document != nil {
					if err := os.MkdirAll(filepath.Join(prefix, filepath.Dir(*v.Document.Path)), 0o700); err != nil {
						return fmt.Errorf("creating document directory: %w", err)
					}
					if err := os.WriteFile(filepath.Join(prefix, *v.Document.Path),
						[]byte(*v.Document.Content), 0o600); err != nil {
						return fmt.Errorf("writing document file: %w", err)
					}
				}
			}
		}
	}
	return nil
}
