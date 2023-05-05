package google

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Parse an import id extracting field values using the given list of regexes.
// They are applied in order. The first in the list is tried first.
//
// e.g:
// - projects/(?P<project>[^/]+)/regions/(?P<region>[^/]+)/subnetworks/(?P<name>[^/]+) (applied first)
// - (?P<project>[^/]+)/(?P<region>[^/]+)/(?P<name>[^/]+),
// - (?P<name>[^/]+) (applied last)
func ParseFrameworkImportId(ctx context.Context, idRegexes []string, id string, attributes map[string]attr.Type, diags *diag.Diagnostics) map[string]attr.Value {
	vals := map[string]attr.Value{}
	for _, idFormat := range idRegexes {
		re, err := regexp.Compile(idFormat)

		if err != nil {
			tflog.Debug(ctx, fmt.Sprintf("Could not compile %s.", idFormat))
			diags.AddError(fmt.Sprintf("Could not compile %s.", idFormat), err.Error())
			return vals
		}

		if fieldValues := re.FindStringSubmatch(id); fieldValues != nil {
			log.Printf("[DEBUG] matching ID %s to regex %s.", id, idFormat)
			// Starting at index 1, the first match is the full string.
			for i := 1; i < len(fieldValues); i++ {
				fieldName := re.SubexpNames()[i]
				fieldValue := fieldValues[i]
				log.Printf("[DEBUG] importing %s = %s", fieldName, fieldValue)
				attrType := attributes[fieldName]

				// set the field to the fieldValue with the correct type
				switch attrType {
				case types.StringType:
					vals[fieldName] = types.StringValue(fieldValue)
				case types.Int64Type:
					intVal, err := strconv.Atoi(fieldValue)
					if err != nil {
						diags.AddError(fmt.Sprintf("Error converting %s to int.", fieldValue), err.Error())
						return vals
					}
					vals[fieldName] = types.Int64Value(int64(intVal))
				}
			}

			return vals
		}
	}
	diags.AddError(fmt.Sprintf("Import id %s doesn't match any of the accepted formats.", id), fmt.Sprintf("Accepted formats: %+v", idRegexes))
	return vals
}
