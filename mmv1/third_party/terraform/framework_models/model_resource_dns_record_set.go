package google

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// GoogleDnsRecordSetResourceModel describes the resource data model.
type GoogleDnsRecordSetResourceModel struct {
	Id            types.String `tfsdk:"id"`
	ManagedZone   types.String `tfsdk:"managed_zone"`
	Name          types.String `tfsdk:"name"`
	Rrdatas       types.List   `tfsdk:"rrdatas"`
	RoutingPolicy types.List   `tfsdk:"routing_policy"`
	Ttl           types.Int64  `tfsdk:"ttl"`
	Type          types.String `tfsdk:"type"`
	Project       types.String `tfsdk:"project"`
}

// GoogleDnsRecordSetRoutingPolicyModel describes the nested routing policy data model.
type GoogleDnsRecordSetRoutingPolicyModel struct {
	Wrr              types.List `tfsdk:"wrr"`
	Geo              types.List `tfsdk:"geo"`
	EnableGeoFencing types.Bool `tfsdk:"enable_geo_fencing"`
	PrimaryBackup    types.List `tfsdk:"primary_backup"`
}

// GoogleDnsRecordSetRoutingPolicyWrrModel describes the nested routing policy data model.
type GoogleDnsRecordSetRoutingPolicyWrrModel struct {
	Weight               types.Float64 `tfsdk:"weight"`
	Rrdatas              types.List    `tfsdk:"rrdatas"`
	HealthCheckedTargets types.List    `tfsdk:"health_checked_targets"`
}

// GoogleDnsRecordSetRoutingPolicyTargetModel describes the nested routing policy data model.
type GoogleDnsRecordSetRoutingPolicyTargetModel struct {
	InternalLoadBalancers types.List `tfsdk:"internal_load_balancers"`
}

// GoogleDnsRecordSetRoutingPolicyTargetIlbModel describes the nested routing policy data model.
type GoogleDnsRecordSetRoutingPolicyTargetIlbModel struct {
	LoadBalancerType types.String `tfsdk:"load_balancer_type"`
	IpAddress        types.String `tfsdk:"ip_address"`
	Port             types.String `tfsdk:"port"`
	IpProtocol       types.String `tfsdk:"ip_protocol"`
	NetworkUrl       types.String `tfsdk:"network_url"`
	Project          types.String `tfsdk:"project"`
	Region           types.String `tfsdk:"region"`
}

// GoogleDnsRecordSetRoutingPolicyGeoModel describes the nested routing policy data model.
type GoogleDnsRecordSetRoutingPolicyGeoModel struct {
	Location             types.String `tfsdk:"location"`
	Rrdatas              types.List   `tfsdk:"rrdatas"`
	HealthCheckedTargets types.List   `tfsdk:"health_checked_targets"`
}

// GoogleDnsRecordSetRoutingPolicyPrimaryBackupModel describes the nested routing policy data model.
type GoogleDnsRecordSetRoutingPolicyPrimaryBackupModel struct {
	Primary                    types.List    `tfsdk:"primary"`
	BackupGeo                  types.List    `tfsdk:"backup_geo"`
	EnableGeoFencingForBackups types.Bool    `tfsdk:"enable_geo_fencing_for_backups"`
	TrickleRatio               types.Float64 `tfsdk:"trickle_ratio"`
}

func (m *GoogleDnsRecordSetResourceModel) ReadUrl() string {
	return fmt.Sprintf("projects/%s/managedZones/%s/rrsets/%s/%s", m.Project.ValueString(), m.ManagedZone.ValueString(), m.Name.ValueString(), m.Type.ValueString())
}

func (m *GoogleDnsRecordSetResourceModel) ParseImportId(ctx context.Context, schemaAttrTypes map[string]attr.Type, diags *diag.Diagnostics) {
	importIds := []string{
		"projects/(?P<project>[^/]+)/managedZones/(?P<managed_zone>[^/]+)/rrsets/(?P<name>[^/]+)/(?P<type>[^/]+)",
		"(?P<project>[^/]+)/(?P<managed_zone>[^/]+)/(?P<name>[^/]+)/(?P<type>[^/]+)",
		"(?P<managed_zone>[^/]+)/(?P<name>[^/]+)/(?P<type>[^/]+)",
	}

	attrVals := ParseFrameworkImportId(ctx, importIds, m.Id.ValueString(), schemaAttrTypes, diags)

	m.ManagedZone = attrVals["managed_zone"].(types.String)
	m.Name = attrVals["name"].(types.String)
	m.Type = attrVals["type"].(types.String)

	m.Id = types.StringValue(m.ReadUrl())
}
