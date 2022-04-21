package gov

//nolint
const (
	// side chain params change
	ProposalTypeSCParamsChange ProposalKind = 0x81
	// cross side chain param change
	ProposalTypeCSCParamsChange ProposalKind = 0x82
)

func validSideProposalType(pt ProposalKind) bool {
	if pt == ProposalTypeSCParamsChange ||
		pt == ProposalTypeCSCParamsChange {
		return true
	}
	return false
}
