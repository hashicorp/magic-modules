package google

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"google.golang.org/api/dns/v1"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-provider-google/google/tpgresource"
	transport_tpg "github.com/hashicorp/terraform-provider-google/google/transport"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &GoogleDnsRecordSetResource{}
var _ resource.ResourceWithImportState = &GoogleDnsRecordSetResource{}
var _ resource.ResourceWithConfigValidators = &GoogleDnsRecordSetResource{}

func NewGoogleDnsRecordSetResource() resource.Resource {
	return &GoogleDnsRecordSetResource{}
}

// GoogleDnsRecordSetResource defines the resource implementation.
type GoogleDnsRecordSetResource struct {
	client  *dns.Service
	project types.String
	provider *frameworkProvider
}

func (r *GoogleDnsRecordSetResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		//resourcevalidator.Conflicting(
		//	path.MatchRoot("routing_policy.0.wrr"),
		//	path.MatchRoot("routing_policy.0.enable_geo_fencing"),
		//	path.MatchRoot("routing_policy.0.primary_backup"),
		//),
		//resourcevalidator.ExactlyOneOf(
		//	path.MatchRoot("routing_policy.0.wrr"),
		//	path.MatchRoot("routing_policy.0.geo"),
		//	path.MatchRoot("routing_policy.0.primary_backup"),
		//),
		//resourcevalidator.ExactlyOneOf(
		//	path.MatchRoot("rrdatas"),
		//	path.MatchRoot("routing_policy"),
		//),
	}
}

func (r *GoogleDnsRecordSetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record_set"
}

