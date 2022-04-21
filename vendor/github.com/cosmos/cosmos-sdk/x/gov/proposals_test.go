package gov_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/x/gov"
)

func TestProposalKind_Format(t *testing.T) {
	typeText, _ := gov.ProposalTypeFromString("Text")
	tests := []struct {
		pt                   gov.ProposalKind
		sprintFArgs          string
		expectedStringOutput string
	}{
		{typeText, "%s", "Text"},
		{typeText, "%v", "1"},
	}
	for _, tt := range tests {
		got := fmt.Sprintf(tt.sprintFArgs, tt.pt)
		require.Equal(t, tt.expectedStringOutput, got)
	}
}

func TestProposalStatus_Format(t *testing.T) {
	statusDepositPeriod, _ := gov.ProposalStatusFromString("DepositPeriod")
	tests := []struct {
		pt                   gov.ProposalStatus
		sprintFArgs          string
		expectedStringOutput string
	}{
		{statusDepositPeriod, "%s", "DepositPeriod"},
		{statusDepositPeriod, "%v", "1"},
	}
	for _, tt := range tests {
		got := fmt.Sprintf(tt.sprintFArgs, tt.pt)
		require.Equal(t, tt.expectedStringOutput, got)
	}
}
