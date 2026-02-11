// Copyright (c) JFrog Ltd. (2025)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apptrust

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/samber/lo"
)

type apptrustError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field"`
}

func (e apptrustError) String() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Field, e.Message, e.Code)
	}
	if e.Code != "" {
		return fmt.Sprintf("%s - %s", e.Code, e.Message)
	}
	return e.Message
}

type AppTrustErrorsResponse struct {
	Errors []apptrustError `json:"errors"`
}

func (r AppTrustErrorsResponse) String() string {
	errs := lo.Reduce(r.Errors, func(err string, item apptrustError, _ int) string {
		if err == "" {
			return item.String()
		} else {
			return fmt.Sprintf("%s, %s", err, item.String())
		}
	}, "")
	return errs
}

// HandleAPIError processes API errors and returns appropriate diagnostics.
// It provides user-friendly error messages based on HTTP status codes.
// It sanitizes error messages to avoid exposing internal implementation details.
// resourceType should be "application", "applications", etc.
func HandleAPIError(response *resty.Response, operation string) diag.Diagnostics {
	return HandleAPIErrorWithType(response, operation, "application")
}

// apiErrorDetail returns the error message from the API. Like JFrog Xray/Artifactory providers:
// we parse errors[].message, message, detail; when the API returns only a generic message
// we return the raw response body so the user sees the exact API output and any field-level details in the JSON.
func apiErrorDetail(response *resty.Response) string {
	body := response.Body()
	msg := extractUserFriendlyError(response)
	if msg == "" {
		if len(body) == 0 {
			return ""
		}
		s := string(body)
		if len(s) > 1000 {
			s = s[:1000] + "... (truncated)"
		}
		return s
	}
	if isGenericValidationMessage(msg) && len(body) > 0 {
		raw := string(body)
		if len(raw) > 2000 {
			raw = raw[:2000] + "... (truncated)"
		}
		return raw
	}
	return msg
}

func isGenericValidationMessage(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "one or more fields failed validation") ||
		strings.Contains(lower, "failed validation") ||
		strings.Contains(lower, "validation failed") ||
		(lower == "invalid request" || strings.TrimSpace(lower) == "invalid request")
}

// HandleAPIErrorWithType processes API errors with a specific resource type.
func HandleAPIErrorWithType(response *resty.Response, operation string, resourceType string) diag.Diagnostics {
	var diags diag.Diagnostics
	statusCode := response.StatusCode()
	errorDetail := apiErrorDetail(response)

	switch statusCode {
	case http.StatusBadRequest:
		if errorDetail != "" {
			diags.AddError("Invalid Request", fmt.Sprintf("Failed to %s %s: %s", operation, resourceType, errorDetail))
		} else {
			diags.AddError("Invalid Request", fmt.Sprintf("Failed to %s %s: The request was invalid (no details from server).", operation, resourceType))
		}
	case http.StatusUnauthorized:
		if errorDetail != "" {
			diags.AddError("Authentication Failed", errorDetail)
		} else {
			diags.AddError("Authentication Failed", "Invalid credentials (no details from server).")
		}
	case http.StatusForbidden:
		if errorDetail != "" {
			diags.AddError("Permission Denied", errorDetail)
		} else {
			diags.AddError("Permission Denied", fmt.Sprintf("You do not have permission to %s %s.", operation, resourceType))
		}
	case http.StatusNotFound:
		if errorDetail != "" {
			diags.AddError("Resource Not Found", errorDetail)
		} else {
			diags.AddError("Resource Not Found", fmt.Sprintf("The %s was not found during %s.", resourceType, operation))
		}
	case http.StatusConflict:
		if errorDetail != "" {
			diags.AddError("Resource Conflict", errorDetail)
		} else {
			diags.AddError("Resource Conflict", fmt.Sprintf("A conflict occurred during %s %s.", operation, resourceType))
		}
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		if errorDetail != "" {
			diags.AddError("Server Error", fmt.Sprintf("Server error (Status: %d): %s", statusCode, errorDetail))
		} else {
			diags.AddError("Server Error", fmt.Sprintf("Server error during %s %s (Status: %d).", operation, resourceType, statusCode))
		}
	default:
		if errorDetail != "" {
			diags.AddError("API Error", fmt.Sprintf("Unexpected error (Status: %d): %s", statusCode, errorDetail))
		} else {
			diags.AddError("API Error", fmt.Sprintf("Unexpected error during %s %s (Status: %d).", operation, resourceType, statusCode))
		}
	}

	return diags
}

// extractUserFriendlyError safely extracts user-friendly error messages from API responses.
func extractUserFriendlyError(response *resty.Response) string {
	body := response.Body()
	if len(body) == 0 {
		return ""
	}

	var errorResponse AppTrustErrorsResponse
	if err := json.Unmarshal(body, &errorResponse); err == nil && len(errorResponse.Errors) > 0 {
		out := errorResponse.String()
		if details := extractDetailsFromBody(body); details != "" {
			out = out + "\n" + details
		}
		return out
	}

	var genericError struct {
		Message string `json:"message"`
		Error   string `json:"error"`
		Detail  string `json:"detail"`
	}
	if err := json.Unmarshal(body, &genericError); err == nil {
		if genericError.Message != "" {
			return genericError.Message
		}
		if genericError.Error != "" {
			return genericError.Error
		}
		if genericError.Detail != "" {
			return genericError.Detail
		}
	}

	var errorArray []struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	if err := json.Unmarshal(body, &errorArray); err == nil && len(errorArray) > 0 {
		var messages []string
		for _, e := range errorArray {
			if e.Message != "" {
				messages = append(messages, e.Message)
			} else if e.Code != "" {
				messages = append(messages, e.Code)
			}
		}
		if len(messages) > 0 {
			return fmt.Sprintf("%s", strings.Join(messages, "; "))
		}
	}

	responseStr := string(body)
	if len(responseStr) > 500 {
		responseStr = responseStr[:500] + "... (truncated)"
	}
	return fmt.Sprintf("API returned: %s", responseStr)
}

func extractDetailsFromBody(body []byte) string {
	var withDetails struct {
		Details          interface{} `json:"details"`
		ValidationErrors interface{} `json:"validation_errors"`
	}
	if err := json.Unmarshal(body, &withDetails); err != nil {
		return ""
	}
	var parts []string
	if withDetails.Details != nil {
		if s, ok := withDetails.Details.(string); ok && s != "" {
			parts = append(parts, "details: "+s)
		} else {
			if b, err := json.Marshal(withDetails.Details); err == nil {
				parts = append(parts, "details: "+string(b))
			}
		}
	}
	if withDetails.ValidationErrors != nil {
		if b, err := json.Marshal(withDetails.ValidationErrors); err == nil {
			parts = append(parts, "validation_errors: "+string(b))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n")
}
