package slashing

import "github.com/cosmos/cosmos-sdk/types"

type SideDowntimeSlashPackage struct {
	SideConsAddr  []byte        `json:"side_cons_addr"`
	SideHeight    uint64        `json:"side_height"`
	SideChainId   types.ChainID `json:"side_chain_id"`
	SideTimestamp uint64        `json:"side_timestamp"`
}
