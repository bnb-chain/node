package app

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"

	"github.com/BiJie/BinanceChain/common/log"
	"github.com/BiJie/BinanceChain/common/tx"
	"github.com/BiJie/BinanceChain/common/types"
)

func distributeFee(ctx sdk.Context, am auth.AccountKeeper) {
	proposerAddr := ctx.BlockHeader().ProposerAddress
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
		voteInfos := ctx.VoteInfos()
		valSize := int64(len(voteInfos))
		log.Info("Distributing the fees to all the validators",
			"totalFees", fee.Tokens, "validatorSize", valSize)
		avgTokens := sdk.Coins{}
		roundingTokens := sdk.Coins{}
		for _, token := range fee.Tokens {
			// TODO: int64 is enough, will drop big.Int
			// TODO: temporarily, the validators average the fees. Will change to use power as a weight to calc fees.
			amount := token.Amount.Int64()
			avgAmount := amount / valSize
			roundingAmount := amount - avgAmount*valSize
			if avgAmount != 0 {
				avgTokens = append(avgTokens, sdk.NewInt64Coin(token.Denom, avgAmount))
			}

			if roundingAmount != 0 {
				roundingTokens = append(roundingTokens, sdk.NewInt64Coin(token.Denom, roundingAmount))
			}
		}

		if avgTokens.IsZero() {
			proposerAcc := am.GetAccount(ctx, proposerAddr)
			proposerAcc.SetCoins(proposerAcc.GetCoins().Plus(fee.Tokens))
			am.SetAccount(ctx, proposerAcc)
		} else {
			for _, voteInfo := range voteInfos {
				validator := voteInfo.Validator
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
