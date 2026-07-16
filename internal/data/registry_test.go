package data

import (
	"reflect"
	"testing"
)

// expectedRuleHeaders is the CSV rule-column order that previously exported CSVs
// depend on. It must never change without a migration, so this test pins it.
var expectedRuleHeaders = []string{
	"RulesCreation",
	"RulesUpdate",
	"RulesDeletion",
	"RulesRequiredLinearHistory",
	"RulesMergeQueue",
	"RulesRequiredDeployments",
	"RulesRequiredSignatures",
	"RulesPullRequest",
	"RulesRequiredStatusChecks",
	"RulesNonFastForward",
	"RulesCommitMessagePattern",
	"RulesCommitAuthorEmailPattern",
	"RulesCommitterEmailPattern",
	"RulesBranchNamePattern",
	"RulesTagNamePattern",
	"RulesFilePathRestriction",
	"RulesFilePathLength",
	"RulesFileExtensionRestriction",
	"RulesMaxFileSize",
	"RulesWorkflows",
	"RulesCodeScanning",
	"RulesCodeQuality",
	"RulesCopilotCodeReview",
	"RulesLicenseComplianceScanning",
	"RulesCodeCoverage",
}

func TestRuleHeadersOrder(t *testing.T) {
	got := RuleHeaders()
	if !reflect.DeepEqual(got, expectedRuleHeaders) {
		t.Errorf("RuleHeaders() = %v, want %v", got, expectedRuleHeaders)
	}
}

func TestRuleRegistryUniqueTypesAndHeaders(t *testing.T) {
	seenTypes := make(map[string]struct{}, len(RuleRegistry))
	seenHeaders := make(map[string]struct{}, len(RuleRegistry))
	for _, spec := range RuleRegistry {
		if spec.Type == "" {
			t.Errorf("rule spec with header %q has empty Type", spec.CSVHeader)
		}
		if spec.CSVHeader == "" {
			t.Errorf("rule spec with type %q has empty CSVHeader", spec.Type)
		}
		if _, ok := seenTypes[spec.Type]; ok {
			t.Errorf("duplicate rule type in registry: %q", spec.Type)
		}
		if _, ok := seenHeaders[spec.CSVHeader]; ok {
			t.Errorf("duplicate CSV header in registry: %q", spec.CSVHeader)
		}
		seenTypes[spec.Type] = struct{}{}
		seenHeaders[spec.CSVHeader] = struct{}{}
	}
}

func TestHeaderMapDerivedFromRegistry(t *testing.T) {
	if len(HeaderMap) != len(RuleRegistry) {
		t.Fatalf("HeaderMap has %d entries, want %d", len(HeaderMap), len(RuleRegistry))
	}
	for _, spec := range RuleRegistry {
		if got := HeaderMap[spec.CSVHeader]; got != spec.Type {
			t.Errorf("HeaderMap[%q] = %q, want %q", spec.CSVHeader, got, spec.Type)
		}
	}
}

func TestNonOmitEmptyFieldsDerivedFromRegistry(t *testing.T) {
	for _, spec := range RuleRegistry {
		got, ok := NonOmitEmptyFields[spec.Type]
		if len(spec.NonOmitEmpty) == 0 {
			if ok {
				t.Errorf("NonOmitEmptyFields should not contain %q", spec.Type)
			}
			continue
		}
		if !reflect.DeepEqual(got, spec.NonOmitEmpty) {
			t.Errorf("NonOmitEmptyFields[%q] = %v, want %v", spec.Type, got, spec.NonOmitEmpty)
		}
	}
}

func TestRuleSpecByType(t *testing.T) {
	t.Run("known type returns matching spec", func(t *testing.T) {
		spec := RuleSpecByType("pull_request")
		if spec == nil {
			t.Fatal("RuleSpecByType(\"pull_request\") = nil, want spec")
			return
		}
		if spec.Type != "pull_request" || spec.CSVHeader != "RulesPullRequest" {
			t.Errorf("RuleSpecByType(\"pull_request\") = %+v, want type pull_request / header RulesPullRequest", spec)
		}
	})

	t.Run("unknown type returns nil", func(t *testing.T) {
		if spec := RuleSpecByType("does_not_exist"); spec != nil {
			t.Errorf("RuleSpecByType(\"does_not_exist\") = %+v, want nil", spec)
		}
	})
}

func TestRuleSpecByHeader(t *testing.T) {
	t.Run("known header returns matching spec", func(t *testing.T) {
		spec := RuleSpecByHeader("RulesMergeQueue")
		if spec == nil {
			t.Fatal("RuleSpecByHeader(\"RulesMergeQueue\") = nil, want spec")
			return
		}
		if spec.Type != "merge_queue" || spec.CSVHeader != "RulesMergeQueue" {
			t.Errorf("RuleSpecByHeader(\"RulesMergeQueue\") = %+v, want type merge_queue", spec)
		}
	})

	t.Run("unknown header returns nil", func(t *testing.T) {
		if spec := RuleSpecByHeader("RulesNope"); spec != nil {
			t.Errorf("RuleSpecByHeader(\"RulesNope\") = %+v, want nil", spec)
		}
	})
}

func TestValidFields(t *testing.T) {
	t.Run("rule with fields returns them", func(t *testing.T) {
		fields := ValidFields("merge_queue")
		if fields == nil {
			t.Fatal("ValidFields(\"merge_queue\") = nil, want fields")
		}
		if _, ok := fields["MergeMethod"]; !ok {
			t.Errorf("ValidFields(\"merge_queue\") missing MergeMethod, got %v", fields)
		}
	})

	t.Run("rule without fields returns nil", func(t *testing.T) {
		if fields := ValidFields("creation"); fields != nil {
			t.Errorf("ValidFields(\"creation\") = %v, want nil", fields)
		}
	})

	t.Run("unknown rule returns nil", func(t *testing.T) {
		if fields := ValidFields("does_not_exist"); fields != nil {
			t.Errorf("ValidFields(\"does_not_exist\") = %v, want nil", fields)
		}
	})
}
