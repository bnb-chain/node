package types

import (
	"fmt"
	"math/big"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TagSendSequence = "SendSequence"
	TagChannel      = "Channel"
	TagRelayerFee   = "relayerFee"

	transferInSuccess = "transferInSuccess_%s_%s"
	transferInRefund = "transferInRefund_%s_%s"
)

func GenerateTransferInTags(receiverAddresses []sdk.AccAddress, symbol string, amounts []*big.Int, excludeList []sdk.AccAddress) sdk.Tags {
	tags := sdk.EmptyTags()
	excludeMap := make(map[string]bool, len(excludeList))
	for _, addr := range excludeList {
		excludeMap[addr.String()] = true
	}
	for idx, receiver := range receiverAddresses {
		if excludeMap[receiver.String()] {
			tags = tags.AppendTag(fmt.Sprintf(transferInSuccess, symbol, receiver.String()), []byte(strconv.FormatInt(amounts[idx].Int64(), 10)))
		} else {
			tags = tags.AppendTag(fmt.Sprintf(transferInRefund, symbol, receiver.String()), []byte(strconv.FormatInt(amounts[idx].Int64(), 10)))
		}

	}
	return tags
}
