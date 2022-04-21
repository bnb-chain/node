package types

import (
	"fmt"
	"math"
	"strconv"

	"github.com/tendermint/tendermint/crypto"
)

const (
	pegInTagName  = "peg_in_%s"
	pegOutTagName = "peg_out_%s"
)

var (
	// bnb prefix address:  bnb1v8vkkymvhe2sf7gd2092ujc6hweta38xadu2pj
	// tbnb prefix address: tbnb1v8vkkymvhe2sf7gd2092ujc6hweta38xnc4wpr
	PegAccount = AccAddress(crypto.AddressHash([]byte("BinanceChainPegAccount")))
)

func GetPegInTag(symbol string, amount int64) Tag {
	return MakeTag(fmt.Sprintf(pegInTagName, symbol), []byte(strconv.FormatInt(amount, 10)))
}

func GetPegOutTag(symbol string, amount int64) Tag {
	return MakeTag(fmt.Sprintf(pegOutTagName, symbol), []byte(strconv.FormatInt(amount, 10)))
}

type CrossChainPackageType uint8

type ChannelID uint8
type ChainID uint16

const (
	SynCrossChainPackageType     CrossChainPackageType = 0x00
	AckCrossChainPackageType     CrossChainPackageType = 0x01
	FailAckCrossChainPackageType CrossChainPackageType = 0x02
)

type ChannelPermission uint8

const (
	ChannelAllow     ChannelPermission = 1
	ChannelForbidden ChannelPermission = 0
)

func IsValidCrossChainPackageType(packageType CrossChainPackageType) bool {
	return packageType == SynCrossChainPackageType || packageType == AckCrossChainPackageType || packageType == FailAckCrossChainPackageType
}

func ParseChannelID(input string) (ChannelID, error) {
	channelID, err := strconv.Atoi(input)
	if err != nil {
		return ChannelID(0), err
	}
	if channelID > math.MaxInt8 || channelID < 0 {
		return ChannelID(0), fmt.Errorf("channelID must be in [0, 255]")
	}
	return ChannelID(channelID), nil
}

func ParseChainID(input string) (ChainID, error) {
	chainID, err := strconv.Atoi(input)
	if err != nil {
		return ChainID(0), err
	}
	if chainID > math.MaxUint16 || chainID < 0 {
		return ChainID(0), fmt.Errorf("cross chainID must be in [0, 65535]")
	}
	return ChainID(chainID), nil
}

type CrossChainApplication interface {
	ExecuteSynPackage(ctx Context, payload []byte, relayerFee int64) ExecuteResult
	ExecuteAckPackage(ctx Context, payload []byte) ExecuteResult
	// When the ack application crash, payload is the payload of the origin package.
	ExecuteFailAckPackage(ctx Context, payload []byte) ExecuteResult
}

type ExecuteResult struct {
	Err     Error
	Tags    Tags
	Payload []byte
}

func (c ExecuteResult) IsOk() bool {
	return c.Err == nil || c.Err.ABCICode().IsOK()
}

func (c ExecuteResult) Code() ABCICodeType {
	if c.Err == nil {
		return ABCICodeOK
	}
	return c.Err.ABCICode()
}

func (c ExecuteResult) Msg() string {
	if c.Err == nil {
		return ""
	}
	return c.Err.RawError()
}
