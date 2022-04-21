package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type PackageType uint8

const (
	StakePackageType PackageType = 0x00
	JailPackageType  PackageType = 0x01
)

type IbcValidator struct {
	ConsAddr []byte
	FeeAddr  []byte
	DistAddr sdk.AccAddress
	Power    uint64
}

type IbcValidatorSetPackage struct {
	Type         PackageType
	ValidatorSet []IbcValidator
}
