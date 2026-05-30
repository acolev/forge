package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

// WizardResult is the configuration produced by RunWizard.
type WizardResult struct {
	EnvFile       string
	DSN           string
	Driver        string
	PluginsDir    string
	ModelsDir     string
	ModelsPackage string
}

// RunWizard runs the interactive configuration wizard, pre-filling defaults from
// any existing .env.forge, and writes the result back to it. If test is non-nil,
// the wizard offers to verify the connection using the freshly built DSN.
func RunWizard(in io.Reader, out io.Writer, test func(dsn string) error) (WizardResult, error) {
	p := &prompter{r: bufio.NewReader(in), w: out}
	envPath := DefaultEnvFile

	parts := ParseDSN(ReadEnvFileValue(envPath, ForgeDBDSNKey))

	fmt.Fprintln(out, "Forge configuration — press Enter to accept the default shown in (parentheses).")
	fmt.Fprintln(out)

	driver := p.askChoice("Database driver", []string{"sqlite", "postgres", "mysql"}, parts.Driver)
	np := DSNParts{Driver: driver}

	if driver == "sqlite" {
		np.SQLitePath = p.ask("SQLite file path", orDefault(parts.SQLitePath, DefaultSQLiteDBPath))
	} else {
		np.Host = p.ask("Host", orDefault(parts.Host, "localhost"))
		np.Port = p.ask("Port", orDefault(parts.Port, DefaultPort(driver)))
		np.User = p.ask("User", parts.User)
		np.Password = p.askPassword("Password", parts.Password)
		np.DBName = p.ask("Database name", parts.DBName)
	}

	dsn, err := BuildDSN(np)
	if err != nil {
		return WizardResult{}, err
	}

	pluginsDir := p.ask("Plugins dir", orDefault(ReadEnvFileValue(envPath, ForgePluginsDirKey), DefaultPluginsDir))
	modelsDir := p.ask("Models output dir", orDefault(ReadEnvFileValue(envPath, ForgeModelsDirKey), DefaultModelsDir))
	modelsPkg := p.ask("Models package", orDefault(ReadEnvFileValue(envPath, ForgeModelsPackageKey), DefaultModelsPackage))

	kv := map[string]string{
		ForgeDBDSNKey:         dsn,
		ForgePluginsDirKey:    pluginsDir,
		ForgeModelsDirKey:     modelsDir,
		ForgeModelsPackageKey: modelsPkg,
	}
	if err := UpdateEnvFile(envPath, kv); err != nil {
		return WizardResult{}, err
	}

	fmt.Fprintf(out, "\nWrote %s\n", envPath)

	if test != nil && askYesNo(p, "Test database connection now?") {
		// sqlite needs the parent directory to exist before the file can be created.
		if driver == "sqlite" {
			if d := filepath.Dir(np.SQLitePath); d != "" && d != "." {
				_ = os.MkdirAll(d, 0o755)
			}
		}
		if err := test(dsn); err != nil {
			fmt.Fprintf(out, "✗ Connection failed: %v\n", err)
		} else {
			fmt.Fprintln(out, "✓ Connected.")
		}
	}

	return WizardResult{
		EnvFile:       envPath,
		DSN:           dsn,
		Driver:        driver,
		PluginsDir:    pluginsDir,
		ModelsDir:     modelsDir,
		ModelsPackage: modelsPkg,
	}, nil
}

func askYesNo(p *prompter, label string) bool {
	ans := strings.ToLower(p.ask(label+" [y/N]", "n"))
	return ans == "y" || ans == "yes"
}

type prompter struct {
	r *bufio.Reader
	w io.Writer
}

func (p *prompter) ask(label, def string) string {
	if def != "" {
		fmt.Fprintf(p.w, "%s (%s): ", label, def)
	} else {
		fmt.Fprintf(p.w, "%s: ", label)
	}
	line, _ := p.r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func (p *prompter) askChoice(label string, options []string, def string) string {
	prompt := fmt.Sprintf("%s [%s]", label, strings.Join(options, "/"))
	for {
		v := strings.ToLower(p.ask(prompt, def))
		for _, o := range options {
			if v == o {
				return v
			}
		}
		fmt.Fprintf(p.w, "  please choose one of: %s\n", strings.Join(options, ", "))
		if def != "" {
			return def
		}
	}
}

// askPassword reads a password with echo disabled when attached to a terminal,
// and falls back to a normal line read otherwise (pipes, tests).
func (p *prompter) askPassword(label, def string) string {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		if def != "" {
			fmt.Fprintf(p.w, "%s (leave blank to keep current): ", label)
		} else {
			fmt.Fprintf(p.w, "%s: ", label)
		}
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(p.w)
		if err != nil {
			return def
		}
		s := strings.TrimSpace(string(b))
		if s == "" {
			return def
		}
		return s
	}
	return p.ask(label, def)
}

func orDefault(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}
