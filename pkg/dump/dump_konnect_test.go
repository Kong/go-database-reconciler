package dump

import (
	"reflect"
	"testing"

	"github.com/kong/go-database-reconciler/pkg/konnect"
	"github.com/kong/go-kong/kong"
)

func Test_kongServiceIDs(t *testing.T) {
	type args struct {
		cpID      string
		relations []*konnect.ControlPlaneServiceRelation
	}
	tests := []struct {
		name string
		args args
		want map[string]bool
	}{
		{
			name: "returns services belonging to the same control plane",
			args: args{
				cpID: "cp1",
				relations: []*konnect.ControlPlaneServiceRelation{
					{
						ID:                   new("id1"),
						ControlPlaneEntityID: new("kong-svc-1"),
						ControlPlane: &konnect.ControlPlane{
							ID: new("cp1"),
						},
					},
					{
						ID:                   new("id2"),
						ControlPlaneEntityID: new("kong-svc-2"),
						ControlPlane: &konnect.ControlPlane{
							ID: new("cp1"),
						},
					},
				},
			},
			want: map[string]bool{
				"kong-svc-1": true,
				"kong-svc-2": true,
			},
		},
		{
			name: "doesn't panic if relation.ControlPlaneEntityID is nil",
			args: args{
				cpID: "cp1",
				relations: []*konnect.ControlPlaneServiceRelation{
					{
						ID: new("id1"),
						ControlPlane: &konnect.ControlPlane{
							ID: new("cp2"),
						},
					},
				},
			},
			want: map[string]bool{},
		},
		{
			name: "doesn't include a service belonging to a different control plane",
			args: args{
				cpID: "cp1",
				relations: []*konnect.ControlPlaneServiceRelation{
					{
						ID:                   new("id1"),
						ControlPlaneEntityID: new("kong-svc-1"),
						ControlPlane: &konnect.ControlPlane{
							ID: new("cp2"),
						},
					},
				},
			},
			want: map[string]bool{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := kongServiceIDs(tt.args.cpID, tt.args.relations)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("kongServiceIDs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_filterNonKongPackages(t *testing.T) {
	type args struct {
		controlPlaneID string
		packages       []*konnect.ServicePackage
		relations      []*konnect.ControlPlaneServiceRelation
	}
	tests := []struct {
		name string
		args args
		want []*konnect.ServicePackage
	}{
		{
			name: "empty packages and relations returns nil",
			args: args{
				controlPlaneID: "cp1",
				packages:       []*konnect.ServicePackage{},
				relations:      []*konnect.ControlPlaneServiceRelation{},
			},
			want: nil,
		},
		{
			name: "package with no versions is returned in output",
			args: args{
				controlPlaneID: "cp1",
				packages: []*konnect.ServicePackage{
					{
						ID:   new("sp-id1"),
						Name: new("sp-name1"),
					},
				},
			},
			want: []*konnect.ServicePackage{
				{
					ID:   new("sp-id1"),
					Name: new("sp-name1"),
				},
			},
		},
		{
			name: "package with version that belong to a different control-plane is not included in output",
			args: args{
				controlPlaneID: "cp1",
				packages: []*konnect.ServicePackage{
					{
						ID:   new("sp-id1"),
						Name: new("sp-name1"),
						Versions: []konnect.ServiceVersion{
							{
								ID:      new("sv-id1"),
								Version: new("sv-v1"),
								ControlPlaneServiceRelation: &konnect.ControlPlaneServiceRelation{
									ControlPlaneEntityID: new("kong-svc-1"),
								},
							},
						},
					},
				},
				relations: []*konnect.ControlPlaneServiceRelation{
					{
						ID:                   new("id1"),
						ControlPlaneEntityID: new("kong-svc-1"),
						ControlPlane: &konnect.ControlPlane{
							ID: new("cp2"),
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "package with version that belong to same control-plane is included in output",
			args: args{
				controlPlaneID: "cp1",
				packages: []*konnect.ServicePackage{
					{
						ID:   new("sp-id1"),
						Name: new("sp-name1"),
						Versions: []konnect.ServiceVersion{
							{
								ID:      new("sv-id1"),
								Version: new("sv-v1"),
								ControlPlaneServiceRelation: &konnect.ControlPlaneServiceRelation{
									ControlPlaneEntityID: new("kong-svc-1"),
								},
							},
						},
					},
				},
				relations: []*konnect.ControlPlaneServiceRelation{
					{
						ID:                   new("id1"),
						ControlPlaneEntityID: new("kong-svc-1"),
						ControlPlane: &konnect.ControlPlane{
							ID: new("cp1"),
						},
					},
				},
			},
			want: []*konnect.ServicePackage{
				{
					ID:   new("sp-id1"),
					Name: new("sp-name1"),
					Versions: []konnect.ServiceVersion{
						{
							ID:      new("sv-id1"),
							Version: new("sv-v1"),
							ControlPlaneServiceRelation: &konnect.ControlPlaneServiceRelation{
								ControlPlaneEntityID: new("kong-svc-1"),
							},
						},
					},
				},
			},
		},
		{
			name: "package with versions without any implementations is not included",
			args: args{
				controlPlaneID: "cp1",
				packages: []*konnect.ServicePackage{
					{
						ID:   new("sp-id1"),
						Name: new("sp-name1"),
						Versions: []konnect.ServiceVersion{
							{
								ID:      new("sv-id1"),
								Version: new("sv-v1"),
							},
							{
								ID:      new("sv-id2"),
								Version: new("sv-v2"),
							},
						},
					},
				},
				relations: []*konnect.ControlPlaneServiceRelation{},
			},
			want: []*konnect.ServicePackage{
				{
					ID:   new("sp-id1"),
					Name: new("sp-name1"),
					Versions: []konnect.ServiceVersion{
						{
							ID:      new("sv-id1"),
							Version: new("sv-v1"),
						},
						{
							ID:      new("sv-id2"),
							Version: new("sv-v2"),
						},
					},
				},
			},
		},
		{
			name: "package with version's implementation absent from relations is not included",
			args: args{
				controlPlaneID: "cp1",
				packages: []*konnect.ServicePackage{
					{
						ID:   new("sp-id1"),
						Name: new("sp-name1"),
						Versions: []konnect.ServiceVersion{
							{
								ID:      new("sv-id1"),
								Version: new("sv-v1"),
								ControlPlaneServiceRelation: &konnect.ControlPlaneServiceRelation{
									ControlPlaneEntityID: new("kong-svc-1"),
								},
							},
						},
					},
				},
				relations: []*konnect.ControlPlaneServiceRelation{
					{
						ID:                   new("id1"),
						ControlPlaneEntityID: new("kong-svc-42"),
						ControlPlane: &konnect.ControlPlane{
							ID: new("cp1"),
						},
					},
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterNonKongPackages(tt.args.controlPlaneID, tt.args.packages, tt.args.relations)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterNonKongPackages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_excludeKonnectManagedPlugins(t *testing.T) {
	tests := []struct {
		name    string
		plugins []*kong.Plugin
		want    []*kong.Plugin
	}{
		{
			name: "exclude konnect tags",
			plugins: []*kong.Plugin{
				{
					Name: new("rate-limiting"),
					Tags: []*string{new("tag1")},
				},
				{
					Name: new("basic-auth"),
					Tags: []*string{},
				},
				{
					Name: new("key-auth"),
					Tags: []*string{
						new("konnect-app-registration"),
						new("konnect-managed-plugin"),
					},
				},
				{
					Name: new("acl"),
					Tags: []*string{
						new("konnect-app-registration"),
						new("konnect-managed-plugin"),
					},
				},
				{
					Name: new("prometheus"),
					Tags: []*string{
						new("konnect-managed-plugin"),
					},
				},
			},
			want: []*kong.Plugin{
				{
					Name: new("rate-limiting"),
					Tags: []*string{new("tag1")},
				},
				{
					Name: new("basic-auth"),
					Tags: []*string{},
				},
			},
		},
		{
			name:    "empty input",
			plugins: []*kong.Plugin{},
			want:    []*kong.Plugin{},
		},
		{
			name: "all konnect managed",
			plugins: []*kong.Plugin{
				{
					Name: new("key-auth"),
					Tags: []*string{
						new("konnect-app-registration"),
						new("konnect-managed-plugin"),
					},
				},
				{
					Name: new("acl"),
					Tags: []*string{
						new("konnect-app-registration"),
						new("konnect-managed-plugin"),
					},
				},
				{
					Name: new("prometheus"),
					Tags: []*string{
						new("konnect-managed-plugin"),
					},
				},
			},
			want: []*kong.Plugin{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := excludeKonnectManagedPlugins(tt.plugins)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("excludeKonnectManagedPlugins() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_excludeKonnectManagedEntities(t *testing.T) {
	tests := []struct {
		name     string
		entities []any
		want     []any
	}{
		{
			name: "exclude konnect managed",
			entities: []any{
				&kong.SNI{
					Name: new("foo"),
					Tags: []*string{
						new("konnect-managed"),
					},
				},
				&kong.SNI{
					Name: new("bar"),
					Tags: []*string{
						new("bar-tag1"),
					},
				},
				&kong.SNI{
					Name: new("baz"),
					Tags: []*string{
						new("konnect-managed"),
					},
				},
			},
			want: []any{
				&kong.SNI{
					Name: new("bar"),
					Tags: []*string{
						new("bar-tag1"),
					},
				},
			},
		},
		{
			name:     "empty input",
			entities: []any{},
			want:     []any{},
		},
		{
			name: "all konnect managed",
			entities: []any{
				&kong.SNI{
					Name: new("sni1"),
					Tags: []*string{
						new("konnect-managed"),
					},
				},
				&kong.SNI{
					Name: new("sni2"),
					Tags: []*string{
						new("konnect-managed"),
					},
				},
			},
			want: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := excludeKonnectManagedEntities(tt.entities)
			if err != nil {
				t.Errorf("excludeKonnectManagedEntities() error = %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("excludeKonnectManagedPlugins() = %v, want %v", got, tt.want)
			}
		})
	}
}