func (r *GoogleDnsRecordSetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	p, ok := req.ProviderData.(*frameworkProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *frameworkProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = p.NewDnsClient(p.userAgent, &resp.Diagnostics)
	r.project = p.project
	r.provider = p
}

func (r *GoogleDnsRecordSetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "A DNS record set within Google Cloud DNS",
		Description:         "A DNS record set within Google Cloud DNS",
		Attributes: map[string]schema.Attribute{
			"managed_zone": schema.StringAttribute{
				Description:         "The name of the zone in which this record set will reside.",
				MarkdownDescription: "The name of the zone in which this record set will reside.",
				Required:            true,
				// 				DiffSuppressFunc: compareSelfLinkOrResourceName,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The DNS name this record set will apply to.",
				MarkdownDescription: "The DNS name this record set will apply to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					TrailingDotValidator(),
				},
			},
			"type": schema.StringAttribute{
				Description:         "The DNS record set type.",
				MarkdownDescription: "The DNS record set type.",
				Required:            true,
			},
			"rrdatas": schema.ListAttribute{
				Description: "The string data for the records in this record set whose meaning depends on the DNS type. " +
					"For TXT record, if the string data contains spaces, add surrounding `\\\"` if you don't want your string to get split on spaces. " +
					"To specify a single record value longer than 255 characters such as a TXT record for DKIM, " +
					"add `\\\" \\\"` inside the Terraform configuration string (e.g. `\"first255characters\\\" \\\"morecharacters\"`).",
				MarkdownDescription: "The string data for the records in this record set whose meaning depends on the DNS type. " +
					"For TXT record, if the string data contains spaces, add surrounding `\\\"` if you don't want your string to get split on spaces. " +
					"To specify a single record value longer than 255 characters such as a TXT record for DKIM, " +
					"add `\\\" \\\"` inside the Terraform configuration string (e.g. `\"first255characters\\\" \\\"morecharacters\"`).",
				Optional:    true,
				ElementType: types.StringType,
				// DiffSuppressFunc: rrdatasDnsDiffSuppress,
			},
			"ttl": schema.Int64Attribute{
				Description:         "The time-to-live of this record set (seconds).",
				MarkdownDescription: "The time-to-live of this record set (seconds).",
				Optional:            true,
			},
			"project": schema.StringAttribute{
				Description:         "The ID of the project in which the resource belongs. If it is not provided, the provider project is used.",
				MarkdownDescription: "The ID of the project in which the resource belongs. If it is not provided, the provider project is used.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Description:         "DNS record set identifier",
				MarkdownDescription: "DNS record set identifier",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"routing_policy": schema.ListNestedBlock{
				Description:         "The configuration for steering traffic based on query. You can specify either Weighted Round Robin(WRR) type or Geolocation(GEO) type.",
				MarkdownDescription: "The configuration for steering traffic based on query. You can specify either Weighted Round Robin(WRR) type or Geolocation(GEO) type.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"enable_geo_fencing": schema.BoolAttribute{
							Description:         "Specifies whether to enable fencing for geo queries.",
							MarkdownDescription: "Specifies whether to enable fencing for geo queries.",
							Optional:            true,
						},
					},
					Blocks: map[string]schema.Block{
						"wrr": schema.ListNestedBlock{
							Description:         "The configuration for Weighted Round Robin based routing policy.",
							MarkdownDescription: "The configuration for Weighted Round Robin based routing policy.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"weight": schema.Float64Attribute{
										Description:         "The ratio of traffic routed to the target.",
										MarkdownDescription: "The ratio of traffic routed to the target.",
										Required:            true,
									},
									"rrdatas": schema.ListAttribute{
										Description: "The string data for the records in this record set whose meaning depends on the DNS type. " +
											"For TXT record, if the string data contains spaces, add surrounding `\\\"` if you don't want your string to get split on spaces. " +
											"To specify a single record value longer than 255 characters such as a TXT record for DKIM, " +
											"add `\\\" \\\"` inside the Terraform configuration string (e.g. `\"first255characters\\\" \\\"morecharacters\"`).",
										MarkdownDescription: "The string data for the records in this record set whose meaning depends on the DNS type. " +
											"For TXT record, if the string data contains spaces, add surrounding `\\\"` if you don't want your string to get split on spaces. " +
											"To specify a single record value longer than 255 characters such as a TXT record for DKIM, " +
											"add `\\\" \\\"` inside the Terraform configuration string (e.g. `\"first255characters\\\" \\\"morecharacters\"`).",
										Optional:    true,
										ElementType: types.StringType,
									},
								},
								Blocks: map[string]schema.Block{
									"health_checked_targets": schema.ListNestedBlock{
										Description: "The list of targets to be health checked. Note that if DNSSEC is enabled for this zone, only one of `rrdatas` " +
											"or `health_checked_targets` can be set.",
										MarkdownDescription: "The list of targets to be health checked. Note that if DNSSEC is enabled for this zone, only one of `rrdatas` " +
											"or `health_checked_targets` can be set.",
										NestedObject: schema.NestedBlockObject{
											Blocks: routingPolicyTargetObject(),
										},
									},
								},
							},
						},
						"geo": schema.ListNestedBlock{
							Description:         `The configuration for Geo location based routing policy.`,
							MarkdownDescription: `The configuration for Geo location based routing policy.`,
							NestedObject:        routingPolicyGeoObject(),
						},
						"primary_backup": schema.ListNestedBlock{
							Description: "The configuration for a primary-backup policy with global to regional failover. " +
								"Queries are responded to with the global primary targets, but if none of the primary targets are healthy, " +
								"then we fallback to a regional failover policy.",
							MarkdownDescription: "The configuration for a primary-backup policy with global to regional failover. " +
								"Queries are responded to with the global primary targets, but if none of the primary targets are healthy, " +
								"then we fallback to a regional failover policy.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"enable_geo_fencing_for_backups": schema.BoolAttribute{
										Description:         "Specifies whether to enable fencing for backup geo queries.",
										MarkdownDescription: "Specifies whether to enable fencing for backup geo queries.",
										Optional:            true,
									},
									"trickle_ratio": schema.Float64Attribute{
										Description:         "Specifies the percentage of traffic to send to the backup targets even when the primary targets are healthy.",
										MarkdownDescription: "Specifies the percentage of traffic to send to the backup targets even when the primary targets are healthy.",
										Optional:            true,
									},
								},
								Blocks: map[string]schema.Block{
									"primary": schema.ListNestedBlock{
										Description:         "The list of global primary targets to be health checked.",
										MarkdownDescription: "The list of global primary targets to be health checked.",
										NestedObject: schema.NestedBlockObject{
											Blocks: routingPolicyTargetObject(),
										},
									},
									"backup_geo": schema.ListNestedBlock{
										Description:         "The backup geo targets, which provide a regional failover policy for the otherwise global primary targets.",
										MarkdownDescription: "The backup geo targets, which provide a regional failover policy for the otherwise global primary targets.",
										NestedObject:        routingPolicyGeoObject(),
									},
								},
							},
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
		},
	}
}

