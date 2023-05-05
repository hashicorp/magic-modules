package google

import (
	"time"

	"google.golang.org/api/dns/v1"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// We'll continue to use the SDK here until more guidance is given with respect to
// how to replace resource.StateChangeConf.WaitForState with the plugin-framework
// See https://github.com/hashicorp/terraform-plugin-framework/issues/513 for details

type DnsChangeWaiter struct {
	Service     *dns.Service
	Change      *dns.Change
	Project     string
	ManagedZone string
}

func (w *DnsChangeWaiter) RefreshFunc() resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		var chg *dns.Change
		var err error

		chg, err = w.Service.Changes.Get(
			w.Project, w.ManagedZone, w.Change.Id).Do()

		if err != nil {
			return nil, "", err
		}

		return chg, chg.Status, nil
	}
}

func (w *DnsChangeWaiter) Conf() *resource.StateChangeConf {
	return &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     []string{"done"},
		Refresh:    w.RefreshFunc(),
		Timeout:    10 * time.Minute,
		MinTimeout: 2 * time.Second,
	}
}
