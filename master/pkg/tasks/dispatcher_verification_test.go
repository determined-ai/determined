package tasks

import (
	"testing"

	"gotest.tools/assert"
)

// Helper function to setup and verify slurm option test cases.
func testEnvironmentSlurm(t *testing.T, slurmOptions []string, expected ...string) {
	validateEnvironmentResult(expected, t, ValidateSlurm(slurmOptions))
}

// Helper function to setup and verify PBS option test cases.
func testEnvironmentPbs(t *testing.T, options []string, expected ...string) {
	validateEnvironmentResult(expected, t, ValidatePbs(options))
}

func validateEnvironmentResult(expected []string, t *testing.T, err []error) {
	if len(expected) == 0 {
		assert.Equal(t, len(err), 0, "Got unexpected errors", err)
	} else {
		assert.Assert(t, len(err) > 0, "Expected some errors", expected)
		for i, msg := range expected {
			assert.ErrorContains(t, err[i], msg)
		}
	}
}

func TestValidateSlurmOptions(t *testing.T) {
	// No slurm args, not error
	testEnvironmentSlurm(t, []string{})
	// Forbidden -G option
	testEnvironmentSlurm(t, []string{"-G1"}, "slurm option -G is not configurable")
	// Forbidden --grpus=#
	testEnvironmentSlurm(t, []string{"--gpus=2"}, "slurm option --gpus= is not configurable")
	// OK --gpus-per-task=#
	testEnvironmentSlurm(t, []string{"--gpus-per-task=2"})
	// OK option containing letters of forbidden option (-n)
	testEnvironmentSlurm(t, []string{"--nice=3"})
	// OK even though it appears to contain a forbidden option (-n)
	testEnvironmentSlurm(t, []string{"-J -nameOfJob"})
	// Forbidden -n option intermixed with OK options
	testEnvironmentSlurm(t, []string{"--nice=7", "-n3", "-lname"},
		"slurm option -n is not configurable")
	// Multiple failures
	testEnvironmentSlurm(t, []string{"--nice=7", " -N2", "-Dmydir", "--partion=pname"},
		"slurm option -N is not configurable",
		"slurm option -D is not configurable")

	// --gres -- is allowed unless specifying GPU resources
	testEnvironmentSlurm(t, []string{"--gres=gpu:tesla:100,cpu:100"},
		"slurm option --gres may not be used to configure GPU resources")
	testEnvironmentSlurm(t, []string{"--gres=cpu:100,gpu:100"},
		"slurm option --gres may not be used to configure GPU resources")
	testEnvironmentSlurm(t, []string{"--gres=cpu:100"})
	testEnvironmentSlurm(t, []string{"--gres=cpu:100,"})
	testEnvironmentSlurm(t, []string{"--gres="})
	testEnvironmentSlurm(t, []string{"--gres=,"})
	testEnvironmentSlurm(t, []string{"--gres"})

	var slurmArgs []string
	testEnvironmentSlurm(t, slurmArgs)
}

func TestValidatePbsOptions(t *testing.T) {
	// No args, not error
	testEnvironmentPbs(t, []string{})

	// These options are all allowed
	testEnvironmentPbs(t, []string{
		"-A account-name",
		"-a date-and=time",
		"-f",
		"-m events",
		"-M users",
		"-N job-name",
		"-p priority",
		"-P project",
		"-l name=value",
		"-l name=value,name2=value",
		"-l place=arrangement",
		"-l place=arrangement:sharing",
		"-l place=arrangement:sharing:grouping",
		"-l place=pack:group=arch",
		"-l select=chunks",
	})

	// These are not allowed
	testEnvironmentPbs(t, []string{"--"}, "PBS option -- is not configurable")
	testEnvironmentPbs(t, []string{"-c checkpoint-spec"}, "PBS option -c is not configurable")
	testEnvironmentPbs(t, []string{"-C NotPBS"}, "PBS option -C is not configurable")
	testEnvironmentPbs(t, []string{"-e path"}, "PBS option -e is not configurable")
	testEnvironmentPbs(t, []string{"-G script"}, "PBS option -G is not configurable")
	testEnvironmentPbs(t, []string{"-h"}, "PBS option -h is not configurable")
	testEnvironmentPbs(t, []string{"-I"}, "PBS option -I is not configurable")
	testEnvironmentPbs(t, []string{"-j join"}, "PBS option -j is not configurable")
	testEnvironmentPbs(t, []string{"-J range"}, "PBS option -J is not configurable")
	testEnvironmentPbs(t, []string{"-k discard"}, "PBS option -k is not configurable")
	testEnvironmentPbs(t, []string{"-o path"}, "PBS option -o is not configurable")
	testEnvironmentPbs(t, []string{"-q queue"}, "PBS option -q is not configurable")
	testEnvironmentPbs(t, []string{"-r yn"}, "PBS option -r is not configurable")
	testEnvironmentPbs(t, []string{"-R opts"}, "PBS option -R is not configurable")
	testEnvironmentPbs(t, []string{"-S path"}, "PBS option -S is not configurable")
	testEnvironmentPbs(t, []string{"-u users"}, "PBS option -u is not configurable")
	testEnvironmentPbs(t, []string{"-v list"}, "PBS option -v is not configurable")
	testEnvironmentPbs(t, []string{"-V"}, "PBS option -V is not configurable")
	testEnvironmentPbs(t, []string{"-W attribs"}, "PBS option -W is not configurable")
	testEnvironmentPbs(t, []string{"-z"}, "PBS option -z is not configurable")
	testEnvironmentPbs(t, []string{"--version"}, "PBS option --version is not configurable")

	// A sneaky test specifying both valid & invalid options in the same argument
	testEnvironmentPbs(t, []string{"-A myAccount   -I"}, "PBS option -I is not configurable")
}
