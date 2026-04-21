package cli

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed skill/SKILL.md
var embeddedSkill string

// SkillCommandOptions configures the skill command.
type SkillCommandOptions struct {
	IO IO
}

// NewSkillCommand creates the "skill" command for showing or installing the llmcli skill.
func NewSkillCommand(opts SkillCommandOptions) *cobra.Command {
	ioCfg := opts.IO.WithDefaults()

	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Show or install the llmcli skill",
		Long: `Show or install the llmcli skill for Claude Code/CLI.

The skill file provides instructions for AI assistants on how to use llmcli.

Examples:
  llmcli skill                              # Show skill content
  llmcli skill show                         # Same as above
  llmcli skill install                      # Install to ~/.claude/skills/llmcli/
  llmcli skill install --path /custom/path  # Install to custom location`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default: show skill
			return showSkill(ioCfg)
		},
	}

	cmd.SetOut(ioCfg.Out)
	cmd.SetErr(ioCfg.Err)
	cmd.AddCommand(newSkillShowCommand(opts))
	cmd.AddCommand(newSkillInstallCommand(opts))

	return cmd
}

func newSkillShowCommand(opts SkillCommandOptions) *cobra.Command {
	ioCfg := opts.IO.WithDefaults()

	return &cobra.Command{
		Use:   "show",
		Short: "Print the llmcli skill to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showSkill(ioCfg)
		},
	}
}

func showSkill(ioCfg IO) error {
	_, err := fmt.Fprint(ioCfg.Out, embeddedSkill)
	return err
}

func newSkillInstallCommand(opts SkillCommandOptions) *cobra.Command {
	ioCfg := opts.IO.WithDefaults()
	var installPath string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install skill to ~/.claude/skills/llmcli/",
		Long: `Install the llmcli skill to the Claude Code/CLI skills directory.

By default, installs to ~/.claude/skills/llmcli/SKILL.md.
Use --path to specify a custom installation path.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return installSkill(ioCfg, installPath)
		},
	}

	cmd.Flags().StringVar(&installPath, "path", "", "Override installation path (default: ~/.claude/skills/llmcli/SKILL.md)")

	return cmd
}

func installSkill(ioCfg IO, customPath string) error {
	path := customPath
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home directory: %w", err)
		}
		path = filepath.Join(home, ".claude", "skills", "llmcli", "SKILL.md")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	// Write skill file
	if err := os.WriteFile(path, []byte(embeddedSkill), 0644); err != nil {
		return fmt.Errorf("write skill file: %w", err)
	}

	fmt.Fprintf(ioCfg.Out, "Installed llmcli skill to %s\n", path)
	return nil
}

// GetEmbeddedSkill returns the embedded skill content.
// Useful for programmatic access.
func GetEmbeddedSkill() string {
	return embeddedSkill
}
