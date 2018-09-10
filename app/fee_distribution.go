package app

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/BiJie/BinanceChain/common/tx"

	"github.com/BiJie/BinanceChain/common/types"
)

func distributeFee(ctx types.Context, am auth.AccountMapper) {
	proposerAddr := ctx.BlockHeader().Proposer.Address
	// extract fees from ctx
	fee := tx.Fee(ctx)
	if fee.IsEmpty() {
		// no fees in this block
		return
	}

	if fee.Type == types.FeeForProposer {
		// The proposer's account must be initialized before it becomes a proposer.
		proposerAcc := am.GetAccount(ctx, proposerAddr)
		proposerAcc.SetCoins(proposerAcc.GetCoins().Plus(fee.Tokens))
		am.SetAccount(ctx, proposerAcc)
	} else if fee.Type == types.FeeForAll {
		signingValidators := ctx.VoteInfos()
		valSize := int64(len(signingValidators))
		avgTokens := sdk.Coins{}
		roundingTokens := sdk.Coins{}
		for _, token := range fee.Tokens {
			// TODO: int64 is enough, will drop big.Int
			// TODO: temporarily, the validators average the fees. Will change to use power as a weight to calc fees.
			amount := token.Amount.Int64()
			avgAmount := amount / valSize
			roundingAmount := amount - avgAmount*valSize
			if avgAmount != 0 {
				avgTokens = append(avgTokens, sdk.NewCoin(token.Denom, avgAmount))
			}

			if roundingAmount != 0 {
				roundingTokens = append(roundingTokens, sdk.NewCoin(token.Denom, roundingAmount))
			}
		}

		if avgTokens.IsZero() {
			proposerAcc := am.GetAccount(ctx, proposerAddr)
			proposerAcc.SetCoins(proposerAcc.GetCoins().Plus(fee.Tokens))
			am.SetAccount(ctx, proposerAcc)
		} else {
			for _, signingValidator := range signingValidators {
				validator := signingValidator.Validator
				validatorAcc := am.GetAccount(ctx, validator.Address)
				if bytes.Equal(proposerAddr, validator.Address) && !roundingTokens.IsZero() {
					validatorAcc.SetCoins(validatorAcc.GetCoins().Plus(roundingTokens))
				}
				validatorAcc.SetCoins(validatorAcc.GetCoins().Plus(avgTokens))
				am.SetAccount(ctx, validatorAcc)
			}
		}
	}
}
