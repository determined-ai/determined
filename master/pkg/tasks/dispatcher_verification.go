package tasks

import (
	"regexp"
	"strings"

	"github.com/determined-ai/determined/master/pkg/check"
)

const (
	wlmSlurm = "slurm"
	wlmPbs   = "PBS"
)

// ValidatePbs checks that the specified PBS options are allowed.
// If any are not messages are returned in an array of errors.
func ValidatePbs(pbsOptions []string) []error {
	// Ref: https://connect.us.cray.com/confluence/display/AT/Use+of+pbsbatch_args
	forbiddenPbsOptions := []string{
		"--version", "--", "-c", "-C", "-e", "-G", "-h", "-I", "-j", "-J", "-k", "-l",
		"-o", "-q", "-r", "-R", "-S", "-u", "-v", "-V", "-W", "-X", "-z",
	}
	return validateWlmOptions(wlmPbs, pbsOptions, forbiddenPbsOptions)
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
