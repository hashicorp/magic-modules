package dns

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-google/google/tpgresource"
	transport_tpg "github.com/hashicorp/terraform-provider-google/google/transport"
)

func DataSourceDnsManagedZone() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDnsManagedZoneRead,

		Schema: map[string]*schema.Schema{
			"dns_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"managed_zone_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"name_servers": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"visibility": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// Google Cloud DNS ManagedZone resources do not have a SelfLink attribute.
			"project": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func dataSourceDnsManagedZoneRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*transport_tpg.Config)
	userAgent, err := tpgresource.GenerateUserAgentString(d, config.UserAgent)
	if err != nil {
		return err
	}

	project, err := tpgresource.GetProject(d, config)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	d.SetId(fmt.Sprintf("projects/%s/managedZones/%s", project, name))

	zone, err := config.NewDnsClient(userAgent).ManagedZones.Get(
		project, name).Do()
	if err != nil {
		return err
	}

	if err := d.Set("name_servers", zone.NameServers); err != nil {
		return fmt.Errorf("Error setting name_servers: %s", err)
	}
	if err := d.Set("name", zone.Name); err != nil {
		return fmt.Errorf("Error setting name: %s", err)
	}
	if err := d.Set("dns_name", zone.DnsName); err != nil {
		return fmt.Errorf("Error setting dns_name: %s", err)
	}
	if err := d.Set("managed_zone_id", zone.Id); err != nil {
		return fmt.Errorf("Error setting managed_zone_id: %s", err)
	}
	if err := d.Set("description", zone.Description); err != nil {
		return fmt.Errorf("Error setting description: %s", err)
	}
	if err := d.Set("visibility", zone.Visibility); err != nil {
		return fmt.Errorf("Error setting visibility: %s", err)
	}
	if err := d.Set("project", project); err != nil {
		return fmt.Errorf("Error setting project: %s", err)
	}

	return nil
}
