package dnsgen

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func getSampleDomains() []string {
	return []string{
		"api.example.com",
		"dev-api01.test.com",
		"staging-auth.prod.company.com",
		"v2.api.service.org",
	}
}

func createTestWordlist(t *testing.T) (string, []string) {
	content := `
# Environment
dev
staging
prod

# Services
api
auth
service

# Version
v1
v2
v3
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_wordlist.txt")
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test wordlist: %v", err)
	}

	words := []string{"dev", "staging", "prod", "api", "auth", "service", "v1", "v2", "v3"}
	return path, words
}

func TestPartiateDomain(t *testing.T) {
	g := NewGenerator(nil)

	testCases := []struct {
		domain   string
		expected []string
	}{
		{"api.example.com", []string{"api", "example.com"}},
		{"dev.api.example.com", []string{"dev", "api", "example.com"}},
		{"test.sub.domain.example.co.uk", []string{"test", "sub", "domain", "example.co.uk"}},
		{"example.com", []string{"", "example.com"}},
	}

	for _, tc := range testCases {
		got := g.PartiateDomain(tc.domain)
		if !reflect.DeepEqual([]string(got), tc.expected) {
			t.Errorf("PartiateDomain(%q) = %v; want %v", tc.domain, got, tc.expected)
		}
	}
}

func TestExtractCustomWords(t *testing.T) {
	g := NewGenerator(nil)
	domains := []string{
		"development-api.example.com",
		"staging-auth.test.com",
		"prod-service.company.com",
	}

	words6 := g.ExtractCustomWords(domains, 6)
	if _, ok := words6["development"]; !ok {
		t.Error("expected 'development' in extracted words for len 6")
	}

	words4 := g.ExtractCustomWords(domains, 4)
	if _, ok := words4["staging"]; !ok {
		t.Error("expected 'staging' in extracted words for len 4")
	}
	if _, ok := words4["api"]; ok {
		t.Error("did not expect 'api' (len 3) in extracted words for len 4")
	}
}

func TestWordInsertionPermutator(t *testing.T) {
	_, words := createTestWordlist(t)
	g := NewGenerator(words)

	domain := "api.example.com"
	variations := g.Generate([]string{domain}, 0, false)

	// Convert slice to map for fast checks
	varMap := make(map[string]struct{})
	for _, v := range variations {
		varMap[v] = struct{}{}
	}

	if _, ok := varMap["dev.api.example.com"]; !ok {
		t.Error("expected 'dev.api.example.com' in generated variations")
	}
	if _, ok := varMap["api.staging.example.com"]; !ok {
		t.Error("expected 'api.staging.example.com' in generated variations")
	}
}

func TestNumberManipulationPermutator(t *testing.T) {
	g := NewGenerator(nil)

	domain := "api2.example.com"
	variations := g.Generate([]string{domain}, 0, false)

	varMap := make(map[string]struct{})
	for _, v := range variations {
		varMap[v] = struct{}{}
	}

	if _, ok := varMap["api1.example.com"]; !ok {
		t.Error("expected 'api1.example.com' in generated variations")
	}
	if _, ok := varMap["api3.example.com"]; !ok {
		t.Error("expected 'api3.example.com' in generated variations")
	}
}

func TestRegionPrefixPermutator(t *testing.T) {
	g := NewGenerator(nil)

	domain := "api.example.com"
	variations := g.Generate([]string{domain}, 0, false)

	varMap := make(map[string]struct{})
	for _, v := range variations {
		varMap[v] = struct{}{}
	}

	if _, ok := varMap["us-east.api.example.com"]; !ok {
		t.Error("expected 'us-east.api.example.com' in generated variations")
	}
	if _, ok := varMap["eu-west.api.example.com"]; !ok {
		t.Error("expected 'eu-west.api.example.com' in generated variations")
	}
}

func TestFastModeGeneration(t *testing.T) {
	_, words := createTestWordlist(t)
	g := NewGenerator(words)

	domain := "api.example.com"

	normalVariations := g.Generate([]string{domain}, 0, false)
	fastVariations := g.Generate([]string{domain}, 0, true)

	if len(fastVariations) >= len(normalVariations) {
		t.Errorf("fast mode generated %d variants, normal mode generated %d. Fast mode should generate fewer.", len(fastVariations), len(normalVariations))
	}
}

func TestEmptyDomainHandling(t *testing.T) {
	g := NewGenerator([]string{"test"})
	variations := g.Generate([]string{}, 0, false)
	if len(variations) != 0 {
		t.Errorf("expected 0 variations for empty input, got %d", len(variations))
	}
}

func TestSpecialCharacterDomains(t *testing.T) {
	g := NewGenerator([]string{"dev"})
	domain := "api-test_01.example.com"
	variations := g.Generate([]string{domain}, 0, false)
	if len(variations) == 0 {
		t.Error("expected variations to be generated for special character domain")
	}
}
