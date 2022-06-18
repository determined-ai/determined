package user

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatching(t *testing.T) {
	testPath := "/agent/match"
	service := Service{}
	require.True(t, service.doesMatch(testPath, testPath))

	testURI := "/agent/no/match"
	require.False(t, service.doesMatch(testURI, testPath))
}

func TestMatchingWildcard(t *testing.T) {
	testPath := "/agent*"
	testURI := "/agent"
	service := Service{}
	require.True(t, service.doesMatch(testURI, testPath))

	testURI = "/agent/longer/path/match/all"
	require.True(t, service.doesMatch(testURI, testPath))

	testURI = "/agent-match/with/more"
	require.True(t, service.doesMatch(testURI, testPath))

	testURI = "no/agent/match"
	require.False(t, service.doesMatch(testURI, testPath))
}

func TestMatchingWildMiddle(t *testing.T) {
	testPath := "/agent/*/test"
	testURI := "/agent/more/test"
	service := Service{}
	require.True(t, service.doesMatch(testURI, testPath))

	testURI = "/agent/big/match/yes/test"
	require.True(t, service.doesMatch(testURI, testPath))

	testURI = "/agent/test/no/match"
	require.False(t, service.doesMatch(testURI, testPath))

	testURI = "no/agent/match/test"
	require.False(t, service.doesMatch(testURI, testPath))
}
