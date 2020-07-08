package types

import (
	"bytes"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/bsc/rlp"
)

type BindPackageType uint8

const (
	BindTypeBind   BindPackageType = 0
	BindTypeUnbind BindPackageType = 1
)

type BindSynPackage struct {
	PackageType     BindPackageType
	Bep2TokenSymbol [32]byte
	ContractAddr    SmartChainAddress
	TotalSupply     *big.Int
	PeggyAmount     *big.Int
	Decimals        uint8
	ExpireTime      uint64
}

func DeserializeBindSynPackage(serializedPackage []byte) (*BindSynPackage, sdk.Error) {
	var pack BindSynPackage
	err := rlp.DecodeBytes(serializedPackage, &pack)
	if err != nil {
		return nil, ErrDeserializePackageFailed("deserialize bind syn package failed")
	}
	return &pack, nil
}

type BindAckPackage struct {
	Bep2TokenSymbol [32]byte
}

type BindStatus uint32

const (
	BindStatusSuccess          BindStatus = 0
	BindStatusRejected         BindStatus = 1
	BindStatusTimeout          BindStatus = 2
	BindStatusInvalidParameter BindStatus = 3
)

type ApproveBindSynPackage struct {
	Status          BindStatus
	Bep2TokenSymbol [32]byte
}

func DeserializeApproveBindSynPackage(serializedPackage []byte) (*ApproveBindSynPackage, sdk.Error) {
	var pack ApproveBindSynPackage
	err := rlp.DecodeBytes(serializedPackage, &pack)
	if err != nil {
		return nil, ErrDeserializePackageFailed("deserialize approve bind package failed")
	}
	return &pack, nil
}

type ApproveBindAckPackage struct {
	Bep2TokenSymbol [32]byte
}

type TransferInSynPackage struct {
	Bep2TokenSymbol   [32]byte
	ContractAddress   SmartChainAddress
	Amounts           []*big.Int
	ReceiverAddresses []sdk.AccAddress
	RefundAddresses   []SmartChainAddress
	ExpireTime        uint64
}

func DeserializeTransferInSynPackage(serializedPackage []byte) (*TransferInSynPackage, sdk.Error) {
	var tp TransferInSynPackage
	err := rlp.DecodeBytes(serializedPackage, &tp)
	if err != nil {
		return nil, ErrDeserializePackageFailed("deserialize transfer in package failed")
	}
	return &tp, nil
}

type TransferInRefundPackage struct {
	ContractAddr    SmartChainAddress
	RefundAmounts   []*big.Int
	RefundAddresses []SmartChainAddress
	RefundReason    RefundReason
}

type TransferOutSynPackage struct {
	Bep2TokenSymbol [32]byte
	ContractAddress SmartChainAddress
	Amount          *big.Int
	Recipient       SmartChainAddress
	RefundAddress   sdk.AccAddress
	ExpireTime      uint64
}

func DeserializeTransferOutSynPackage(serializedPackage []byte) (*TransferOutSynPackage, sdk.Error) {
	var tp TransferOutSynPackage
	err := rlp.DecodeBytes(serializedPackage, &tp)
	if err != nil {
		return nil, ErrDeserializePackageFailed("deserialize transfer out package failed")
	}
	return &tp, nil
}

type RefundReason uint32

const (
	UnboundToken        RefundReason = 1
	Timeout             RefundReason = 2
	InsufficientBalance RefundReason = 3
	Unknown             RefundReason = 4
)

type TransferOutRefundPackage struct {
	Bep2TokenSymbol [32]byte
	RefundAmount    *big.Int
	RefundAddr      sdk.AccAddress
	RefundReason    RefundReason
}

func DeserializeTransferOutRefundPackage(serializedPackage []byte) (*TransferOutRefundPackage, sdk.Error) {
	var tp TransferOutRefundPackage
	err := rlp.DecodeBytes(serializedPackage, &tp)
	if err != nil {
		return nil, ErrDeserializePackageFailed("deserialize transfer out refund package failed")
	}
	return &tp, nil
}

func SymbolToBytes(symbol string) [32]byte {
	// length of bound token symbol length should not be larger than 32
	serializedBytes := [32]byte{}
	copy(serializedBytes[:], symbol)
	return serializedBytes
}

func BytesToSymbol(symbolBytes [32]byte) string {
	bep2TokenSymbolBytes := make([]byte, 32, 32)
	copy(bep2TokenSymbolBytes[:], symbolBytes[:])
	return string(bytes.Trim(bep2TokenSymbolBytes, "\x00"))
}
