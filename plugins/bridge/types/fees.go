package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/fees"
)

const (
	BindRelayFeeName   = "crossBindRelayFee"
	UnbindRelayFeeName = "crossUnbindRelayFee"
	TransferOutFeeName = "crossTransferOutRelayFee"
)

func GetFee(feeName string) (sdk.Fee, sdk.Error) {
	calculator := fees.GetCalculator(feeName)
	if calculator == nil {
		return sdk.Fee{}, ErrFeeNotFound("missing calculator for fee type:" + feeName)
	}
	return calculator(nil), nil
}
