package wizard

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
)

// ErrCancelled is returned when the user aborts the wizard with Ctrl+C.
var ErrCancelled = terminal.InterruptErr

var tenantIDRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

// CommonAzureRegions provides region choices with sensible defaults.
var CommonAzureRegions = []string{
	"westeurope",
	"northeurope",
	"francecentral",
	"eastus",
	"eastus2",
	"westus2",
	"uksouth",
	"swedencentral",
	"germanywestcentral",
	"canadacentral",
}

// ValidateTenantID validates a tenant UUID.
func ValidateTenantID(value interface{}) error {
	v := strings.TrimSpace(fmt.Sprintf("%v", value))
	if !tenantIDRegex.MatchString(v) {
		return fmt.Errorf("tenant ID must be a valid UUID")
	}
	return nil
}

// ValidateNonEmpty ensures a required value is provided.
func ValidateNonEmpty(value interface{}) error {
	if strings.TrimSpace(fmt.Sprintf("%v", value)) == "" {
		return fmt.Errorf("value is required")
	}
	return nil
}

// Prompter abstracts user interaction for testing.
type Prompter interface {
	Input(label, defaultValue string, validator survey.Validator) (string, error)
	Select(label string, options []string, defaultValue string) (string, error)
	Confirm(label string, defaultValue bool) (bool, error)
	MultiSelect(label string, options []string, defaults []string) ([]string, error)
}

// SurveyPrompter implements Prompter with survey/v2.
type SurveyPrompter struct{}

// NewSurveyPrompter returns a survey-based prompter.
func NewSurveyPrompter() *SurveyPrompter {
	return &SurveyPrompter{}
}

func (p *SurveyPrompter) Input(label, defaultValue string, validator survey.Validator) (string, error) {
	var value string
	err := survey.AskOne(&survey.Input{
		Message: label,
		Default: defaultValue,
	}, &value, survey.WithValidator(validator))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func (p *SurveyPrompter) Select(label string, options []string, defaultValue string) (string, error) {
	var value string
	err := survey.AskOne(&survey.Select{
		Message: label,
		Options: options,
		Default: defaultValue,
	}, &value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (p *SurveyPrompter) Confirm(label string, defaultValue bool) (bool, error) {
	var value bool
	err := survey.AskOne(&survey.Confirm{
		Message: label,
		Default: defaultValue,
	}, &value)
	if err != nil {
		return false, err
	}
	return value, nil
}

func (p *SurveyPrompter) MultiSelect(label string, options []string, defaults []string) ([]string, error) {
	var selected []string
	err := survey.AskOne(&survey.MultiSelect{
		Message: label,
		Options: options,
		Default: defaults,
	}, &selected)
	if err != nil {
		return nil, err
	}
	return selected, nil
}
