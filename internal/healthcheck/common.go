package healthcheck

import (
	"bytes"
	"regexp"

	"github.com/openkcm/checker/internal/config"
)

func verifyChecks(checks []config.Check, body, status []byte, errors []ErrorResponse) []ErrorResponse {
	errorsAlias := errors
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
					errorsAlias = append(errorsAlias, ErrorResponse{
						Message: "Response body does not contain given value: " + check.Value,
						Error:   "Check Contains",
					})
				}
			}
		case config.RegularExpressionCheckType:
			{
				reg, err := regexp.Compile(check.Value)
				if err != nil {
					errorsAlias = append(errorsAlias, ErrorResponse{
						Message: err.Error(),
						Error:   "RegularExpression Compile",
					})
					return errorsAlias
				}

				if !reg.Match(sourceValue) {
					errorsAlias = append(errorsAlias, ErrorResponse{
						Message: "Response body does not match: " + check.Value,
						Error:   "Check RegularExpression",
					})
				}
			}
		case config.SuffixCheckType:
			{
				if !bytes.HasSuffix(sourceValue, []byte(check.Value)) {
					errorsAlias = append(errorsAlias, ErrorResponse{
						Message: "Response body does not suffix given value: " + check.Value,
						Error:   "Check Suffix",
					})
				}
			}
		case config.PrefixCheckType:
			{
				if !bytes.HasPrefix(sourceValue, []byte(check.Value)) {
					errorsAlias = append(errorsAlias, ErrorResponse{
						Message: "Response body does not prefix given value: " + check.Value,
						Error:   "Check Prefix",
					})
				}
			}
		case config.EqualCheckType:
			{
				if !bytes.Equal(sourceValue, []byte(check.Value)) {
					errorsAlias = append(errorsAlias, ErrorResponse{
						Message: "Response body is not same as given value: " + check.Value,
						Error:   "Check Equal",
					})
				}
			}
		default:
			{
				errorsAlias = append(errorsAlias, ErrorResponse{
					Message: "Unknow check type: " + string(check.Type),
					Error:   "Unknow Check Type",
				})
			}
		}
	}
	return errorsAlias
}
