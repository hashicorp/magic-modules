package google

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func GetFwTestProvider(t *testing.T) *frameworkTestProvider {
	configsLock.RLock()
	fwProvider, ok := fwProviders[t.Name()]
	configsLock.RUnlock()
	if ok {
		return fwProvider
	}

	var diags *diag.Diagnostics
	p := NewFrameworkTestProvider(t.Name())
	configureApiClient(context.Background(), &p.frameworkProvider, diags)
	if diags.HasError() {
		log.Fatalf("%d errors when configuring test provider client: first is %s", diags.ErrorsCount(), diags.Errors()[0].Detail())
	}

	return p
}

func TestAccFrameworkProviderMeta_setModuleName(t *testing.T) {
	t.Parallel()

	moduleName := "my-module"
	managedZoneName := fmt.Sprintf("tf-test-zone-%s", RandString(t, 10))

	VcrTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: ProtoV5ProviderFactories(t),
		CheckDestroy:             testAccCheckDNSManagedZoneDestroyProducerFramework(t),
		Steps: []resource.TestStep{
			{
				Config: testAccFrameworkProviderMeta_setModuleName(moduleName, managedZoneName, RandString(t, 10)),
			},
		},
	})
}

func TestFrameworkProvider_impl(t *testing.T) {
	var _ provider.ProviderWithMetaSchema = New("test")
}

func TestFrameworkProvider_loadCredentialsFromFile(t *testing.T) {
	cv := CredentialsValidator()

	req := validator.StringRequest{
		ConfigValue: types.StringValue(testFakeCredentialsPath),
	}

	resp := validator.StringResponse{
		Diagnostics: diag.Diagnostics{},
	}

	cv.ValidateString(context.Background(), req, &resp)

	if resp.Diagnostics.WarningsCount() > 0 {
		t.Errorf("Expected 0 warnings, got %d", resp.Diagnostics.WarningsCount())
	}
	if resp.Diagnostics.HasError() {
		t.Errorf("Expected 0 errors, got %d", resp.Diagnostics.ErrorsCount())
	}
}

func TestFrameworkProvider_loadCredentialsFromJSON(t *testing.T) {
	contents, err := ioutil.ReadFile(testFakeCredentialsPath)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	cv := CredentialsValidator()

	req := validator.StringRequest{
		ConfigValue: types.StringValue(string(contents)),
	}

	resp := validator.StringResponse{
		Diagnostics: diag.Diagnostics{},
	}

	cv.ValidateString(context.Background(), req, &resp)
	if resp.Diagnostics.WarningsCount() > 0 {
		t.Errorf("Expected 0 warnings, got %d", resp.Diagnostics.WarningsCount())
	}
	if resp.Diagnostics.HasError() {
		t.Errorf("Expected 0 errors, got %d", resp.Diagnostics.ErrorsCount())
	}
}

func TestAccFrameworkProviderBasePath_setInvalidBasePath(t *testing.T) {
	t.Parallel()

	VcrTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckComputeAddressDestroyProducer(t),
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"google": {
						VersionConstraint: "4.58.0",
						Source:            "hashicorp/google<%= "-" + version unless version == 'ga'  -%>",
					},
				},
				Config:      testAccProviderBasePath_setBasePath("https://www.example.com/compute/beta/", RandString(t, 10)),
				ExpectError: regexp.MustCompile("got HTTP response code 404 with body"),
			},
			{
				ProtoV5ProviderFactories: ProtoV5ProviderFactories(t),
				Config:                   testAccProviderBasePath_setBasePath("https://www.example.com/compute/beta/", RandString(t, 10)),
				ExpectError:              regexp.MustCompile("got HTTP response code 404 with body"),
			},
		},
	})
}

func TestAccFrameworkProviderBasePath_setBasePath(t *testing.T) {
	t.Parallel()

	VcrTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckDNSManagedZoneDestroyProducerFramework(t),
		Steps: []resource.TestStep{
			{
				ExternalProviders: ProviderVersion450(),
				Config:            testAccFrameworkProviderBasePath_setBasePath("https://www.googleapis.com/dns/v1beta2/", RandString(t, 10)),
			},
			{
				ExternalProviders: ProviderVersion450(),
				ResourceName:      "google_dns_managed_zone.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ProtoV5ProviderFactories: ProtoV5ProviderFactories(t),
				Config:                   testAccFrameworkProviderBasePath_setBasePath("https://www.googleapis.com/dns/v1beta2/", RandString(t, 10)),
			},
			{
				ProtoV5ProviderFactories: ProtoV5ProviderFactories(t),
				ResourceName:             "google_dns_managed_zone.foo",
				ImportState:              true,
				ImportStateVerify:        true,
			},
			{
				ProtoV5ProviderFactories: ProtoV5ProviderFactories(t),
				Config:                   testAccFrameworkProviderBasePath_setBasePathstep3("https://www.googleapis.com/dns/v1beta2/", RandString(t, 10)),
			},
		},
	})
}

func testAccFrameworkProviderMeta_setModuleName(key, managedZoneName, recordSetName string) string {
	return fmt.Sprintf(`
terraform {
  provider_meta "google" {
    module_name = "%s"
  }
}

provider "google" {}

resource "google_dns_managed_zone" "zone" {
  name     = "%s-hashicorptest-com"
  dns_name = "%s.hashicorptest.com."
}

resource "google_dns_record_set" "rs" {
  managed_zone = google_dns_managed_zone.zone.name
  name         = "%s.${google_dns_managed_zone.zone.dns_name}"
  type         = "A"
  ttl          = 300
  rrdatas      = [
	"192.168.1.0",
  ]
}

data "google_dns_record_set" "rs" {
  managed_zone = google_dns_record_set.rs.managed_zone
  name         = google_dns_record_set.rs.name
  type         = google_dns_record_set.rs.type
}`, key, managedZoneName, managedZoneName, recordSetName)
}

func testAccFrameworkProviderBasePath_setBasePath(endpoint, name string) string {
	return fmt.Sprintf(`
provider "google" {
  alias               = "dns_custom_endpoint"
  dns_custom_endpoint = "%s"
}

resource "google_dns_managed_zone" "foo" {
	provider    = "google.dns_custom_endpoint"
  name        = "tf-test-zone-%s"
  dns_name    = "tf-test-zone-%s.hashicorptest.com."
  description = "QA DNS zone"
}

data "google_dns_managed_zone" "qa" {
	provider    = "google.dns_custom_endpoint"
  name = google_dns_managed_zone.foo.name
}`, endpoint, name, name)
}

func testAccFrameworkProviderBasePath_setBasePathstep3(endpoint, name string) string {
	return fmt.Sprintf(`
provider "google" {
  alias               = "dns_custom_endpoint"
  dns_custom_endpoint = "%s"
}

resource "google_dns_managed_zone" "foo" {
  provider    = google.dns_custom_endpoint
  name        = "tf-test-zone-%s"
  dns_name    = "tf-test-zone-%s.hashicorptest.com."
  description = "QA DNS zone"
}
`, endpoint, name, name)
}
