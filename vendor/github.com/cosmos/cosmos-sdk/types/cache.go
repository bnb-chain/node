package types

type AccountStoreCache interface {
	GetAccount(addr AccAddress) Account
	SetAccount(addr AccAddress, acc Account)
	Delete(addr AccAddress)
	ClearCache() // only used by state sync to clear genesis status of accounts
}

type AccountCache interface {
	AccountStoreCache

	Cache() AccountCache
	Write()
}

type DummyAccountCache struct {
}

func (d *DummyAccountCache) GetAccount(addr AccAddress) Account {
	return nil
}

func (d *DummyAccountCache) SetAccount(addr AccAddress, acc Account) {
}

func (d *DummyAccountCache) Delete(addr AccAddress) {
}

func (d *DummyAccountCache) ClearCache() {
}

func (d *DummyAccountCache) Cache() AccountCache {
	return d
}

func (d *DummyAccountCache) Write() {
}
