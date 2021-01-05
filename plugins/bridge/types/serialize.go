package types

import (
	"bytes"
	"math/big"

	"github.com/cosmos/cosmos-sdk/bsc/rlp"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type BindPackageType uint8

const (
	BindTypeBind   BindPackageType = 0
	BindTypeUnbind BindPackageType = 1
)

type BindSynPackage struct {
	PackageType  BindPackageType
	TokenSymbol  [32]byte
	ContractAddr SmartChainAddress
	TotalSupply  *big.Int
	PeggyAmount  *big.Int
	Decimals     uint8
	ExpireTime   uint64
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
	TokenSymbol [32]byte
}

type BindStatus uint32

const (
	BindStatusSuccess          BindStatus = 0
	BindStatusRejected         BindStatus = 1
	BindStatusTimeout          BindStatus = 2
	BindStatusInvalidParameter BindStatus = 3
)

type ApproveBindSynPackage struct {
	Status      BindStatus
	TokenSymbol [32]byte
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
	TokenSymbol [32]byte
}

type TransferInSynPackage struct {
	TokenSymbol       [32]byte
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
	TokenSymbol     [32]byte
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
	UnboundToken              RefundReason = 1
	Timeout                   RefundReason = 2
	InsufficientBalance       RefundReason = 3
	Unknown                   RefundReason = 4
	ForbidTransferToBPE12Addr RefundReason = 5
)

type TransferOutRefundPackage struct {
	TokenSymbol  [32]byte
	RefundAmount *big.Int
	RefundAddr   sdk.AccAddress
	RefundReason RefundReason
}

func DeserializeTransferOutRefundPackage(serializedPackage []byte) (*TransferOutRefundPackage, sdk.Error) {
	var tp TransferOutRefundPackage
	err := rlp.DecodeBytes(serializedPackage, &tp)
	if err != nil {
		return nil, ErrDeserializePackageFailed("deserialize transfer out refund package failed")
	}
	return &tp, nil
}

type MirrorSynPackage struct {
	MirrorSender     SmartChainAddress
	ContractAddr     SmartChainAddress
	BEP20Name        [32]byte
	BEP20Symbol      [32]byte
	BEP20TotalSupply *big.Int
	BEP20Decimals    uint8
	MirrorFee        *big.Int
	ExpireTime       uint64
}

func DeserializeMirrorSynPackage(serializedPackage []byte) (*MirrorSynPackage, sdk.Error) {
	var ms MirrorSynPackage
	err := rlp.DecodeBytes(serializedPackage, &ms)
	if err != nil {
		return nil, ErrDeserializePackageFailed("deserialize mirror package failed")
	}
	return &ms, nil
}

const (
	MirrorErrCodeExpired          uint8 = 1
	MirrorErrCodeBEP2SymbolExists uint8 = 2
	MirrorErrCodeDecimalOverflow  uint8 = 3
	MirrorErrCodeInvalidSymbol    uint8 = 4
	MirrorErrCodeInvalidSupply    uint8 = 5
)

type MirrorAckPackage struct {
	MirrorSender SmartChainAddress
	ContractAddr SmartChainAddress
	Decimals     uint8
	BEP2Symbol   [32]byte
	MirrorFee    *big.Int
	ErrorCode    uint8
}

const (
	MirrorSyncErrCodeExpired      uint8 = 1
	MirrorSyncErrNotBoundByMirror uint8 = 2
	MirrorSyncErrInvalidSupply    uint8 = 3
)

type MirrorSyncSynPackage struct {
	SyncSender       SmartChainAddress
	ContractAddr     SmartChainAddress
	BEP2Symbol       [32]byte
	BEP20TotalSupply *big.Int
	SyncFee          *big.Int
	ExpireTime       uint64
}

func DeserializeMirrorSyncSynPackage(serializedPackage []byte) (*MirrorSyncSynPackage, sdk.Error) {
	var ms MirrorSyncSynPackage
	err := rlp.DecodeBytes(serializedPackage, &ms)
	if err != nil {
		return nil, ErrDeserializePackageFailed("deserialize mirror sync package failed")
	}
	return &ms, nil
}

type MirrorSyncAckPackage struct {
	SyncSender   SmartChainAddress
	ContractAddr SmartChainAddress
	SyncFee      *big.Int
	ErrorCode    uint8
}

func SymbolToBytes(symbol string) [32]byte {
	// length of bound token symbol length should not be larger than 32
	serializedBytes := [32]byte{}
	copy(serializedBytes[:], symbol)
	return serializedBytes
}

func BytesToSymbol(symbolBytes [32]byte) string {
	tokenSymbolBytes := make([]byte, 32, 32)
	copy(tokenSymbolBytes[:], symbolBytes[:])
	return string(bytes.Trim(tokenSymbolBytes, "\x00"))
}
