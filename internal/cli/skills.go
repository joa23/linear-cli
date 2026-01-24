package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/joa23/linear-cli/internal/skills"
	"github.com/spf13/cobra"
)

func newSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage Claude Code skills for Linear workflows",
		Long: `Install and manage Claude Code skills for agentic Linear workflows.

Skills are installed to .claude/skills/ in your project directory and provide
specialized workflows like PRD creation, backlog triage, and cycle planning.`,
	}

	cmd.AddCommand(
		newSkillsListCmd(),
		newSkillsInstallCmd(),
	)

	return cmd
}

func newSkillsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available skills",
		Long:  "Display all available Linear skills that can be installed.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("AVAILABLE SKILLS")
			fmt.Println(strings.Repeat("─", 60))
			fmt.Println()

			for _, skill := range skills.AvailableSkills {
				fmt.Printf("  /%s\n", skill.Name)
				fmt.Printf("    %s\n\n", skill.Description)
			}

			fmt.Println(strings.Repeat("─", 60))
			fmt.Println("Install with: linear skills install [skill-name|--all]")

			return nil
		},
	}
}

func newSkillsInstallCmd() *cobra.Command {
	var installAll bool

	cmd := &cobra.Command{
		Use:   "install [skill-name]",
		Short: "Install skills to .claude/skills/",
		Long: `Install Linear skills to .claude/skills/ in your project.

Skills provide Claude Code with specialized workflows for Linear management.
Each skill is a directory containing SKILL.md and optionally template files.

Note: This command will NOT overwrite existing files. If a skill already
exists, you must remove it manually first.`,
		Example: `  # Install all skills
  linear skills install --all

  # Install a specific skill
  linear skills install prd
  linear skills install triage`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !installAll && len(args) == 0 {
				return fmt.Errorf("specify a skill name or use --all to install all skills")
			}

			// Determine target directory
			targetDir := ".claude/skills"

			// Create base directory if needed
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return fmt.Errorf("failed to create skills directory: %w", err)
			}

			var toInstall []skills.SkillInfo

			if installAll {
				toInstall = skills.AvailableSkills
			} else {
				skillName := args[0]
				skill := skills.GetSkillByName(skillName)
				if skill == nil {
					return fmt.Errorf("unknown skill: %s\nRun 'linear skills list' to see available skills", skillName)
				}
				toInstall = []skills.SkillInfo{*skill}
			}

			installed := 0
			for _, skill := range toInstall {
				err := installSkill(skill, targetDir)
				if err != nil {
					fmt.Printf("  [SKIP] %s: %v\n", skill.Name, err)
				} else {
					fmt.Printf("  [OK] /%s installed\n", skill.Name)
					installed++
				}
			}

			fmt.Println()
			if installed > 0 {
				fmt.Printf("Installed %d skill(s) to %s/\n", installed, targetDir)
				fmt.Println("\nUse /<skill-name> in Claude Code to invoke.")
			} else {
				fmt.Println("No skills were installed.")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&installAll, "all", false, "Install all available skills")

	return cmd
}

func installSkill(skill skills.SkillInfo, targetDir string) error {
	skillDir := filepath.Join(targetDir, skill.Name)

	// Check if directory already exists
	if _, err := os.Stat(skillDir); err == nil {
		return fmt.Errorf("already exists (remove manually to reinstall)")
	}

	// Create skill directory
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Copy all files from embedded filesystem
	err := fs.WalkDir(skills.SkillFiles, skill.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory
		if path == skill.Dir {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(skill.Dir, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(skillDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// Read and write file
		content, err := skills.SkillFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		return os.WriteFile(targetPath, content, 0644)
	})

	if err != nil {
		// Clean up on failure
		_ = os.RemoveAll(skillDir)
		return fmt.Errorf("failed to copy files: %w", err)
	}

	return nil
}
