package dnsgen

import (
	"embed"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/publicsuffix"
)

//go:embed words.txt
var defaultWordsEmbed embed.FS

// DomainParts represents split domains: subdomains followed by the registered domain.
type DomainParts []string

// PermutatorFunc is the signature for permutation generators.
type PermutatorFunc func(parts DomainParts) []string

// Generator handles wordlists and domain name permutations.
type Generator struct {
	Words           []string
	NumCount        int
	Permutators     []PermutatorFunc
	FastPermutators []PermutatorFunc
}

// LoadDefaultWords reads and parses the default embedded words.txt.
func LoadDefaultWords() ([]string, error) {
	data, err := defaultWordsEmbed.ReadFile("words.txt")
	if err != nil {
		return nil, err
	}
	return ParseWordlist(string(data)), nil
}

// ParseWordlist filters out comments and empty lines from a wordlist string.
func ParseWordlist(content string) []string {
	var words []string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			words = append(words, line)
		}
	}
	return words
}

// NewGenerator creates a new Generator instance.
func NewGenerator(words []string) *Generator {
	g := &Generator{
		Words:    words,
		NumCount: 3,
	}

	g.FastPermutators = []PermutatorFunc{
		g.modifyNumbers,
		g.commonPorts,
	}

	g.Permutators = []PermutatorFunc{
		g.insertWordEveryIndex,
		g.modifyNumbers,
		g.environmentPrefix,
		g.cloudProviderAdditions,
		g.regionPrefixes,
		g.microservicePatterns,
		g.internalTooling,
		g.commonPorts,
	}

	return g
}

// PartiateDomain splits a domain into subdomain parts and the registered domain.
func (g *Generator) PartiateDomain(domain string) DomainParts {
	domain = strings.ToLower(domain)
	tldPlusOne, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return strings.Split(domain, ".")
	}

	if domain == tldPlusOne {
		return DomainParts{"", tldPlusOne}
	}

	suffix := "." + tldPlusOne
	if strings.HasSuffix(domain, suffix) {
		subdomain := domain[:len(domain)-len(suffix)]
		parts := strings.Split(subdomain, ".")
		return append(parts, tldPlusOne)
	}

	return DomainParts{"", tldPlusOne}
}

// ExtractCustomWords extracts custom words from domain names based on naming conventions.
func (g *Generator) ExtractCustomWords(domains []string, wordlen int) map[string]struct{} {
	validTokens := make(map[string]struct{})

	for _, domain := range domains {
		parts := g.PartiateDomain(domain)
		if len(parts) <= 1 {
			continue
		}
		// All parts except the registered domain
		partition := parts[:len(parts)-1]

		for _, part := range partition {
			partLower := strings.ToLower(part)
			// Split by hyphen
			subparts := strings.Split(partLower, "-")
			for _, sp := range subparts {
				if len(sp) >= wordlen {
					validTokens[sp] = struct{}{}
				}
			}
			if len(partLower) >= wordlen {
				validTokens[partLower] = struct{}{}
			}
		}
	}

	return validTokens
}

// Generate yields all domain permutations.
func (g *Generator) Generate(domains []string, wordlen int, fastMode bool) []string {
	var activePermutators []PermutatorFunc
	if fastMode {
		activePermutators = g.FastPermutators
	} else {
		activePermutators = g.Permutators
	}

	// Extract and temporarily inject custom words if wordlen > 0
	customWords := g.ExtractCustomWords(domains, wordlen)
	originalWords := g.Words
	if len(customWords) > 0 {
		combined := make([]string, 0, len(g.Words)+len(customWords))
		combined = append(combined, g.Words...)
		for cw := range customWords {
			combined = append(combined, cw)
		}
		g.Words = combined
	}
	defer func() {
		g.Words = originalWords
	}()

	// Use map for unique results
	uniqueDomains := make(map[string]struct{})

	// Deduplicate input domains to prevent redundant runs
	inputMap := make(map[string]struct{})
	for _, domain := range domains {
		trimmed := strings.TrimSpace(strings.ToLower(domain))
		if trimmed != "" {
			inputMap[trimmed] = struct{}{}
		}
	}

	for domain := range inputMap {
		parts := g.PartiateDomain(domain)
		for _, permutator := range activePermutators {
			variations := permutator(parts)
			for _, v := range variations {
				uniqueDomains[v] = struct{}{}
			}
		}
	}

	// Convert back to slice
	results := make([]string, 0, len(uniqueDomains))
	for d := range uniqueDomains {
		results = append(results, d)
	}

	return results
}

