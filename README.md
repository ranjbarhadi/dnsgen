# dnsgen

A fast, lightweight, and modern Go port of [dnsgen](https://github.com/AlephNullSK/dnsgen) by [AlephNullSK](https://github.com/AlephNullSK).

`dnsgen` generates domain name permutations from a list of input domains and an optional wordlist. It is designed to assist in security assessments, bug bounty hunting, and subdomain discovery.

## 🚀 Key Features

- **Original Logic Parity:** Implements all 8 core permutation generators from the Python version.
- **Fast and Lightweight:** Built in Go for high execution speed and low resource consumption.
- **Zero External Data Dependencies:** The default wordlist is embedded directly into the binary using Go's `//go:embed`.
- **Bug Fixes:** Resolves an issue in the original Python version where custom words extracted from input domains with length constraints (`-l` / `--wordlen`) were ignored during generation.
- **Pipeline-Friendly:** Correctly writes logs to `stderr` and clean domain outputs to `stdout`, facilitating easy integration with commands like `massdns` or `nmap`.

## 📦 Installation

To install `dnsgen`, ensure you have Go installed, then run:

```bash
GOPROXY=direct go install github.com/ranjbarhadi/dnsgen@latest
```

_(Or build it locally from the repository root):_

```bash
go build -o dnsgen main.go
```

## 🛠 Usage

You can feed domains to `dnsgen` via standard input (stdin) or by passing a file path.

```bash
# Using stdin
cat domains.txt | ./dnsgen -

# Using input file
./dnsgen domains.txt
```

### CLI Options

```text
Usage:
  dnsgen [flags] [INPUT_FILE]

Flags:
  -f, --fast             Fast generation mode (skips insertion permutators to generate a smaller subset)
  -h, --help             help for dnsgen
  -l, --wordlen int      Minimum length of custom words extracted from domains (default 6)
  -o, --output string    Output file path (defaults to stdout)
  -v, --verbose          Enable verbose logging (outputs stats to stderr)
  -w, --wordlist string  Path to custom wordlist file (uses embedded words.txt by default)
```

## 🧠 Permutation Engines Included

`dnsgen` implements all default generators:

1. **Insert Word Every Index:** Inserts words from the wordlist at every position between subdomain parts.
2. **Modify Numbers:** Increments and decrements numbers found in subdomains (preserving leading zero formats, e.g., `01` -> `02`).
3. **Environment Prefix:** Adds common environment-related subdomains (`dev`, `staging`, `uat`, `prod`, `test`).
4. **Cloud Provider Additions:** Prepends cloud platform combinations (e.g. `service-cloud`).
5. **Region Prefixes:** Prepends regional formats (e.g. `us-east`, `eu-west`).
6. **Microservice Patterns:** Appends standard microservice identifiers (`-service`, `-api`).
7. **Internal Tooling:** Prepends or appends internal platform indicators (e.g. `jenkins`, `gitlab`).
8. **Common Ports:** Prepends port names or numbers (e.g. `8080`, `port-8080`).

## 🤝 Acknowledgments

This tool is a Go port of the excellent Python library [dnsgen](https://github.com/AlephNullSK/dnsgen) written by [AlephNullSK](https://github.com/AlephNullSK). All core permutation logic and default wordlists originate from their work.
