package tasks

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
)

const (
	wlmSlurm            = "slurm"
	wlmPbs              = "PBS"
	optEllRequiresValue = "PBS option -l requires a value: resource=value,... or place=..."
)

// ValidatePbs checks that the specified PBS options are allowed.
// If any are not messages are returned in an array of errors.
func ValidatePbs(pbsOptions []string) []error {
	// Ref: https://connect.us.cray.com/confluence/display/AT/Use+of+pbsbatch_args
	forbiddenPbsOptions := []string{
		"--version", "--", "-c", "-C", "-e", "-G", "-h", "-I", "-j", "-J", "-k",
		"-o", "-q", "-r", "-R", "-S", "-u", "-v", "-V", "-W", "-X", "-z",
	}
	validationErrors := validateWlmOptions(wlmPbs, pbsOptions, forbiddenPbsOptions)

	for _, option := range pbsOptions {
		if option = strings.TrimSpace(option); strings.HasPrefix(strings.TrimSpace(option), "-l") {
			validationErrors = validateResourceAndPlacementRequests(option, validationErrors)
		}
	}
	return validationErrors
}

func validateResourceAndPlacementRequests(option string, validationErrors []error) []error {
	switch {
	case isResourceRequest(option):
	case isPlacementRequest(option):
	case isResourceSelect(option):
		validationErrors = append(validationErrors,
			errors.Errorf("PBS option -l select is not configurable"))
	default:
		// If we fall through to here then we have a -l value we don't accept.
		validationErrors = append(validationErrors, errors.Errorf(optEllRequiresValue))
	}
	return validationErrors
}

// isResourceSelect returns true if the option is the disallowed -l select=<anything>.
func isResourceSelect(option string) bool {
	match, _ := regexp.MatchString("^-l\\s+select=.*$", option)
	return match
}

// isPlacementRequest returns true if the option is a placement request: -l place=...
func isPlacementRequest(option string) bool {
	match, _ := regexp.MatchString("^-l\\s+place=.*$", option)
	return match
}

// isResourceRequest returns true if the opion is a resource request:
// -l <resource name>=<value>[:<resource name>=<value> ...].
func isResourceRequest(option string) bool {
	match, _ := regexp.MatchString("^-l\\s+\\w+=\\w+(,\\w+=\\w+)*$", option)
	return match
}

// ValidateSlurm checks that the specified slurm options are allowed.
// If any are not messages are returned in an array of errors.
func ValidateSlurm(slurm []string) []error {
	forbiddenArgs := []string{
		"--ntasks-per-node=",
		"--gpus=", "-G",
		"--gres=",
		"--nodes=", "-N",
		"--ntasks=", "-n",
		"--chdir=", "-D",
		"--error=", "-e",
		"--output=", "-o",
		"--partition=", "-p",
	}
	return validateWlmOptions(wlmSlurm, slurm, forbiddenArgs)
}

// validateWlmOptions validates the specified options against the WLM-specific
// list of disallowed options.
func validateWlmOptions(wlm string, options []string, forbiddenOptions []string) []error {
	validationErrors := []error{}
	for _, arg := range options {
		for _, forbidden := range forbiddenOptions {
			// If an arg starts with a forbidden option
			matches := strings.HasPrefix(strings.TrimSpace(arg), forbidden)
			matchesAdditional := false
			if wlm == wlmPbs {
				// Or if contains a forbidden option (PBS only)
				matchesAdditional, _ = regexp.MatchString("\\s+"+forbidden, strings.TrimSpace(arg))
			}
			// then add an error.
			err := check.TrueSilent(!(matches || matchesAdditional),
				wlm+" option "+forbidden+" is not configurable")
			if err != nil {
				validationErrors = append(validationErrors, err)
			}
		}
	}
	return validationErrors
}
