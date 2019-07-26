package swap

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestCalculateRandomHash(t *testing.T) {
	randomNumberStr := "52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649"
	randomNumber, _ := hex.DecodeString(randomNumberStr)
	timestamp := int64(1564471835)

	randomNumberHash := CalculateRandomHash(randomNumber, timestamp)
	require.Equal(t, "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167", hex.EncodeToString(randomNumberHash))
}

func TestCalculateSwapID(t *testing.T) {
	randomNumberHashStr := "52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649"
	randomNumberHash, _ := hex.DecodeString(randomNumberHashStr)

	sender := sdk.AccAddress(crypto.AddressHash([]byte("sender")))
	senderOtherChain := "0x833914c3A745d924bf71d98F9F9Ae126993E3C88"

	swapID := CalculateSwapID(randomNumberHash, sender, senderOtherChain)
	require.Equal(t, "1e2103882a9da088befc55eea4d25b6ef0a634ef483c6249615fa62078f0dc79", hex.EncodeToString(swapID))
}
