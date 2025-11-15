package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/apenella/go-ansible/v2/pkg/execute"
	"github.com/apenella/go-ansible/v2/pkg/playbook"
	"github.com/spf13/cobra"
)

var (
	ansibleExtraVars     []string
	ansibleExtraVarsJSON string
)

// ansiblePlaybookCmd represents the ansible-playbook command
var ansiblePlaybookCmd = &cobra.Command{
	Use:   "ansible-playbook [playbook-path]",
	Short: "Execute an Ansible playbook",
	Long: `Execute an Ansible playbook using the go-ansible library.

This command allows you to run Ansible playbooks from the PatchMon agent.
The playbook path should be an absolute or relative path to a valid Ansible playbook YAML file.

Variables can be passed using --extra-vars (key=value pairs) or --extra-vars-json (JSON format).

Example:
  patchmon-agent ansible-playbook /etc/patchmon/profiles/myplaybook.yml
  patchmon-agent --ansible-playbook /etc/patchmon/profiles/myplaybook.yml
  patchmon-agent ansible-playbook playbook.yml --extra-vars "key1=value1 key2=value2"
  patchmon-agent ansible-playbook playbook.yml --extra-vars-json '{"key":"value"}'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		playbookPath := args[0]

		// Check if playbook file exists
		if _, err := os.Stat(playbookPath); os.IsNotExist(err) {
			return fmt.Errorf("playbook file not found: %s", playbookPath)
		}

		// Parse extra vars
		extraVars, err := parseExtraVars(ansibleExtraVars, ansibleExtraVarsJSON)
		if err != nil {
			return fmt.Errorf("failed to parse extra vars: %w", err)
		}

		return runAnsiblePlaybook(playbookPath, extraVars)
	},
}

func init() {
	ansiblePlaybookCmd.Flags().StringArrayVarP(&ansibleExtraVars, "extra-vars", "e", []string{}, "Extra variables as key=value pairs (can be specified multiple times)")
	ansiblePlaybookCmd.Flags().StringVar(&ansibleExtraVarsJSON, "extra-vars-json", "", "Extra variables as JSON string")
}

// parseKeyValuePairs parses a string containing key=value pairs, handling quoted strings with spaces
// This function handles both cases:
// 1. Single key=value pair: "key=value with spaces"
// 2. Multiple key=value pairs: "key1=value1 key2=value2"
func parseKeyValuePairs(input string) []string {
	var pairs []string
	var current strings.Builder
	inQuotes := false
	quoteChar := byte(0)

	input = strings.TrimSpace(input)
	if input == "" {
		return pairs
	}

	for i := 0; i < len(input); i++ {
		char := input[i]

		// Check for escaped quote
		if char == '\\' && i+1 < len(input) && (input[i+1] == '"' || input[i+1] == '\'') {
			if inQuotes {
				current.WriteByte(input[i+1])
			} else {
				current.WriteByte(char)
				current.WriteByte(input[i+1])
			}
			i++ // Skip next character
			continue
		}

		// Handle quote characters
		if char == '"' || char == '\'' {
			if !inQuotes {
				// Starting a quoted section
				inQuotes = true
				quoteChar = char
				current.WriteByte(char)
			} else if char == quoteChar {
				// Ending a quoted section
				inQuotes = false
				quoteChar = 0
				current.WriteByte(char)
			} else {
				// Different quote type inside quotes, treat as literal
				current.WriteByte(char)
			}
			continue
		}

		// If we encounter whitespace outside quotes, split here
		if !inQuotes && (char == ' ' || char == '\t') {
			if current.Len() > 0 {
				pairs = append(pairs, current.String())
				current.Reset()
			}
			// Skip whitespace
			continue
		}

		current.WriteByte(char)
	}

	// Add the last pair if there's anything left
	if current.Len() > 0 {
		pairs = append(pairs, current.String())
	}

	return pairs
}

// parseExtraVars parses extra vars from both key=value pairs and JSON format
func parseExtraVars(extraVarsArray []string, extraVarsJSON string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Parse JSON format if provided
	if extraVarsJSON != "" {
		var jsonVars map[string]interface{}
		if err := json.Unmarshal([]byte(extraVarsJSON), &jsonVars); err != nil {
			return nil, fmt.Errorf("invalid JSON format: %w", err)
		}
		for k, v := range jsonVars {
			result[k] = v
		}
	}

	// Parse key=value pairs, handling quoted strings with spaces
	for _, ev := range extraVarsArray {
		pairs := parseKeyValuePairs(ev)
		for _, pair := range pairs {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid extra-var format: %s (expected key=value)", pair)
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Try to parse as JSON value (for booleans, numbers, etc.), fallback to string
			var jsonValue interface{}
			// First try parsing the entire value as JSON (handles quotes)
			if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
				// If it's a string in JSON, unwrap it
				if str, ok := jsonValue.(string); ok {
					result[key] = str
				} else {
					result[key] = jsonValue
				}
			} else {
				// Remove quotes if present (handles both single and double quotes)
				if len(value) >= 2 {
					if (value[0] == '"' && value[len(value)-1] == '"') ||
						(value[0] == '\'' && value[len(value)-1] == '\'') {
						value = value[1 : len(value)-1]
					}
				}
				result[key] = value
			}
		}
	}

	return result, nil
}

// runAnsiblePlaybook executes an Ansible playbook
func runAnsiblePlaybook(playbookPath string, extraVars map[string]interface{}) error {
	logger.WithField("playbook", playbookPath).Info("Executing Ansible playbook")
	if len(extraVars) > 0 {
		logger.WithField("extra_vars", extraVars).Debug("Using extra variables")
	}

	ctx := context.Background()

	// Define the playbook options
	playbookOptions := &playbook.AnsiblePlaybookOptions{
		Inventory: "localhost,", // Default to localhost, can be made configurable
		ExtraVars: extraVars,
	}

	// Create the Ansible playbook command
	playbookCmd := &playbook.AnsiblePlaybookCmd{
		Playbooks:       []string{playbookPath},
		PlaybookOptions: playbookOptions,
	}

	// Create executor and execute the playbook
	exec := execute.NewDefaultExecute(
		execute.WithCmd(playbookCmd),
		execute.WithErrorEnrich(playbook.NewAnsiblePlaybookErrorEnrich()),
	)

	// Execute the playbook
	err := exec.Execute(ctx)
	if err != nil {
		logger.WithError(err).Error("Failed to execute Ansible playbook")
		return fmt.Errorf("playbook execution failed: %w", err)
	}

	logger.Info("✅ Ansible playbook executed successfully")
	fmt.Println("✅ Ansible playbook executed successfully")
	return nil
}
