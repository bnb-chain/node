package slashing

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSetGetSlashRecord(t *testing.T) {
	ctx, _, _, _, keeper := createTestInput(t, DefaultParams())
	sideConsAddr := randomSideConsAddr()
	iHeight := uint64(100)
	sHeight := int64(150)
	jailUtil := time.Now().Add(60 * 60 * time.Second)
	sr := SlashRecord{
		ConsAddr:         sideConsAddr,
		InfractionType:   DoubleSign,
		InfractionHeight: iHeight,
		SlashHeight:      sHeight,
		JailUntil:        jailUtil,
		SlashAmt:         100e8,
	}
	keeper.setSlashRecord(ctx, sr)

	getSr, found := keeper.getSlashRecord(ctx, sideConsAddr, DoubleSign, iHeight)
	require.True(t, found)
	require.EqualValues(t, sr.ConsAddr, getSr.ConsAddr)
	require.EqualValues(t, sr.InfractionType, getSr.InfractionType)
	require.EqualValues(t, sr.InfractionHeight, getSr.InfractionHeight)
	require.EqualValues(t, sr.SlashHeight, getSr.SlashHeight)
	require.EqualValues(t, sr.JailUntil.Second(), getSr.JailUntil.Second())
	require.EqualValues(t, sr.SlashAmt, getSr.SlashAmt)

	require.True(t, keeper.hasSlashRecord(ctx, sideConsAddr, DoubleSign, iHeight))
}

func randomSideConsAddr() []byte {
	bz := make([]byte, 20)
	rand.Read(bz)
	return bz
}
