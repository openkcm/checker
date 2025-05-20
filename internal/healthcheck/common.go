package healthcheck

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/openkcm/checker/internal/config"
)

func verifyChecks(checks []config.Check, body, status []byte, errors []ErrorResponse) {
	for _, check := range checks {
		sourceValue := body

		switch check.Source {
		case config.ResponseBodySourceType:
			sourceValue = body
		case config.ResponseStatusSourceType:
			sourceValue = status
		}

		switch check.Type {
		case config.ContainsCheckType:
			{
				if !bytes.Contains(sourceValue, []byte(check.Value)) {
					errors = append(errors, ErrorResponse{
						Message: fmt.Sprintf("Response body does not contain given value: %s", check.Value),
						Error:   "Check Contains",
					})

				}
			}
		case config.RegularExpressionCheckType:
			{
				reg, err := regexp.Compile(check.Value)
				if err != nil {
					errors = append(errors, ErrorResponse{
						Message: err.Error(),
						Error:   "RegularExpression Compile",
					})
					return
				}

				if !reg.Match(sourceValue) {
					errors = append(errors, ErrorResponse{
						Message: fmt.Sprintf("Response body does not match: %s", check.Value),
						Error:   "Check RegularExpression",
					})
				}
			}
		case config.SuffixCheckType:
			{
				if !bytes.HasSuffix(sourceValue, []byte(check.Value)) {
					errors = append(errors, ErrorResponse{
						Message: fmt.Sprintf("Response body does not suffix given value: %s", check.Value),
						Error:   "Check Suffix",
					})
				}
			}
		case config.PrefixCheckType:
			{
				if !bytes.HasPrefix(sourceValue, []byte(check.Value)) {
					errors = append(errors, ErrorResponse{
						Message: fmt.Sprintf("Response body does not prefix given value: %s", check.Value),
						Error:   "Check Prefix",
					})
				}
			}
		case config.EqualCheckType:
			{
				if !bytes.Equal(sourceValue, []byte(check.Value)) {
					errors = append(errors, ErrorResponse{
						Message: fmt.Sprintf("Response body is not same as given value: %s", check.Value),
						Error:   "Check Equal",
					})
				}
			}
		default:
			{
				errors = append(errors, ErrorResponse{
					Message: fmt.Sprintf("Unknow check type: %s", check.Type),
					Error:   "Unknow Check Type",
				})
			}
		}
	}
}
