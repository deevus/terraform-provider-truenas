package datasources

import (
	"context"
	"errors"
	"testing"

	truenas "github.com/deevus/truenas-go"
	"github.com/deevus/terraform-provider-truenas/internal/services"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewSnapshotsDataSource(t *testing.T) {
	ds := NewSnapshotsDataSource()
	if ds == nil {
		t.Fatal("expected non-nil data source")
	}

	_ = datasource.DataSource(ds)
	_ = datasource.DataSourceWithConfigure(ds.(*SnapshotsDataSource))
}

func TestSnapshotsDataSource_Metadata(t *testing.T) {
	ds := NewSnapshotsDataSource()

	req := datasource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_snapshots" {
		t.Errorf("expected TypeName 'truenas_snapshots', got %q", resp.TypeName)
	}
}

func TestSnapshotsDataSource_Schema(t *testing.T) {
	ds := NewSnapshotsDataSource()

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(context.Background(), req, resp)

	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify dataset_id is required
	datasetIDAttr, ok := resp.Schema.Attributes["dataset_id"]
	if !ok {
		t.Fatal("expected 'dataset_id' attribute in schema")
	}
	if !datasetIDAttr.IsRequired() {
		t.Error("expected 'dataset_id' attribute to be required")
	}

	// Verify snapshots is computed
	snapshotsAttr, ok := resp.Schema.Attributes["snapshots"]
	if !ok {
		t.Fatal("expected 'snapshots' attribute in schema")
	}
	if !snapshotsAttr.IsComputed() {
		t.Error("expected 'snapshots' attribute to be computed")
	}
}

func TestSnapshotsDataSource_Configure_Success(t *testing.T) {
	ds := NewSnapshotsDataSource().(*SnapshotsDataSource)

	svc := &services.TrueNASServices{}

	req := datasource.ConfigureRequest{
		ProviderData: svc,
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func getSnapshotsDataSourceSchema(t *testing.T) datasource.SchemaResponse {
	t.Helper()
	ds := NewSnapshotsDataSource()
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)
	return *schemaResp
}

func TestSnapshotsDataSource_Read_Success(t *testing.T) {
	ds := &SnapshotsDataSource{
		services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				ListFunc: func(ctx context.Context) ([]truenas.Snapshot, error) {
					return []truenas.Snapshot{
						{
							ID:           "tank/data@snap1",
							Dataset:      "tank/data",
							SnapshotName: "snap1",
							Used:         1024,
							Referenced:   2048,
							HasHold:      false,
						},
						{
							ID:           "tank/data@snap2",
							Dataset:      "tank/data",
							SnapshotName: "snap2",
							Used:         512,
							Referenced:   1024,
							HasHold:      true,
						},
					}, nil
				},
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, nil),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotsDataSourceModel
	resp.State.Get(context.Background(), &data)

	if len(data.Snapshots) != 2 {
		t.Errorf("expected 2 snapshots, got %d", len(data.Snapshots))
	}
}

func TestSnapshotsDataSource_Read_Empty(t *testing.T) {
	ds := &SnapshotsDataSource{
		services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				ListFunc: func(ctx context.Context) ([]truenas.Snapshot, error) {
					return []truenas.Snapshot{}, nil
				},
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, nil),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotsDataSourceModel
	resp.State.Get(context.Background(), &data)

	if len(data.Snapshots) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(data.Snapshots))
	}
}

func TestSnapshotsDataSource_Configure_NilProviderData(t *testing.T) {
	ds := NewSnapshotsDataSource().(*SnapshotsDataSource)

	req := datasource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors for nil provider data: %v", resp.Diagnostics)
	}
}

func TestSnapshotsDataSource_Configure_WrongType(t *testing.T) {
	ds := NewSnapshotsDataSource().(*SnapshotsDataSource)

	req := datasource.ConfigureRequest{
		ProviderData: "not a services",
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong provider data type")
	}
}

func TestSnapshotsDataSource_Read_APIError(t *testing.T) {
	ds := &SnapshotsDataSource{
		services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				ListFunc: func(ctx context.Context) ([]truenas.Snapshot, error) {
					return nil, errors.New("connection refused")
				},
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, nil),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API failure")
	}
}

