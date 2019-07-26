package swap

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculteRandomHash(t *testing.T) {
	randomNumberStr := "52fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c649"
	randomNumber, _ := hex.DecodeString(randomNumberStr)
	timestamp := int64(1564471835)

	randomNumberHash := CalculteRandomHash(randomNumber, timestamp)
	require.Equal(t, "be543130668282f267580badb1c956dacd4502be3b57846443c9921118ffa167", hex.EncodeToString(randomNumberHash))
}
