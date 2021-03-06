package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"

	"github.com/chavacava/gusano/formatter"

	"github.com/BurntSushi/toml"
	"github.com/chavacava/gusano/lint"
	"github.com/chavacava/gusano/rule"
	"golang.org/x/tools/go/packages"
	gopack "golang.org/x/tools/go/packages"
)

func fail(err string) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

var defaultRules = []lint.Rule{
	&rule.UnusedSymbolRule{},
}

var allRules = defaultRules

var allFormatters = []lint.Formatter{
	&formatter.Stylish{},
	&formatter.Friendly{},
	&formatter.JSON{},
	&formatter.NDJSON{},
	&formatter.Default{},
	&formatter.Unix{},
	&formatter.Checkstyle{},
	&formatter.Plain{},
}

func getFormatters() map[string]lint.Formatter {
	result := map[string]lint.Formatter{}
	for _, f := range allFormatters {
		result[f.Name()] = f
	}
	return result
}

func getLintingRules(config *lint.Config) []lint.Rule {
	rulesMap := map[string]lint.Rule{}
	for _, r := range allRules {
		rulesMap[r.Name()] = r
	}

	lintingRules := []lint.Rule{}
	for name := range config.Rules {
		rule, ok := rulesMap[name]
		if !ok {
			fail("cannot find rule: " + name)
		}
		lintingRules = append(lintingRules, rule)
	}

	return lintingRules
}

func parseConfig(path string) *lint.Config {
	config := &lint.Config{}
	file, err := ioutil.ReadFile(path)
	if err != nil {
		fail("cannot read the config file")
	}
	_, err = toml.Decode(string(file), config)
	if err != nil {
		fail("cannot parse the config file: " + err.Error())
	}
	return config
}

func normalizeConfig(config *lint.Config) {
	if config.Confidence == 0 {
		config.Confidence = 0.8
	}
	severity := config.Severity
	if severity != "" {
		for k, v := range config.Rules {
			if v.Severity == "" {
				v.Severity = severity
			}
			config.Rules[k] = v
		}
		for k, v := range config.Directives {
			if v.Severity == "" {
				v.Severity = severity
			}
			config.Directives[k] = v
		}
	}
}

func getConfig() *lint.Config {
	config := defaultConfig()
	if configPath != "" {
		config = parseConfig(configPath)
	}
	normalizeConfig(config)
	return config
}

func getFormatter() lint.Formatter {
	formatters := getFormatters()
	formatter := formatters["default"]
	if formatterName != "" {
		f, ok := formatters[formatterName]
		if !ok {
			fail("unknown formatter " + formatterName)
		}
		formatter = f
	}
	return formatter
}

func buildDefaultConfigPath() string {
	var result string
	if homeDir, err := homedir.Dir(); err == nil {
		result = filepath.Join(homeDir, "gusano.toml")
		if _, err := os.Stat(result); err != nil {
			result = ""
		}
	}

	return result
}

func defaultConfig() *lint.Config {
	defaultConfig := lint.Config{
		Confidence: 0.0,
		Severity:   lint.SeverityWarning,
		Rules:      map[string]lint.RuleConfig{},
	}
	for _, r := range defaultRules {
		defaultConfig.Rules[r.Name()] = lint.RuleConfig{}
	}
	return &defaultConfig
}

func normalizeSplit(strs []string) []string {
	res := []string{}
	for _, s := range strs {
		t := strings.Trim(s, " \t")
		if len(t) > 0 {
			res = append(res, t)
		}
	}
	return res
}

func getPackages() []*packages.Package {
	globs := normalizeSplit(flag.Args())
	if len(globs) == 0 {
		globs = append(globs, ".")
	}

	cfg := &gopack.Config{Mode: gopack.LoadSyntax}
	packages, err := gopack.Load(cfg, globs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load: %v\n", err)
		os.Exit(1)
	}
	if gopack.PrintErrors(packages) > 0 {
		os.Exit(1)
	}

	return packages
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join([]string(*i), " ")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var configPath string
var excludePaths arrayFlags
var formatterName string

var originalUsage = flag.Usage

func init() {
	flag.Usage = func() {
		fmt.Println(banner)
		originalUsage()
	}
	// command line help strings
	const (
		configUsage    = "path to the configuration TOML file, defaults to $HOME/gusano.toml, if present (e.g. -config myconf.toml)"
		excludeUsage   = "list of globs which specify files to be excluded (e.g. -exclude foo/...)"
		formatterUsage = "formatter to be used for the output (e.g. -formatter stylish)"
	)

	defaultConfigPath := buildDefaultConfigPath()

	flag.StringVar(&configPath, "config", defaultConfigPath, configUsage)
	flag.Var(&excludePaths, "exclude", excludeUsage)
	flag.StringVar(&formatterName, "formatter", "", formatterUsage)
	flag.Parse()
}
