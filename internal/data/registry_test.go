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
