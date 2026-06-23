package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ranjbarhadi/dnsgen/pkg/dnsgen"
	"github.com/spf13/cobra"
)

var (
	wordlen  int
	wordlist string
	fast     bool
	output   string
	verbose  bool
)

func logInfo(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, "[*] "+format+"\n", v...)
}

func logDebug(format string, v ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", v...)
	}
}

func logError(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, "[!] "+format+"\n", v...)
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "dnsgen [INPUT_FILE]",
		Short: "Generate DNS name permutations for domain discovery.",
		Long:  `DNSGen - DNS name permutation generator. Generates variations of domain names for discovery and security testing.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runDNSGen(args[0])
		},
	}

	rootCmd.Flags().IntVarP(&wordlen, "wordlen", "l", 6, "Minimum length of custom words extracted from domains.")
	rootCmd.Flags().StringVarP(&wordlist, "wordlist", "w", "", "Path to custom wordlist file.")
	rootCmd.Flags().BoolVarP(&fast, "fast", "f", false, "Use fast generation mode (fewer permutations).")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "Output file path.")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging.")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runDNSGen(inputPath string) {
	// 1. Read input domains
	var scanner *bufio.Scanner
	if inputPath == "-" {
		scanner = bufio.NewScanner(os.Stdin)
		logDebug("Reading domains from standard input")
	} else {
		file, err := os.Open(inputPath)
		if err != nil {
			logError("Failed to open input file: %v", err)
			os.Exit(1)
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
		logDebug("Reading domains from file: %s", inputPath)
	}

	var inputDomains []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			inputDomains = append(inputDomains, line)
		}
	}
	if err := scanner.Err(); err != nil {
		logError("Error reading input: %v", err)
		os.Exit(1)
	}

	logInfo("Read %d domains from input file", len(inputDomains))

	// 2. Setup words list
	var words []string
	var err error
	if wordlist != "" {
		logDebug("Loading custom wordlist from: %s", wordlist)
		data, err := os.ReadFile(wordlist)
		if err != nil {
			logError("Failed to read custom wordlist: %v", err)
			os.Exit(1)
		}
		words = dnsgen.ParseWordlist(string(data))
	} else {
		logDebug("Loading default embedded wordlist")
		words, err = dnsgen.LoadDefaultWords()
		if err != nil {
			logError("Failed to load default wordlist: %v", err)
			os.Exit(1)
		}
	}

	logInfo("Generator initialized successfully with %d words", len(words))

	// 3. Setup generator & run
	generator := dnsgen.NewGenerator(words)
	generated := generator.Generate(inputDomains, wordlen, fast)

	logInfo("Generated %d unique domain variations", len(generated))

	// 4. Output results
	sort.Strings(generated)
	if output != "" {
		outFile, err := os.Create(output)
		if err != nil {
			logError("Failed to create output file: %v", err)
			os.Exit(1)
		}
		defer outFile.Close()
		writer := bufio.NewWriter(outFile)
		for _, domain := range generated {
			_, _ = writer.WriteString(domain + "\n")
		}
		_ = writer.Flush()
		logInfo("Results written to %s", output)
	} else {
		for _, domain := range generated {
			fmt.Println(domain)
		}
	}
}
