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
		"--version", "--", "-c", "-C", "-e", "-G", "-h", "-I", "-j", "-J", "-k",
		"-o", "-q", "-r", "-R", "-S", "-u", "-v", "-V", "-W", "-X", "-z",
	}
	return validateWlmOptions(wlmPbs, pbsOptions, forbiddenPbsOptions)
}

// ValidateSlurm checks that the specified slurm options are allowed.
// If any are not messages are returned in an array of errors.
func ValidateSlurm(slurmOptions []string) []error {
	forbiddenArgs := []string{
		"--ntasks-per-node=",
		"--gpus=", "-G",
		"--nodes=", "-N",
		"--ntasks=", "-n",
		"--chdir=", "-D",
		"--error=", "-e",
		"--output=", "-o",
		"--partition=", "-p",
		"--no-requeue",
		"--requeue",
	}
	errors := validateWlmOptions(wlmSlurm, slurmOptions, forbiddenArgs)

	errors = disallowGresGpuConfiguration(slurmOptions, errors)
	return errors
}

// disallowGresGpuConfiguration adds a validation error if --gres references a GPU resource.
func disallowGresGpuConfiguration(slurmOptions []string, errors []error) []error {
	for _, option := range slurmOptions {
		gresSpecs := strings.Split(strings.TrimSpace(option), "=")
		if gresSpecs[0] == "--gres" && len(gresSpecs) > 1 {
			// Expect --gres=<list> where entries in the list are of the form
			// "name[[:type]:count]", separated by commas.
			for _, gresExpression := range strings.Split(gresSpecs[1], ",") {
				err := check.TrueSilent(
					strings.Split(gresExpression, ":")[0] != "gpu",
					"slurm option --gres may not be used to configure GPU resources")
				if err != nil {
					errors = append(errors, err)
					break
				}
			}
		}
	}
	return errors
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