func TestSnapshotsDataSource_Read_Recursive(t *testing.T) {
	ds := &SnapshotsDataSource{
		services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				ListFunc: func(ctx context.Context) ([]truenas.Snapshot, error) {
					return []truenas.Snapshot{
						{
							ID:           "tank/data@snap1",
							Dataset:      "tank/data",
							SnapshotName: "snap1",
							Used:         1024,
							Referenced:   2048,
							HasHold:      false,
						},
						{
							ID:           "tank/data/child@snap2",
							Dataset:      "tank/data/child",
							SnapshotName: "snap2",
							Used:         512,
							Referenced:   1024,
							HasHold:      false,
						},
						{
							ID:           "tank/other@snap3",
							Dataset:      "tank/other",
							SnapshotName: "snap3",
							Used:         256,
							Referenced:   512,
							HasHold:      false,
						},
					}, nil
				},
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, true),
		"name_pattern": tftypes.NewValue(tftypes.String, nil),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotsDataSourceModel
	resp.State.Get(context.Background(), &data)

	// Should include tank/data and tank/data/child, but not tank/other
	if len(data.Snapshots) != 2 {
		t.Errorf("expected 2 snapshots, got %d", len(data.Snapshots))
	}
}

func TestSnapshotsDataSource_Read_NamePattern_Match(t *testing.T) {
	ds := &SnapshotsDataSource{
		services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				ListFunc: func(ctx context.Context) ([]truenas.Snapshot, error) {
					return []truenas.Snapshot{
						{
							ID:           "tank/data@pre-upgrade-1",
							Dataset:      "tank/data",
							SnapshotName: "pre-upgrade-1",
							Used:         1024,
							Referenced:   2048,
							HasHold:      false,
						},
						{
							ID:           "tank/data@post-upgrade",
							Dataset:      "tank/data",
							SnapshotName: "post-upgrade",
							Used:         512,
							Referenced:   1024,
							HasHold:      false,
						},
						{
							ID:           "tank/data@pre-upgrade-2",
							Dataset:      "tank/data",
							SnapshotName: "pre-upgrade-2",
							Used:         256,
							Referenced:   512,
							HasHold:      false,
						},
					}, nil
				},
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, "pre-*"),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotsDataSourceModel
	resp.State.Get(context.Background(), &data)

	// Should only match pre-upgrade-1 and pre-upgrade-2
	if len(data.Snapshots) != 2 {
		t.Errorf("expected 2 snapshots matching 'pre-*', got %d", len(data.Snapshots))
	}
}

func TestSnapshotsDataSource_Read_NamePattern_NoMatch(t *testing.T) {
	ds := &SnapshotsDataSource{
		services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				ListFunc: func(ctx context.Context) ([]truenas.Snapshot, error) {
					return []truenas.Snapshot{
						{
							ID:           "tank/data@snap1",
							Dataset:      "tank/data",
							SnapshotName: "snap1",
							Used:         1024,
							Referenced:   2048,
							HasHold:      false,
						},
					}, nil
				},
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, "backup-*"),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotsDataSourceModel
	resp.State.Get(context.Background(), &data)

	if len(data.Snapshots) != 0 {
		t.Errorf("expected 0 snapshots matching 'backup-*', got %d", len(data.Snapshots))
	}
}

func TestSnapshotsDataSource_Read_NamePattern_Invalid(t *testing.T) {
	ds := &SnapshotsDataSource{
		services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				ListFunc: func(ctx context.Context) ([]truenas.Snapshot, error) {
					return []truenas.Snapshot{
						{
							ID:           "tank/data@snap1",
							Dataset:      "tank/data",
							SnapshotName: "snap1",
							Used:         1024,
							Referenced:   2048,
							HasHold:      false,
						},
					}, nil
				},
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, "[invalid"),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid glob pattern")
	}
}

func TestSnapshotsDataSource_Read_GetConfigError(t *testing.T) {
	ds := &SnapshotsDataSource{
		services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)
	// Create an invalid config with wrong type for dataset_id (number instead of string)
	invalidValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.Number, // Wrong type - should be String
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.Number, 12345), // Wrong value type
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, nil),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    invalidValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid config value")
	}
}

func TestSnapshotsDataSource_Read_FiltersOtherDatasets(t *testing.T) {
	ds := &SnapshotsDataSource{
		services: &services.TrueNASServices{
			Snapshot: &truenas.MockSnapshotService{
				ListFunc: func(ctx context.Context) ([]truenas.Snapshot, error) {
					return []truenas.Snapshot{
						{
							ID:           "tank/data@snap1",
							Dataset:      "tank/data",
							SnapshotName: "snap1",
							Used:         1024,
							Referenced:   2048,
							HasHold:      false,
						},
						{
							ID:           "tank/other@snap2",
							Dataset:      "tank/other",
							SnapshotName: "snap2",
							Used:         512,
							Referenced:   1024,
							HasHold:      false,
						},
					}, nil
				},
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, nil),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotsDataSourceModel
	resp.State.Get(context.Background(), &data)

	// Should only include tank/data snapshot, not tank/other
	if len(data.Snapshots) != 1 {
		t.Errorf("expected 1 snapshot, got %d", len(data.Snapshots))
	}
}
