package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

/*
struct BindPackage {
	uint8   bindType         // 1  0:1
	bytes32 bep2TokenSymbol; // 32 1:33
	address contractAddr;    // 20 33:53
	uint256 totalSupply;     // 32 53:85
	uint256 peggyAmount;     // 32 85:117
	uint8   erc20Decimals;   // 1  117:118
	uint64  expireTime;      // 8  118:126
	uint256 relayReward;     // 32 126:158
}
*/

type BindType int8

const (
	BindTypeBind   BindType = 0
	BindTypeUnbind BindType = 1
)

type BindPackage struct {
	BindType        BindType
	Bep2TokenSymbol string
	ContractAddr    []byte
	TotalSupply     sdk.Int
	PeggyAmount     sdk.Int
	Decimals        int8
	ExpireTime      int64
	RelayReward     sdk.Int
}

func SerializeBindPackage(bindPackage *BindPackage) ([]byte, error) {
	serializedBytes := make([]byte, 1+32+20+32+32+1+8+32)
	if len(bindPackage.Bep2TokenSymbol) > 32 {
		return nil, fmt.Errorf("bep2 token symbol length should be no more than 32")
	}

	copy(serializedBytes[0:1], []byte{byte(bindPackage.BindType)})

	copy(serializedBytes[1:33], bindPackage.Bep2TokenSymbol)

	if len(bindPackage.ContractAddr) != 20 {
		return nil, fmt.Errorf("contract address length must be 20")
	}
	copy(serializedBytes[33:53], bindPackage.ContractAddr)
	if bindPackage.TotalSupply.BigInt().BitLen() > 255 ||
		bindPackage.PeggyAmount.BigInt().BitLen() > 255 ||
		bindPackage.RelayReward.BigInt().BitLen() > 255 {
		return nil, fmt.Errorf("overflow, maximum bits is 255")
	}

	length := len(bindPackage.TotalSupply.BigInt().Bytes())
	copy(serializedBytes[85-length:85], bindPackage.TotalSupply.BigInt().Bytes())

	length = len(bindPackage.PeggyAmount.BigInt().Bytes())
	copy(serializedBytes[117-length:117], bindPackage.PeggyAmount.BigInt().Bytes())

	copy(serializedBytes[117:118], []byte{byte(bindPackage.Decimals)})

	binary.BigEndian.PutUint64(serializedBytes[118:126], uint64(bindPackage.ExpireTime))

	length = len(bindPackage.RelayReward.BigInt().Bytes())
	copy(serializedBytes[158-length:158], bindPackage.RelayReward.BigInt().Bytes())

	return serializedBytes, nil
}

/*
struct TransferInRefundPackage {
	uint256 refundAmount;        // 32 0:32
	address contractAddr;        // 20 32:52
	address payable refundAddr;  // 20 52:72
	uint64  transferInSequence;  // 8  72:80
	uint16  refundReason         // 2  80:82
}
*/

type TransferInRefundPackage struct {
	RefundAmount       sdk.Int
	ContractAddr       []byte
	RefundAddr         []byte
	TransferInSequence int64
	RefundReason       RefundReason
}

func SerializeTransferInRefundPackage(refundPackage *TransferInRefundPackage) ([]byte, error) {
	serializedBytes := make([]byte, 32+20+20+8+2)
	if len(refundPackage.ContractAddr) != 20 || len(refundPackage.RefundAddr) != 20 {
		return nil, fmt.Errorf("length of address must be 20")
	}
	if refundPackage.RefundAmount.BigInt().BitLen() > 255 {
		return nil, fmt.Errorf("amount overflow, maximum bits is 256")
	}
	length := len(refundPackage.RefundAmount.BigInt().Bytes())
	copy(serializedBytes[32-length:32], refundPackage.RefundAmount.BigInt().Bytes())

	copy(serializedBytes[32:52], refundPackage.ContractAddr)
	copy(serializedBytes[52:72], refundPackage.RefundAddr)

	binary.BigEndian.PutUint64(serializedBytes[72:80], uint64(refundPackage.TransferInSequence))
	binary.BigEndian.PutUint16(serializedBytes[80:82], uint16(refundPackage.RefundReason))

	return serializedBytes, nil
}

