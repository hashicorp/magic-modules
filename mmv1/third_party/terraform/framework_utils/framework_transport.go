package google

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	transport_tpg "github.com/hashicorp/terraform-provider-google/google/transport"

	"google.golang.org/api/googleapi"
)

func sendFrameworkRequest(p *frameworkProvider, method, project, rawurl, userAgent string, body map[string]interface{}, errorRetryPredicates ...transport_tpg.RetryErrorPredicateFunc) (map[string]interface{}, diag.Diagnostics) {
	return sendFrameworkRequestWithTimeout(p, method, project, rawurl, userAgent, body, transport_tpg.DefaultRequestTimeout, errorRetryPredicates...)
}

func sendFrameworkRequestWithTimeout(p *frameworkProvider, method, project, rawurl, userAgent string, body map[string]interface{}, timeout time.Duration, errorRetryPredicates ...transport_tpg.RetryErrorPredicateFunc) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	reqHeaders := make(http.Header)
	reqHeaders.Set("User-Agent", userAgent)
	reqHeaders.Set("Content-Type", "application/json")

	if p.userProjectOverride && project != "" {
		// When project is "NO_BILLING_PROJECT_OVERRIDE" in the function GetCurrentUserEmail,
		// set the header X-Goog-User-Project to be empty string.
		if project == "NO_BILLING_PROJECT_OVERRIDE" {
			reqHeaders.Set("X-Goog-User-Project", "")
		} else {
			// Pass the project into this fn instead of parsing it from the URL because
			// both project names and URLs can have colons in them.
			reqHeaders.Set("X-Goog-User-Project", project)
		}
	}

	if timeout == 0 {
		timeout = time.Hour
	}

	var res *http.Response
	err := transport_tpg.RetryTimeDuration(
		func() error {
			var buf bytes.Buffer
			if body != nil {
				err := json.NewEncoder(&buf).Encode(body)
				if err != nil {
					return err
				}
			}

			u, err := transport_tpg.AddQueryParams(rawurl, map[string]string{"alt": "json"})
			if err != nil {
				return err
			}
			req, err := http.NewRequest(method, u, &buf)
			if err != nil {
				return err
			}

			req.Header = reqHeaders
			res, err = p.client.Do(req)
			if err != nil {
				return err
			}

			if err := googleapi.CheckResponse(res); err != nil {
				googleapi.CloseBody(res)
				return err
			}

			return nil
		},
		timeout,
		errorRetryPredicates...,
	)
	if err != nil {
		diags.AddError("error sending request", err.Error())
		return nil, diags
	}

	if res == nil {
		diags.AddError("Unable to parse server response.", "This is most likely a terraform problem, please file a bug at https://github.com/hashicorp/terraform-provider-google/issues.")
		return nil, diags
	}

	// The defer call must be made outside of the retryFunc otherwise it's closed too soon.
	defer googleapi.CloseBody(res)

	// 204 responses will have no body, so we're going to error with "EOF" if we
	// try to parse it. Instead, we can just return nil.
	if res.StatusCode == 204 {
		return nil, diags
	}
	result := make(map[string]interface{})
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		diags.AddError("error decoding response body", err.Error())
		return nil, diags
	}

	return result, diags
}

func replaceVarsFramework(/* location LocationDescription, */ project types.String, linkTmpl string, diags *diag.Diagnostics) string {
	return replaceVarsRecursiveFramework(/* location,  */ project, linkTmpl, false, 0, diags)
}

// ReplaceVars must be done recursively because there are baseUrls that can contain references to regions
// (eg cloudrun service) there aren't any cases known for 2+ recursion but we will track a run away
// substitution as 10+ calls to allow for future use cases.
func replaceVarsRecursiveFramework(/* location LocationDescription, */ project types.String, linkTmpl string, shorten bool, depth int, diags *diag.Diagnostics) string {
	if depth > 10 {
		diags.AddError("Recursive substitution detected", fmt.Sprintf("depths is %d", depth))
		return ""
	}

	// https://github.com/google/re2/wiki/Syntax
	re := regexp.MustCompile("{{([%[:word:]]+)}}")
	f := buildReplacementFunc(re, /* location,  */ project, linkTmpl, shorten, diags)
	if diags.HasError() {
		return ""
	}
	final := re.ReplaceAllStringFunc(linkTmpl, f)

	if re.Match([]byte(final)) {
		return replaceVarsRecursiveFramework(/* location,  */ project, final, shorten, depth+1)
	}

	return final
}

// This function replaces references to Terraform properties (in the form of {{var}}) with their value in Terraform
// It also replaces {{project}}, {{project_id_or_project}}, {{region}}, and {{zone}} with their appropriate values
// This function supports URL-encoding the result by prepending '%' to the field name e.g. {{%var}}
func buildReplacementFunc(re *regexp.Regexp, /* location LocationDescription, */ project types.String, linkTmpl string, shorten bool) (func(string) string) {
	// var region, zone string

	// This option only seems to be necessary with the validator, I'm not sure the necessity of bringing it over
	// into the framework version or not. (TODO: mbang)

	// var ProjectID
	// if strings.Contains(linkTmpl, "{{project_id_or_project}}") {
	// 	v, ok := d.GetOkExists("project_id")
	// 	if ok {
	// 		projectID, _ = v.(string)
	// 	}
	// 	if projectID == "" {
	// 		project, err = getProject(d, config)
	// 	}
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	// if strings.Contains(linkTmpl, "{{region}}") {
	// 	region, err = location.getRegion()
	// 	if err != nil {
	// 		diags.AddError("Error getting region", err.Error())
	// 		return nil
	// 	}
	// }

	// if strings.Contains(linkTmpl, "{{zone}}") {
	// 	zone, err = location.getZone()
	// 	if err != nil {
	// 		diags.AddError("Error getting zone", err.Error())
	// 		return nil
	// 	}
	// }

	// if strings.Contains(linkTmpl, "{{location}}") {
	// 	loc, err = location.getLocation()
	// 	if err != nil {
	// 		diags.AddError("Error getting location", err.Error())
	// 		return nil
	// 	}
	// }

	f := func(s string) string {

		m := re.FindStringSubmatch(s)[1]
		if m == "project" {
			return project.ValueString()
		}
		// if m == "project_id_or_project" {
		// 	if projectID != "" {
		// 		return projectID
		// 	}
		// 	return project
		// }
		// if m == "region" {
		// 	return region
		// }
		// if m == "zone" {
		// 	return zone
		// }
		// if m == "location" {
		// 	return loc
		// }
		// if string(m[0]) == "%" {
		// 	v, ok := d.GetOkExists(m[1:])
		// 	if ok {
		// 		return url.PathEscape(fmt.Sprintf("%v", v))
		// 	}
		// } else {
		// 	v, ok := d.GetOkExists(m)
		// 	if ok {
		// 		if shorten {
		// 			return GetResourceNameFromSelfLink(fmt.Sprintf("%v", v))
		// 		} else {
		// 			return fmt.Sprintf("%v", v)
		// 		}
		// 	}
		// }

		return ""
	}

	return f
}