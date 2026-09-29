package healthcheck

import (
	"testing"

	"github.com/openkcm/checker/internal/config"
)

func TestVerifyChecksContains(t *testing.T) {
	checks := []config.Check{{Type: config.ContainsCheckType, Source: config.ResponseBodySourceType, Value: "ok"}}

	if errs := verifyChecks(checks, []byte("status ok"), nil, nil); len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}

	if errs := verifyChecks(checks, []byte("status bad"), nil, nil); len(errs) != 1 {
		t.Errorf("expected 1 error, got %v", errs)
	}
}

func TestVerifyChecksStatusSource(t *testing.T) {
	checks := []config.Check{{Type: config.ContainsCheckType, Source: config.ResponseStatusSourceType, Value: "200"}}

	errs := verifyChecks(checks, []byte("body"), []byte("200 OK"), nil)
	if len(errs) != 0 {
		t.Errorf("expected no errors matching status, got %v", errs)
	}
}

func TestVerifyChecksRegularExpression(t *testing.T) {
	ok := []config.Check{{Type: config.RegularExpressionCheckType, Value: `^v[0-9]+`}}
	if errs := verifyChecks(ok, []byte("v12"), nil, nil); len(errs) != 0 {
		t.Errorf("expected match, got %v", errs)
	}

	noMatch := []config.Check{{Type: config.RegularExpressionCheckType, Value: `^v[0-9]+`}}
	if errs := verifyChecks(noMatch, []byte("abc"), nil, nil); len(errs) != 1 {
		t.Errorf("expected 1 error, got %v", errs)
	}
}

func TestVerifyChecksRegularExpressionInvalid(t *testing.T) {
	checks := []config.Check{{Type: config.RegularExpressionCheckType, Value: `(unclosed`}}

	errs := verifyChecks(checks, []byte("abc"), nil, nil)
	if len(errs) != 1 || errs[0].Error != "RegularExpression Compile" {
		t.Errorf("expected compile error, got %v", errs)
	}
}

func TestVerifyChecksSuffix(t *testing.T) {
	checks := []config.Check{{Type: config.SuffixCheckType, Value: "end"}}

	if errs := verifyChecks(checks, []byte("the end"), nil, nil); len(errs) != 0 {
		t.Errorf("expected suffix match, got %v", errs)
	}

	if errs := verifyChecks(checks, []byte("end first"), nil, nil); len(errs) != 1 {
		t.Errorf("expected suffix mismatch, got %v", errs)
	}
}

func TestVerifyChecksPrefix(t *testing.T) {
	checks := []config.Check{{Type: config.PrefixCheckType, Value: "start"}}

	if errs := verifyChecks(checks, []byte("start here"), nil, nil); len(errs) != 0 {
		t.Errorf("expected prefix match, got %v", errs)
	}

	if errs := verifyChecks(checks, []byte("no start"), nil, nil); len(errs) != 1 {
		t.Errorf("expected prefix mismatch, got %v", errs)
	}
}

func TestVerifyChecksEqual(t *testing.T) {
	checks := []config.Check{{Type: config.EqualCheckType, Value: "exact"}}

	if errs := verifyChecks(checks, []byte("exact"), nil, nil); len(errs) != 0 {
		t.Errorf("expected equal match, got %v", errs)
	}

	if errs := verifyChecks(checks, []byte("different"), nil, nil); len(errs) != 1 {
		t.Errorf("expected equal mismatch, got %v", errs)
	}
}

func TestVerifyChecksUnknownType(t *testing.T) {
	checks := []config.Check{{Type: config.CheckType("Bogus"), Value: "x"}}

	errs := verifyChecks(checks, []byte("x"), nil, nil)
	if len(errs) != 1 || errs[0].Error != "Unknow Check Type" {
		t.Errorf("expected unknown check type error, got %v", errs)
	}
}

func TestVerifyChecksAppendsToExisting(t *testing.T) {
	existing := []ErrorResponse{{Error: "pre-existing"}}
	checks := []config.Check{{Type: config.ContainsCheckType, Value: "missing"}}

	errs := verifyChecks(checks, []byte("body"), nil, existing)
	if len(errs) != 2 {
		t.Errorf("expected 2 errors (existing + new), got %d", len(errs))
	}
}
