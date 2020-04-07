package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/binance-chain/node/common/fees"
	"github.com/binance-chain/node/common/types"
)

const (
	BindRelayFeeName   = "crossBindRelayFee"
	TransferOutFeeName = "crossTransferOutRelayFee"
)

func GetFee(feeName string) (types.Fee, sdk.Error) {
	calculator := fees.GetCalculator(feeName)
	if calculator == nil {
		return types.Fee{}, ErrFeeNotFound("missing calculator for fee type:" + feeName)
	}
	return calculator(nil), nil
}
