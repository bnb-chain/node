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

	TagMirrorContract  = "contract"
	TagMirrorSymbol    = "symbol"
	TagMirrorSupply    = "supply"
	TagMirrorErrorCode = "errCode"

	transferInSuccess = "transferInSuccess_%s_%s"
	transferInRefund  = "transferInRefund_%s_%s"
)

func GenerateTransferInTags(receiverAddresses []sdk.AccAddress, symbol string, amounts []*big.Int, isRefund bool) sdk.Tags {
	tags := sdk.EmptyTags()
	for idx, receiver := range receiverAddresses {
		tags = tags.AppendTag(fmt.Sprintf(transferInSuccess, symbol, receiver.String()), []byte(strconv.FormatInt(amounts[idx].Int64(), 10)))
		if isRefund {
			tags = tags.AppendTag(fmt.Sprintf(transferInRefund, symbol, receiver.String()), []byte(strconv.FormatInt(amounts[idx].Int64(), 10)))
		} else {
			tags = tags.AppendTag(fmt.Sprintf(transferInSuccess, symbol, receiver.String()), []byte(strconv.FormatInt(amounts[idx].Int64(), 10)))
		}
	}
	return tags
}