// 1. insertWordEveryIndex: Insert words from wordlist between subdomain levels.
func (g *Generator) insertWordEveryIndex(parts DomainParts) []string {
	var domains []string
	if len(parts) == 0 {
		return domains
	}

	regDomain := parts[len(parts)-1]
	subdomains := parts[:len(parts)-1]

	for _, w := range g.Words {
		for i := 0; i <= len(subdomains); i++ {
			tmp := make([]string, 0, len(subdomains)+1)
			tmp = append(tmp, subdomains[:i]...)
			tmp = append(tmp, w)
			tmp = append(tmp, subdomains[i:]...)

			domains = append(domains, strings.Join(tmp, ".")+"."+regDomain)
		}
	}
	return domains
}

var digitRegex = regexp.MustCompile(`\d{1,3}`)

// 2. modifyNumbers: Increase and decrease numbers found in subdomain parts.
func (g *Generator) modifyNumbers(parts DomainParts) []string {
	var domains []string
	if len(parts) < 2 {
		return domains
	}

	partsJoined := strings.Join(parts[:len(parts)-1], ".")
	digits := digitRegex.FindAllString(partsJoined, -1)

	for _, d := range digits {
		val, _ := strconv.Atoi(d)
		dLen := len(d)

		// Increase numbers
		for m := 0; m < g.NumCount; m++ {
			newVal := val + 1 + m
			replacement := fmt.Sprintf("%0*d", dLen, newVal)
			tmpDomain := strings.ReplaceAll(partsJoined, d, replacement)
			domains = append(domains, tmpDomain+"."+parts[len(parts)-1])
		}

		// Decrease numbers
		for m := 0; m < g.NumCount; m++ {
			newVal := val - 1 - m
			if newVal >= 0 {
				replacement := fmt.Sprintf("%0*d", dLen, newVal)
				tmpDomain := strings.ReplaceAll(partsJoined, d, replacement)
				domains = append(domains, tmpDomain+"."+parts[len(parts)-1])
			}
		}
	}

	return domains
}

// 3. environmentPrefix: Add common environment prefixes.
func (g *Generator) environmentPrefix(parts DomainParts) []string {
	var domains []string
	if len(parts) == 0 {
		return domains
	}

	environments := []string{"dev", "staging", "uat", "prod", "test"}
	regDomain := parts[len(parts)-1]
	subdomains := parts[:len(parts)-1]

	for _, env := range environments {
		tmp := make([]string, 0, len(subdomains)+1)
		tmp = append(tmp, env)
		tmp = append(tmp, subdomains...)
		domains = append(domains, strings.Join(tmp, ".")+"."+regDomain)
	}
	return domains
}

// 4. cloudProviderAdditions: Add common cloud provider related subdomains.
func (g *Generator) cloudProviderAdditions(parts DomainParts) []string {
	var domains []string
	if len(parts) == 0 {
		return domains
	}

	cloudTerms := []string{"aws", "azure", "gcp", "k8s", "cloud"}
	serviceTerms := []string{"api", "cdn", "storage", "auth", "db"}
	regDomain := parts[len(parts)-1]
	subdomains := parts[:len(parts)-1]

	for _, term := range cloudTerms {
		for _, service := range serviceTerms {
			tmp := make([]string, 0, len(subdomains)+1)
			tmp = append(tmp, fmt.Sprintf("%s-%s", service, term))
			tmp = append(tmp, subdomains...)
			domains = append(domains, strings.Join(tmp, ".")+"."+regDomain)
		}
	}
	return domains
}