func (r *GoogleDnsRecordSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GoogleDnsRecordSetResourceModel
	var metaData *ProviderMetaModel
	var diags diag.Diagnostics

	// Read Provider meta into the meta model
	resp.Diagnostics.Append(req.ProviderMeta.Get(ctx, &metaData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.client.UserAgent = generateFrameworkUserAgentString(metaData, r.client.UserAgent)

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Project = getProjectFramework(data.Project, r.project, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the change
	rset := &dns.ResourceRecordSet{
		Name: data.Name.ValueString(),
		Type: data.Type.ValueString(),
		Ttl:  data.Ttl.ValueInt64(),
	}

	if !data.Rrdatas.IsNull() {
		rset.Rrdatas = expandDnsRecordSetRrdata(ctx, data.Rrdatas, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !data.RoutingPolicy.IsNull() {
		rset.RoutingPolicy = expandDnsRecordSetRoutingPolicy(ctx, r.provider, data.Project, data.RoutingPolicy, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	chg := &dns.Change{
		Additions: []*dns.ResourceRecordSet{rset},
	}

	// The terraform provider is authoritative, so what we do here is check if
	// any records that we are trying to create already exist and make sure we
	// delete them, before adding in the changes requested.  Normally this would
	// result in an AlreadyExistsError.
	tflog.Debug(ctx, fmt.Sprintf("DNS record list request for %s", data.ManagedZone.ValueString()))
	res, err := r.client.ResourceRecordSets.List(data.Project.ValueString(), data.ManagedZone.ValueString()).Do()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error retrieving record sets for %s", data.ManagedZone.ValueString()), err.Error())
		return
	}
	var deletions []*dns.ResourceRecordSet

	for _, record := range res.Rrsets {
		if record.Type != data.Type.ValueString() || record.Name != data.Name.ValueString() {
			continue
		}
		deletions = append(deletions, record)
	}
	if len(deletions) > 0 {
		chg.Deletions = deletions
	}

	tflog.Debug(ctx, fmt.Sprintf("DNS Record create request: %#v", chg))
	chg, err = r.client.Changes.Create(data.Project.ValueString(), data.ManagedZone.ValueString(), chg).Do()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error creating DNS RecordSet"), err.Error())
		return
	}

	data.Id = types.StringValue(fmt.Sprintf("projects/%s/managedZones/%s/rrsets/%s/%s",
		data.Project.ValueString(), data.ManagedZone.ValueString(), data.Name.ValueString(), data.Type.ValueString()))

	w := &DnsChangeWaiter{
		Service:     r.client,
		Change:      chg,
		Project:     data.Project.ValueString(),
		ManagedZone: data.ManagedZone.ValueString(),
	}

	_, err = w.Conf().WaitForState()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error waiting for Google DNS change"), err.Error())
		return
	}

	tflog.Trace(ctx, "created dns record set resource")

	clientResp, err := r.client.ResourceRecordSets.List(data.Project.ValueString(), data.ManagedZone.ValueString()).Name(data.Name.ValueString()).Type(data.Type.ValueString()).Do()
	if err != nil {
		handleResourceNotFoundError(ctx, err, &resp.State, fmt.Sprintf("resourceDnsRecordSet %q", data.Name.ValueString()), &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if len(clientResp.Rrsets) != 1 {
		resp.Diagnostics.AddError("expected 1 record set", fmt.Sprintf("%d record sets were returned", len(clientResp.Rrsets)))
		return
	}

	tflog.Trace(ctx, "read dns record set resource")

	data.Type = types.StringValue(clientResp.Rrsets[0].Type)
	data.Ttl = types.Int64Value(clientResp.Rrsets[0].Ttl)
	data.Rrdatas, diags = types.ListValueFrom(ctx, types.StringType, clientResp.Rrsets[0].Rrdatas)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if clientResp.Rrsets[0].RoutingPolicy != nil {
		data.RoutingPolicy, diags = types.ListValueFrom(ctx, data.RoutingPolicy.ElementType(ctx), []*dns.RRSetRoutingPolicy{clientResp.Rrsets[0].RoutingPolicy})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GoogleDnsRecordSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GoogleDnsRecordSetResourceModel
	var metaData *ProviderMetaModel
	var diags diag.Diagnostics

	// Read Provider meta into the meta model
	resp.Diagnostics.Append(req.ProviderMeta.Get(ctx, &metaData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.client.UserAgent = generateFrameworkUserAgentString(metaData, r.client.UserAgent)

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Project = getProjectFramework(data.Project, r.project, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	var clientResp *dns.ResourceRecordSetsListResponse
	err := retry(func() error {
		var reqErr error
		clientResp, reqErr = r.client.ResourceRecordSets.List(
			data.Project.ValueString(), data.ManagedZone.ValueString()).Name(data.Name.ValueString()).Type(data.Type.ValueString()).Do()
		return reqErr
	})
	if err != nil {
		handleResourceNotFoundError(ctx, err, &resp.State, fmt.Sprintf("resourceDnsRecordSet %q", data.Name.ValueString()), &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if len(clientResp.Rrsets) == 0 {
		// The resource doesn't exist anymore
		resp.State.RemoveResource(ctx)
		return
	}

	if len(clientResp.Rrsets) > 1 {
		resp.Diagnostics.AddError("only expected 1 record set", fmt.Sprintf("%d record sets were returned", len(clientResp.Rrsets)))
	}

	tflog.Trace(ctx, "read dns record set resource")

	data.Type = types.StringValue(clientResp.Rrsets[0].Type)
	data.Ttl = types.Int64Value(clientResp.Rrsets[0].Ttl)
	data.Rrdatas, diags = types.ListValueFrom(ctx, types.StringType, clientResp.Rrsets[0].Rrdatas)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if clientResp.Rrsets[0].RoutingPolicy != nil {
		data.RoutingPolicy, diags = types.ListValueFrom(ctx, data.RoutingPolicy.ElementType(ctx), []*dns.RRSetRoutingPolicy{clientResp.Rrsets[0].RoutingPolicy})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GoogleDnsRecordSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var configData GoogleDnsRecordSetResourceModel
	var stateData GoogleDnsRecordSetResourceModel
	var metaData *ProviderMetaModel
	var diags diag.Diagnostics

	// Read Provider meta into the meta model
	resp.Diagnostics.Append(req.ProviderMeta.Get(ctx, &metaData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.client.UserAgent = generateFrameworkUserAgentString(metaData, r.client.UserAgent)

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &configData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	configData.Project = getProjectFramework(configData.Project, r.project, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	stateRrdatas := expandDnsRecordSetRrdata(ctx, stateData.Rrdatas, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	configRrdatas := expandDnsRecordSetRrdata(ctx, configData.Rrdatas, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	//stateRoutingPolicy := expandDnsRecordSetRoutingPolicy(ctx, stateData.RoutingPolicy, &resp.Diagnostics)
	//if resp.Diagnostics.HasError() {
	//	return
	//}

	configRoutingPolicy := expandDnsRecordSetRoutingPolicy(ctx, r.provider, configData.RoutingPolicy, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	chg := &dns.Change{
		Deletions: []*dns.ResourceRecordSet{
			{
				Name:    stateData.Name.ValueString(),
				Type:    stateData.Type.ValueString(),
				Ttl:     stateData.Ttl.ValueInt64(),
				Rrdatas: stateRrdatas,
				// RoutingPolicy: stateRoutingPolicy,
			},
		},
		Additions: []*dns.ResourceRecordSet{
			{
				Name:          stateData.Name.ValueString(),
				Type:          configData.Type.ValueString(),
				Ttl:           configData.Ttl.ValueInt64(),
				Rrdatas:       configRrdatas,
				RoutingPolicy: configRoutingPolicy,
			},
		},
	}

	tflog.Debug(ctx, fmt.Sprintf("DNS Record change request: %#v old: %#v new: %#v", chg, chg.Deletions[0], chg.Additions[0]))

	chg, err := r.client.Changes.Create(configData.Project.ValueString(), stateData.ManagedZone.ValueString(), chg).Do()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error changing DNS RecordSet"), err.Error())
		return
	}

	w := &DnsChangeWaiter{
		Service:     r.client,
		Change:      chg,
		Project:     configData.Project.ValueString(),
		ManagedZone: stateData.ManagedZone.ValueString(),
	}

	_, err = w.Conf().WaitForState()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error waiting for Google DNS change"), err.Error())
		return
	}

	stateData.Id = types.StringValue(fmt.Sprintf("projects/%s/managedZones/%s/rrsets/%s/%s",
		configData.Project.ValueString(), stateData.ManagedZone.ValueString(), stateData.Name.ValueString(), configData.Type.ValueString()))

	var clientResp *dns.ResourceRecordSetsListResponse
	err = retry(func() error {
		var reqErr error
		clientResp, reqErr = r.client.ResourceRecordSets.List(
			stateData.Project.ValueString(), stateData.ManagedZone.ValueString()).Name(stateData.Name.ValueString()).Type(stateData.Type.ValueString()).Do()
		return reqErr
	})
	if err != nil {
		handleResourceNotFoundError(ctx, err, &resp.State, fmt.Sprintf("resourceDnsRecordSet %q", stateData.Name.ValueString()), &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if len(clientResp.Rrsets) == 0 {
		// The resource doesn't exist anymore
		resp.State.RemoveResource(ctx)
		return
	}

	if len(clientResp.Rrsets) > 1 {
		resp.Diagnostics.AddError("only expected 1 record set", fmt.Sprintf("%d record sets were returned", len(clientResp.Rrsets)))
	}

	tflog.Trace(ctx, "read dns record set resource")

	stateData.Type = types.StringValue(clientResp.Rrsets[0].Type)
	stateData.Ttl = types.Int64Value(clientResp.Rrsets[0].Ttl)
	stateData.Rrdatas, diags = types.ListValueFrom(ctx, types.StringType, clientResp.Rrsets[0].Rrdatas)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if clientResp.Rrsets[0].RoutingPolicy != nil {
		stateData.RoutingPolicy, diags = types.ListValueFrom(ctx, stateData.RoutingPolicy.ElementType(ctx), []*dns.RRSetRoutingPolicy{clientResp.Rrsets[0].RoutingPolicy})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &configData)...)
}

func (r *GoogleDnsRecordSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GoogleDnsRecordSetResourceModel
	var metaData *ProviderMetaModel

	// Read Provider meta into the meta model
	resp.Diagnostics.Append(req.ProviderMeta.Get(ctx, &metaData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.client.UserAgent = generateFrameworkUserAgentString(metaData, r.client.UserAgent)

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Project = getProjectFramework(data.Project, r.project, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// NS records must always have a value, so we short-circuit delete
	// this allows terraform delete to work, but may have unexpected
	// side-effects when deleting just that record set.
	// Unfortunately, you can set NS records on subdomains, and those
	// CAN and MUST be deleted, so we need to retrieve the managed zone,
	// check if what we're looking at is a subdomain, and only not delete
	// if it's not actually a subdomain
	if data.Type.ValueString() == "NS" {
		mz, err := r.client.ManagedZones.Get(data.Project.ValueString(), data.ManagedZone.ValueString()).Do()
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("error retrieving managed zone %s from project %s", data.ManagedZone.ValueString(), data.Project.ValueString()), err.Error())
			return
		}
		domain := mz.DnsName

		if domain == data.Name.ValueString() {
			tflog.Debug(ctx, "NS records can't be deleted due to API restrictions, so they're being left in place. See https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_record_set for more information.")
			return
		}
	}

	//routingPolicy := expandDnsRecordSetRoutingPolicy(ctx, data.RoutingPolicy, &resp.Diagnostics)
	//if resp.Diagnostics.HasError() {
	//	return
	//}

	rrdata := expandDnsRecordSetRrdata(ctx, data.Rrdatas, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the change
	chg := &dns.Change{
		Deletions: []*dns.ResourceRecordSet{
			{
				Name:    data.Name.ValueString(),
				Type:    data.Type.ValueString(),
				Ttl:     data.Ttl.ValueInt64(),
				Rrdatas: rrdata,
				//RoutingPolicy: routingPolicy,
			},
		},
	}

	chg, err := r.client.Changes.Create(data.Project.ValueString(), data.ManagedZone.ValueString(), chg).Do()
	if err != nil {
		handleResourceNotFoundError(ctx, err, &resp.State, fmt.Sprintf("resourceDnsRecordSet %q", data.Name.ValueString()), &resp.Diagnostics)
		return
	}

	w := &DnsChangeWaiter{
		Service:     r.client,
		Change:      chg,
		Project:     data.Project.ValueString(),
		ManagedZone: data.ManagedZone.ValueString(),
	}

	_, err = w.Conf().WaitForState()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error waiting for Google DNS change"), err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *GoogleDnsRecordSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	data := GoogleDnsRecordSetResourceModel{
		Id: types.StringValue(req.ID),
	}

	data.Project = getProjectFramework(data.Project, r.project, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ParseImportId(ctx, GetSchemaAttributeTypes(resp.State.Schema.(schema.Schema)), &resp.Diagnostics)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), &data.Id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), &data.Project)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), &data.Name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), &data.Type)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("managed_zone"), &data.ManagedZone)...)
}

// routingPolicyGeoObject is a helper function for the routing_policy.geo schema and
// is also used by routing_policy.primary_backup.backup_geo schema
func routingPolicyGeoObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"location": schema.StringAttribute{
				Description:         "The location name defined in Google Cloud.",
				MarkdownDescription: "The location name defined in Google Cloud.",
				Required:            true,
			},
			"rrdatas": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"health_checked_targets": schema.ListNestedBlock{
				Description:         "For A and AAAA types only. The list of targets to be health checked. These can be specified along with `rrdatas` within this item.",
				MarkdownDescription: "For A and AAAA types only. The list of targets to be health checked. These can be specified along with `rrdatas` within this item.",
				NestedObject: schema.NestedBlockObject{
					Blocks: routingPolicyTargetObject(),
				},
			},
		},
	}
}

// routingPolicyTargetObject is a helper function for the routing_policy.wrr.health_checked_targets, routing_policy.geo.health_checked_targets schema and
// is also used by routing_policy.primary_backup.primary schema
func routingPolicyTargetObject() map[string]schema.Block {
	return map[string]schema.Block{
		"internal_load_balancers": schema.ListNestedBlock{
			NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"load_balancer_type": schema.StringAttribute{
						Description:         "The type of load balancer. This value is case-sensitive. Possible values: [\"regionalL4ilb\"]",
						MarkdownDescription: "The type of load balancer. This value is case-sensitive. Possible values: [\"regionalL4ilb\"]",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.OneOf([]string{"regionalL4ilb"}...),
						},
					},
					"ip_address": schema.StringAttribute{
						Description:         "The frontend IP address of the load balancer.",
						MarkdownDescription: "The frontend IP address of the load balancer.",
						Required:            true,
					},
					"port": schema.StringAttribute{
						Description:         "The configured port of the load balancer.",
						MarkdownDescription: "The configured port of the load balancer.",
						Required:            true,
					},
					"ip_protocol": schema.StringAttribute{
						Description:         "The configured IP protocol of the load balancer. This value is case-sensitive. Possible values: [\"tcp\", \"udp\"]",
						MarkdownDescription: "The configured IP protocol of the load balancer. This value is case-sensitive. Possible values: [\"tcp\", \"udp\"]",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.OneOf([]string{"tcp", "udp"}...),
						},
					},
					"network_url": schema.StringAttribute{
						Description: "The fully qualified url of the network in which the load balancer belongs. " +
							"This should be formatted like `https://www.googleapis.com/compute/v1/projects/{project}/global/networks/{network}`.",
						MarkdownDescription: "The fully qualified url of the network in which the load balancer belongs. " +
							"This should be formatted like `https://www.googleapis.com/compute/v1/projects/{project}/global/networks/{network}`.",
						Required: true,
						// 			DiffSuppressFunc: compareSelfLinkOrResourceName,
					},
					"project": schema.StringAttribute{
						Description:         "The ID of the project in which the load balancer belongs.",
						MarkdownDescription: "The ID of the project in which the load balancer belongs.",
						Required:            true,
					},
					"region": schema.StringAttribute{
						Description:         "The region of the load balancer. Only needed for regional load balancers.",
						MarkdownDescription: "The region of the load balancer. Only needed for regional load balancers.",
						Optional:            true,
					},
				},
			},
		},
	}
}

func expandDnsRecordSetRrdata(ctx context.Context, configured types.List, diags *diag.Diagnostics) []string {
	var rrdatas []string
	diags.Append(configured.ElementsAs(ctx, &rrdatas, false)...)
	return rrdatas
}

func expandDnsRecordSetRoutingPolicy(ctx context.Context, project types.String, p *frameworkProvider, configured types.List, diags *diag.Diagnostics) *dns.RRSetRoutingPolicy {
	var routingPolicyObject dns.RRSetRoutingPolicy
	var routingPolicies []GoogleDnsRecordSetRoutingPolicyModel

	d := configured.ElementsAs(ctx, &routingPolicies, true)
	diags.Append(d...)
	if diags.HasError() {
		return &routingPolicyObject
	}

	routingPolicy := routingPolicies[0]

	if !routingPolicy.Wrr.IsNull() {
		wrrItems := expandDnsRecordSetRoutingPolicyWrrItems(ctx, p, project, routingPolicy.Wrr, diags)
		if diags.HasError() {
			return &routingPolicyObject
		}
		routingPolicyObject.Wrr = &dns.RRSetRoutingPolicyWrrPolicy{
			Items: wrrItems,
		}

		return &routingPolicyObject
	}

	if !routingPolicy.Geo.IsNull() {
		geoItems := expandDnsRecordSetRoutingPolicyGeoItems(ctx, p, project, routingPolicy.Geo, diags)
		if diags.HasError() {
			return &routingPolicyObject
		}

		routingPolicyObject.Geo = &dns.RRSetRoutingPolicyGeoPolicy{
			Items:         geoItems,
			EnableFencing: routingPolicy.EnableGeoFencing.ValueBool(),
		}

		return &routingPolicyObject
	}

	if !routingPolicy.PrimaryBackup.IsNull() {
		routingPolicyObject.PrimaryBackup = expandDnsRecordSetRoutingPolicyPrimaryBackup(ctx, p, project, routingPolicy.PrimaryBackup, diags)
		return &routingPolicyObject
	}

	return &routingPolicyObject // unreachable here if ps is valid data
}

func expandDnsRecordSetRoutingPolicyWrrItems(ctx context.Context, project, configured types.List, diags *diag.Diagnostics) []*dns.RRSetRoutingPolicyWrrPolicyWrrPolicyItem {
	var policyItems []*dns.RRSetRoutingPolicyWrrPolicyWrrPolicyItem
	var wrrItems []GoogleDnsRecordSetRoutingPolicyWrrModel

	d := configured.ElementsAs(ctx, &wrrItems, false)
	diags.Append(d...)
	if diags.HasError() {
		return policyItems
	}

	for _, r := range wrrItems {
		rrdatas := expandDnsRecordSetRrdata(ctx, r.Rrdatas, diags)
		if diags.HasError() {
			return policyItems
		}

		policyItem := &dns.RRSetRoutingPolicyWrrPolicyWrrPolicyItem{
			Rrdatas: rrdatas,
			Weight:  r.Weight.ValueFloat64(),
		}

		if !r.HealthCheckedTargets.IsNull() {
			policyItem.HealthCheckedTargets = expandDnsRecordSetHealthCheckedTargets(ctx, p, project, r.HealthCheckedTargets, diags)
			if diags.HasError() {
				return policyItems
			}
		}

		policyItems = append(policyItems, policyItem)
	}

	return policyItems
}

func expandDnsRecordSetRoutingPolicyGeoItems(ctx context.Context, p *frameworkProvider, project types.String, configured types.List, diags *diag.Diagnostics) []*dns.RRSetRoutingPolicyGeoPolicyGeoPolicyItem {
	var policyItems []*dns.RRSetRoutingPolicyGeoPolicyGeoPolicyItem
	var geoItems []GoogleDnsRecordSetRoutingPolicyGeoModel

	d := configured.ElementsAs(ctx, &geoItems, false)
	diags.Append(d...)
	if diags.HasError() {
		return policyItems
	}

	for _, r := range geoItems {
		rrdatas := expandDnsRecordSetRrdata(ctx, r.Rrdatas, diags)
		if diags.HasError() {
			return policyItems
		}

		policyItem := &dns.RRSetRoutingPolicyGeoPolicyGeoPolicyItem{
			Rrdatas:  rrdatas,
			Location: r.Location.ValueString(),
		}

		if !r.HealthCheckedTargets.IsNull() {
			policyItem.HealthCheckedTargets = expandDnsRecordSetHealthCheckedTargets(ctx, p, project, r.HealthCheckedTargets, diags)
			if diags.HasError() {
				return policyItems
			}
		}

		policyItems = append(policyItems, policyItem)
	}

	return policyItems
}

func expandDnsRecordSetHealthCheckedTargets(ctx context.Context, p *frameworkProvider, project types.String, configured types.List, diags *diag.Diagnostics) *dns.RRSetRoutingPolicyHealthCheckTargets {
	targetObjects := &dns.RRSetRoutingPolicyHealthCheckTargets{
		InternalLoadBalancers: []*dns.RRSetRoutingPolicyLoadBalancerTarget{},
	}
	var targetsBlock []GoogleDnsRecordSetRoutingPolicyTargetModel

	d := configured.ElementsAs(ctx, &targetsBlock, true)
	diags.Append(d...)
	if diags.HasError() {
		return targetObjects
	}

	targets := targetsBlock[0]

	for _, ilb := range targets.InternalLoadBalancers.Elements() {
		var internalLb GoogleDnsRecordSetRoutingPolicyTargetIlbModel

		d = ilb.(types.Object).As(ctx, &internalLb, unhandledAsEmpty)
		diags.Append(d...)
		if diags.HasError() {
			return targetObjects
		}

		networkUrl := expandDnsRecordSetHealthCheckedTargetsInternalLoadBalancerNetworkUrl(p, project, internalLb.NetworkUrl, diags)
		if diags.HasError() {
			return targetObjects
		}

		ilbObject := &dns.RRSetRoutingPolicyLoadBalancerTarget{
			LoadBalancerType: internalLb.LoadBalancerType.ValueString(),
			IpAddress:        internalLb.IpAddress.ValueString(),
			Port:             internalLb.Port.ValueString(),
			IpProtocol:       internalLb.IpProtocol.ValueString(),
			NetworkUrl:       networkUrl,
			Project:          internalLb.Project.ValueString(),
			Region:           internalLb.Region.ValueString(),
		}

		fmt.Println(fmt.Sprintf("to: %+v", targetObjects))
		fmt.Println(fmt.Sprintf("ILBo: %+v", ilbObject))

		targetObjects.InternalLoadBalancers = append(targetObjects.InternalLoadBalancers, ilbObject)
	}

	return targetObjects
}

func expandDnsRecordSetHealthCheckedTargetsInternalLoadBalancerNetworkUrl(p *frameworkProvider, project types.String, configured types.String, diags *diag.Diagnostics) string {
	if configured.IsNull() || configured.IsUnknown() {
		return ""
	} else if strings.HasPrefix(configured.ValueString(), "https://") {
		return configured.ValueString(), nil
	}
	url := replaceVarsFramework(project, p.ComputeBasePath + configured.ValueString(), diags)
	if diags.HasError() {
		return ""
	}
	return tpgresource.ConvertSelfLinkToV1(url), nil
}

func expandDnsRecordSetRoutingPolicyPrimaryBackup(ctx context.Context, p *frameworkProvider, project types.String, configured types.List, diags *diag.Diagnostics) *dns.RRSetRoutingPolicyPrimaryBackupPolicy {
	var backupObject *dns.RRSetRoutingPolicyPrimaryBackupPolicy
	var primaryBackups []GoogleDnsRecordSetRoutingPolicyPrimaryBackupModel

	d := configured.ElementsAs(ctx, &primaryBackups, true)
	diags.Append(d...)
	if diags.HasError() {
		return backupObject
	}

	primaryBackup := primaryBackups[0]

	backupObject.TrickleTraffic = primaryBackup.TrickleRatio.ValueFloat64()

	backupObject.PrimaryTargets = expandDnsRecordSetHealthCheckedTargets(ctx, p, project, primaryBackup.Primary, diags)
	if diags.HasError() {
		return backupObject
	}

	backupObject.BackupGeoTargets = &dns.RRSetRoutingPolicyGeoPolicy{
		EnableFencing: primaryBackup.EnableGeoFencingForBackups.ValueBool(),
	}

	backupObject.BackupGeoTargets.Items = expandDnsRecordSetRoutingPolicyGeoItems(ctx, primaryBackup.BackupGeo, diags)
	if diags.HasError() {
		return backupObject
	}

	return backupObject
}

// func flattenDnsRecordSetRoutingPolicy(policy *dns.RRSetRoutingPolicy) []interface{} {
// 	if policy == nil {
// 		return []interface{}{}
// 	}
// 	ps := make([]interface{}, 0, 1)
// 	p := make(map[string]interface{})
// 	if policy.Wrr != nil {
// 		p["wrr"] = flattenDnsRecordSetRoutingPolicyWRR(policy.Wrr)
// 	}
// 	if policy.Geo != nil {
// 		p["geo"] = flattenDnsRecordSetRoutingPolicyGEO(policy.Geo)
// 		p["enable_geo_fencing"] = policy.Geo.EnableFencing
// 	}
// 	if policy.PrimaryBackup != nil {
// 		p["primary_backup"] = flattenDnsRecordSetRoutingPolicyPrimaryBackup(policy.PrimaryBackup)
// 	}
// 	return append(ps, p)
// }

// func flattenDnsRecordSetRoutingPolicyWRR(wrr *dns.RRSetRoutingPolicyWrrPolicy) []interface{} {
// 	ris := make([]interface{}, 0, len(wrr.Items))
// 	for _, item := range wrr.Items {
// 		ri := make(map[string]interface{})
// 		ri["weight"] = item.Weight
// 		ri["rrdatas"] = item.Rrdatas
// 		ri["health_checked_targets"] = flattenDnsRecordSetHealthCheckedTargets(item.HealthCheckedTargets)
// 		ris = append(ris, ri)
// 	}
// 	return ris
// }

// func flattenDnsRecordSetRoutingPolicyGEO(geo *dns.RRSetRoutingPolicyGeoPolicy) []interface{} {
// 	ris := make([]interface{}, 0, len(geo.Items))
// 	for _, item := range geo.Items {
// 		ri := make(map[string]interface{})
// 		ri["location"] = item.Location
// 		ri["rrdatas"] = item.Rrdatas
// 		ri["health_checked_targets"] = flattenDnsRecordSetHealthCheckedTargets(item.HealthCheckedTargets)
// 		ris = append(ris, ri)
// 	}
// 	return ris
// }

// func flattenDnsRecordSetHealthCheckedTargets(targets *dns.RRSetRoutingPolicyHealthCheckTargets) []map[string]interface{} {
// 	if targets == nil {
// 		return nil
// 	}

// 	data := map[string]interface{}{
// 		"internal_load_balancers": flattenDnsRecordSetInternalLoadBalancers(targets.InternalLoadBalancers),
// 	}

// 	return []map[string]interface{}{data}
// }

// func flattenDnsRecordSetInternalLoadBalancers(ilbs []*dns.RRSetRoutingPolicyLoadBalancerTarget) []map[string]interface{} {
// 	ilbsSchema := make([]map[string]interface{}, 0, len(ilbs))
// 	for _, ilb := range ilbs {
// 		data := map[string]interface{}{
// 			"load_balancer_type": ilb.LoadBalancerType,
// 			"ip_address":         ilb.IpAddress,
// 			"port":               ilb.Port,
// 			"ip_protocol":        ilb.IpProtocol,
// 			"network_url":        ilb.NetworkUrl,
// 			"project":            ilb.Project,
// 			"region":             ilb.Region,
// 		}
// 		ilbsSchema = append(ilbsSchema, data)
// 	}
// 	return ilbsSchema
// }

// func flattenDnsRecordSetRoutingPolicyPrimaryBackup(primaryBackup *dns.RRSetRoutingPolicyPrimaryBackupPolicy) []map[string]interface{} {
// 	if primaryBackup == nil {
// 		return nil
// 	}

// 	data := map[string]interface{}{
// 		"primary":                        flattenDnsRecordSetHealthCheckedTargets(primaryBackup.PrimaryTargets),
// 		"trickle_ratio":                  primaryBackup.TrickleTraffic,
// 		"backup_geo":                     flattenDnsRecordSetRoutingPolicyGEO(primaryBackup.BackupGeoTargets),
// 		"enable_geo_fencing_for_backups": primaryBackup.BackupGeoTargets.EnableFencing,
// 	}

// 	return []map[string]interface{}{data}
// }

// func rrdatasDnsDiffSuppress(k, old, new string, d *schema.ResourceData) bool {
// 	if k == "rrdatas.#" && (new == "0" || new == "") && old != new {
// 		return false
// 	}

// 	o, n := d.GetChange("rrdatas")
// 	if o == nil || n == nil {
// 		return false
// 	}

// 	oList := convertStringArr(o.([]interface{}))
// 	nList := convertStringArr(n.([]interface{}))

// 	parseFunc := func(record string) string {
// 		switch d.Get("type") {
// 		case "AAAA":
// 			// parse ipv6 to a key from one list
// 			return net.ParseIP(record).String()
// 		case "MX", "DS":
// 			return strings.ToLower(record)
// 		case "TXT":
// 			return strings.ToLower(strings.Trim(record, `"`))
// 		default:
// 			return record
// 		}
// 	}
// 	return rrdatasListDiffSuppress(oList, nList, parseFunc, d)
// }

// // suppress on a list when 1) its items have dups that need to be ignored
// // and 2) string comparison on the items may need a special parse function
// // example of usage can be found ../../../third_party/terraform/tests/resource_dns_record_set_test.go.erb
// func rrdatasListDiffSuppress(oldList, newList []string, fun func(x string) string, _ *schema.ResourceData) bool {
// 	// compare two lists of unordered records
// 	diff := make(map[string]bool, len(oldList))
// 	for _, oldRecord := range oldList {
// 		// set all new IPs to true
// 		diff[fun(oldRecord)] = true
// 	}
// 	for _, newRecord := range newList {
// 		// set matched IPs to false otherwise can't suppress
// 		if diff[fun(newRecord)] {
// 			diff[fun(newRecord)] = false
// 		} else {
// 			return false
// 		}
// 	}
// 	// can't suppress if unmatched records are found
// 	for _, element := range diff {
// 		if element {
// 			return false
// 		}
// 	}
// 	return true
// }