/*
struct TransferOutPackage {
	bytes32 bep2TokenSymbol;    // 32 0:32
	address contractAddr;       // 20 32:52
	address refundAddr;         // 20 52:72
	address payable recipient;  // 20 72:92
	uint256 amount;             // 32 92:124
	uint64  expireTime;         // 8  124:132
	uint256 relayReward;        // 32 132:164
}
*/

type TransferOutPackage struct {
	Bep2TokenSymbol string
	ContractAddress []byte
	RefundAddress   []byte
	Recipient       []byte
	Amount          sdk.Int
	ExpireTime      int64
	RelayReward     sdk.Int
}

func SerializeTransferOutPackage(bep2TokenSymbol string, contractAddr []byte, refundAddr []byte, recipient []byte,
	amount sdk.Int, expireTime int64, relayReward sdk.Int) ([]byte, error) {
	serializedBytes := make([]byte, 32+20+20+20+32+8+32)
	if len(bep2TokenSymbol) > 32 {
		return nil, fmt.Errorf("bep2 token symbol length should be no more than 32")
	}
	copy(serializedBytes[0:32], bep2TokenSymbol)

	if len(contractAddr) != 20 || len(refundAddr) != 20 || len(recipient) != 20 {
		return nil, fmt.Errorf("length of address must be 20")
	}
	copy(serializedBytes[32:52], contractAddr)
	copy(serializedBytes[52:72], refundAddr)
	copy(serializedBytes[72:92], recipient)

	if amount.BigInt().BitLen() > 255 || relayReward.BigInt().BitLen() > 255 {
		return nil, fmt.Errorf("overflow, maximum bits is 255")
	}

	length := len(amount.BigInt().Bytes())
	copy(serializedBytes[124-length:124], amount.BigInt().Bytes())

	binary.BigEndian.PutUint64(serializedBytes[124:132], uint64(expireTime))

	length = len(relayReward.BigInt().Bytes())
	copy(serializedBytes[164-length:164], relayReward.BigInt().Bytes())

	return serializedBytes, nil
}

func DeserializeTransferOutPackage(serializedPackage []byte) (*TransferOutPackage, error) {
	packageLength := 164
	if len(serializedPackage) != packageLength {
		return nil, fmt.Errorf("package length should be %d", packageLength)
	}

	transferOutPackage := &TransferOutPackage{}

	bep2TokenSymbolBytes := make([]byte, 32, 32)
	copy(bep2TokenSymbolBytes[:], serializedPackage[0:32])
	transferOutPackage.Bep2TokenSymbol = string(bytes.Trim(bep2TokenSymbolBytes, "\x00"))

	contractAddressBytes := make([]byte, 20, 20)
	copy(contractAddressBytes[:], serializedPackage[32:52])
	transferOutPackage.ContractAddress = contractAddressBytes[:]

	refundAddressBytes := make([]byte, 20, 20)
	copy(refundAddressBytes[:], serializedPackage[52:72])
	transferOutPackage.RefundAddress = refundAddressBytes[:]

	recipientAddress := make([]byte, 20, 20)
	copy(recipientAddress[:], serializedPackage[72:92])
	transferOutPackage.Recipient = recipientAddress[:]

	transferOutPackage.Amount = sdk.NewIntFromBigInt(big.NewInt(0).SetBytes(serializedPackage[92:124]))
	transferOutPackage.ExpireTime = int64(binary.BigEndian.Uint64(serializedPackage[124:132]))
	transferOutPackage.RelayReward = sdk.NewIntFromBigInt(big.NewInt(0).SetBytes(serializedPackage[132:164]))

	return transferOutPackage, nil
}