// 5. regionPrefixes: Add common region/location prefixes.
func (g *Generator) regionPrefixes(parts DomainParts) []string {
	var domains []string
	if len(parts) == 0 {
		return domains
	}

	regions := []string{"us-east", "us-west", "eu-west", "eu-central", "ap-south", "ap-northeast", "sa-east", "af-south"}
	regDomain := parts[len(parts)-1]
	subdomains := parts[:len(parts)-1]

	for _, region := range regions {
		tmp := make([]string, 0, len(subdomains)+1)
		tmp = append(tmp, region)
		tmp = append(tmp, subdomains...)
		domains = append(domains, strings.Join(tmp, ".")+"."+regDomain)
	}
	return domains
}

// 6. microservicePatterns: Add common microservice naming patterns.
func (g *Generator) microservicePatterns(parts DomainParts) []string {
	var domains []string
	if len(parts) == 0 {
		return domains
	}

	services := []string{"auth", "user", "payment", "notification", "order", "inventory"}
	suffixes := []string{"service", "svc", "api", "app"}
	regDomain := parts[len(parts)-1]
	subdomains := parts[:len(parts)-1]

	for _, service := range services {
		for _, suffix := range suffixes {
			tmp := make([]string, 0, len(subdomains)+1)
			tmp = append(tmp, fmt.Sprintf("%s-%s", service, suffix))
			tmp = append(tmp, subdomains...)
			domains = append(domains, strings.Join(tmp, ".")+"."+regDomain)
		}
	}
	return domains
}

// 7. internalTooling: Add common internal tool and platform subdomains.
func (g *Generator) internalTooling(parts DomainParts) []string {
	var domains []string
	if len(parts) == 0 {
		return domains
	}

	tools := []string{"jenkins", "gitlab", "grafana", "kibana", "prometheus", "monitoring", "jira"}
	prefixes := []string{"internal", "tools", "admin"}
	regDomain := parts[len(parts)-1]
	subdomains := parts[:len(parts)-1]

	for _, tool := range tools {
		for _, prefix := range prefixes {
			// prefix, tool
			tmp1 := make([]string, 0, len(subdomains)+2)
			tmp1 = append(tmp1, subdomains...)
			tmp1 = append(tmp1, prefix, tool)
			domains = append(domains, strings.Join(tmp1, ".")+"."+regDomain)

			// tool, prefix
			tmp2 := make([]string, 0, len(subdomains)+2)
			tmp2 = append(tmp2, subdomains...)
			tmp2 = append(tmp2, tool, prefix)
			domains = append(domains, strings.Join(tmp2, ".")+"."+regDomain)
		}
	}
	return domains
}

// 8. commonPorts: Add common port numbers as prefixes.
func (g *Generator) commonPorts(parts DomainParts) []string {
	var domains []string
	if len(parts) == 0 {
		return domains
	}

	ports := []string{"8080", "8443", "3000", "5000", "9000", "8888"}
	regDomain := parts[len(parts)-1]
	subdomains := parts[:len(parts)-1]

	for _, port := range ports {
		// port directly
		tmp1 := make([]string, 0, len(subdomains)+1)
		tmp1 = append(tmp1, port)
		tmp1 = append(tmp1, subdomains...)
		domains = append(domains, strings.Join(tmp1, ".")+"."+regDomain)

		// port with 'port-' prefix
		tmp2 := make([]string, 0, len(subdomains)+1)
		tmp2 = append(tmp2, "port-"+port)
		tmp2 = append(tmp2, subdomains...)
		domains = append(domains, strings.Join(tmp2, ".")+"."+regDomain)
	}
	return domains
}
