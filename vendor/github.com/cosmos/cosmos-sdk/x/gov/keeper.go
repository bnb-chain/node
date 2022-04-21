package gov

import (
	"fmt"
	"time"

	"github.com/tendermint/tendermint/crypto"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/params"
)

// Parameter store default namestore
const (
	DefaultParamSpace = "gov"
)

// Parameter store key
var (
	ParamStoreKeyDepositParams = []byte("depositparams")
	ParamStoreKeyTallyParams   = []byte("tallyparams")

	// Will hold deposit of both BC chain and side chain.
	DepositedCoinsAccAddr = sdk.AccAddress(crypto.AddressHash([]byte("BinanceChainDepositedCoins")))
)

// Type declaration for parameters
func ParamTypeTable() params.TypeTable {
	return params.NewTypeTable(
		ParamStoreKeyDepositParams, DepositParams{},
		ParamStoreKeyTallyParams, TallyParams{},
	)
}

type SideChainKeeper interface {
	PrepareCtxForSideChain(ctx sdk.Context, sideChainId string) (sdk.Context, error)
	GetAllSideChainPrefixes(ctx sdk.Context) ([]string, [][]byte)
}

// Governance Keeper
type Keeper struct {
	// The reference to the Param Keeper to get and set Global Params
	paramsKeeper params.Keeper

	// The reference to the Paramstore to get and set gov specific params
	paramSpace params.Subspace

	// The reference to the CoinKeeper to modify balances
	ck bank.Keeper

	// The ValidatorSet to get information about validators
	vs sdk.ValidatorSet

	// The reference to the DelegationSet to get information about delegators
	ds sdk.DelegationSet

	// The (unexposed) keys used to access the stores from the Context.
	storeKey sdk.StoreKey

	// The codec codec for binary encoding/decoding.
	cdc *codec.Codec

	// Hooks registered
	hooks map[ProposalKind][]GovHooks

	// Reserved codespace
	codespace sdk.CodespaceType

	// shared memory for block level state
	pool *sdk.Pool

	// if you want to enable side chains, you need call `SetupForSideChain`
	ScKeeper SideChainKeeper
}

// NewKeeper returns a governance keeper. It handles:
// - submitting governance proposals
// - depositing funds into proposals, and activating upon sufficient funds being deposited
// - users voting on proposals, with weight proportional to stake in the system
// - and tallying the result of the vote.
func NewKeeper(cdc *codec.Codec, key sdk.StoreKey, paramsKeeper params.Keeper, paramSpace params.Subspace, ck bank.Keeper, ds sdk.DelegationSet, codespace sdk.CodespaceType, pool *sdk.Pool) Keeper {
	return Keeper{
		storeKey:     key,
		paramsKeeper: paramsKeeper,
		paramSpace:   paramSpace.WithTypeTable(ParamTypeTable()),
		ck:           ck,
		ds:           ds,
		hooks:        make(map[ProposalKind][]GovHooks),
		vs:           ds.GetValidatorSet(),
		cdc:          cdc,
		codespace:    codespace,
		pool:         pool,
	}
}

func (keeper *Keeper) SetupForSideChain(scKeeper SideChainKeeper) {
	keeper.ScKeeper = scKeeper
}

// AddHooks add hooks for gov keeper
func (keeper Keeper) AddHooks(proposalType ProposalKind, hooks GovHooks) Keeper {
	hs := keeper.hooks[proposalType]
	if hs == nil {
		hs = make([]GovHooks, 0, 0)
	}
	hs = append(hs, hooks)
	keeper.hooks[proposalType] = hs
	return keeper
}

// =====================================================
// Proposals

// Creates a NewProposal
func (keeper Keeper) NewTextProposal(ctx sdk.Context, title string, description string, proposalType ProposalKind, votingPeriod time.Duration) Proposal {
	proposalID, err := keeper.getNewProposalID(ctx)
	if err != nil {
		return nil
	}
	var proposal Proposal = &TextProposal{
		ProposalID:   proposalID,
		Title:        title,
		Description:  description,
		ProposalType: proposalType,
		VotingPeriod: votingPeriod,
		Status:       StatusDepositPeriod,
		TallyResult:  EmptyTallyResult(),
		TotalDeposit: sdk.Coins{},
		SubmitTime:   ctx.BlockHeader().Time,
	}
	keeper.SetProposal(ctx, proposal)
	keeper.InactiveProposalQueuePush(ctx, proposal)
	return proposal
}

// Get Proposal from store by ProposalID
func (keeper Keeper) GetProposal(ctx sdk.Context, proposalID int64) Proposal {
	store := ctx.KVStore(keeper.storeKey)
	bz := store.Get(KeyProposal(proposalID))
	if bz == nil {
		return nil
	}

	var proposal Proposal
	keeper.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &proposal)

	return proposal
}

// Implements sdk.AccountKeeper.
func (keeper Keeper) SetProposal(ctx sdk.Context, proposal Proposal) {
	store := ctx.KVStore(keeper.storeKey)
	bz := keeper.cdc.MustMarshalBinaryLengthPrefixed(proposal)
	store.Set(KeyProposal(proposal.GetProposalID()), bz)
}

// Implements sdk.AccountKeeper.
func (keeper Keeper) DeleteProposal(ctx sdk.Context, proposal Proposal) {
	store := ctx.KVStore(keeper.storeKey)
	store.Delete(KeyProposal(proposal.GetProposalID()))
}

func (keeper Keeper) Iterate(ctx sdk.Context, voterAddr sdk.AccAddress, depositerAddr sdk.AccAddress, status ProposalStatus, numLatest int64, reverse bool, iter func(Proposal) bool) {

	maxProposalID, err := keeper.peekCurrentProposalID(ctx)
	if err != nil {
		return
	}

	if numLatest <= 0 {
		if reverse {
			numLatest = 0
		} else {
			numLatest = maxProposalID
		}
	}
	var initProposalID int64
	var step int64

	if reverse {
		initProposalID = maxProposalID - 1
		step = -1
	} else {
		initProposalID = maxProposalID - numLatest
		step = 1
	}
	for proposalID := initProposalID; (!reverse && proposalID < maxProposalID) || (reverse && proposalID > numLatest); proposalID += step {
		if voterAddr != nil && len(voterAddr) != 0 {
			_, found := keeper.GetVote(ctx, proposalID, voterAddr)
			if !found {
				continue
			}
		}

		if depositerAddr != nil && len(depositerAddr) != 0 {
			_, found := keeper.GetDeposit(ctx, proposalID, depositerAddr)
			if !found {
				continue
			}
		}

		proposal := keeper.GetProposal(ctx, proposalID)
		if proposal == nil {
			continue
		}

		if validProposalStatus(status) {
			if proposal.GetStatus() != status {
				continue
			}
		}
		stop := iter(proposal)
		if stop {
			break
		}

	}
	return

}

// Get Proposal from store by ProposalID
func (keeper Keeper) GetProposalsFiltered(ctx sdk.Context, voterAddr sdk.AccAddress, depositerAddr sdk.AccAddress, status ProposalStatus, numLatest int64) []Proposal {

	matchingProposals := []Proposal{}
	keeper.Iterate(ctx, voterAddr, depositerAddr, status, numLatest, false, func(proposal Proposal) bool {
		matchingProposals = append(matchingProposals, proposal)
		return false
	})

	return matchingProposals
}

func (keeper Keeper) SetInitialProposalID(ctx sdk.Context, proposalID int64) sdk.Error {
	store := ctx.KVStore(keeper.storeKey)
	bz := store.Get(KeyNextProposalID)
	if bz != nil {
		return ErrInvalidGenesis(keeper.codespace, "Initial ProposalID already set")
	}
	bz = keeper.cdc.MustMarshalBinaryLengthPrefixed(proposalID)
	store.Set(KeyNextProposalID, bz)
	return nil
}

// Get the last used proposal ID
func (keeper Keeper) GetLastProposalID(ctx sdk.Context) (proposalID int64) {
	proposalID, err := keeper.peekCurrentProposalID(ctx)
	if err != nil {
		return 0
	}
	proposalID--
	return
}

// Gets the next available ProposalID and increments it
func (keeper Keeper) getNewProposalID(ctx sdk.Context) (proposalID int64, err sdk.Error) {
	store := ctx.KVStore(keeper.storeKey)
	bz := store.Get(KeyNextProposalID)
	if bz == nil {
		return -1, ErrInvalidGenesis(keeper.codespace, "InitialProposalID never set")
	}
	keeper.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &proposalID)
	bz = keeper.cdc.MustMarshalBinaryLengthPrefixed(proposalID + 1)
	store.Set(KeyNextProposalID, bz)
	return proposalID, nil
}

// Peeks the next available ProposalID without incrementing it
func (keeper Keeper) peekCurrentProposalID(ctx sdk.Context) (proposalID int64, err sdk.Error) {
	store := ctx.KVStore(keeper.storeKey)
	bz := store.Get(KeyNextProposalID)
	if bz == nil {
		return -1, ErrInvalidGenesis(keeper.codespace, "InitialProposalID never set")
	}
	keeper.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &proposalID)
	return proposalID, nil
}

func (keeper Keeper) ActivateVotingPeriod(ctx sdk.Context, proposal Proposal) {
	proposal.SetVotingStartTime(ctx.BlockHeader().Time)
	proposal.SetStatus(StatusVotingPeriod)
	keeper.SetProposal(ctx, proposal)
	keeper.ActiveProposalQueuePush(ctx, proposal)
}

// =====================================================
// Params

// Returns the current Deposit Params from the global param store
// nolint: errcheck
func (keeper Keeper) GetDepositParams(ctx sdk.Context) DepositParams {
	var depositParams DepositParams
	keeper.paramSpace.Get(ctx, ParamStoreKeyDepositParams, &depositParams)
	return depositParams
}

// Returns the current Tally Params from the global param store
// nolint: errcheck
func (keeper Keeper) GetTallyParams(ctx sdk.Context) TallyParams {
	var tallyParams TallyParams
	keeper.paramSpace.Get(ctx, ParamStoreKeyTallyParams, &tallyParams)
	return tallyParams
}

// nolint: errcheck
func (keeper Keeper) SetDepositParams(ctx sdk.Context, depositParams DepositParams) {
	keeper.paramSpace.Set(ctx, ParamStoreKeyDepositParams, &depositParams)
}

// nolint: errcheck
func (keeper Keeper) SetTallyParams(ctx sdk.Context, tallyParams TallyParams) {
	keeper.paramSpace.Set(ctx, ParamStoreKeyTallyParams, &tallyParams)
}

// =====================================================
// Votes

// Adds a vote on a specific proposal
func (keeper Keeper) AddVote(ctx sdk.Context, proposalID int64, voterAddr sdk.AccAddress, option VoteOption) sdk.Error {
	proposal := keeper.GetProposal(ctx, proposalID)
	if proposal == nil {
		return ErrUnknownProposal(keeper.codespace, proposalID)
	}
	if proposal.GetStatus() != StatusVotingPeriod {
		return ErrInactiveProposal(keeper.codespace, proposalID)
	}

	if !validVoteOption(option) {
		return ErrInvalidVote(keeper.codespace, option)
	}

	vote := Vote{
		ProposalID: proposalID,
		Voter:      voterAddr,
		Option:     option,
	}
	keeper.setVote(ctx, proposalID, voterAddr, vote)

	return nil
}

// Gets the vote of a specific voter on a specific proposal
func (keeper Keeper) GetVote(ctx sdk.Context, proposalID int64, voterAddr sdk.AccAddress) (Vote, bool) {
	store := ctx.KVStore(keeper.storeKey)
	bz := store.Get(KeyVote(proposalID, voterAddr))
	if bz == nil {
		return Vote{}, false
	}
	var vote Vote
	keeper.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &vote)
	return vote, true
}

func (keeper Keeper) setVote(ctx sdk.Context, proposalID int64, voterAddr sdk.AccAddress, vote Vote) {
	store := ctx.KVStore(keeper.storeKey)
	bz := keeper.cdc.MustMarshalBinaryLengthPrefixed(vote)
	store.Set(KeyVote(proposalID, voterAddr), bz)
}

// Gets all the votes on a specific proposal
func (keeper Keeper) GetVotes(ctx sdk.Context, proposalID int64) sdk.Iterator {
	store := ctx.KVStore(keeper.storeKey)
	return sdk.KVStorePrefixIterator(store, KeyVotesSubspace(proposalID))
}

func (keeper Keeper) deleteVote(ctx sdk.Context, proposalID int64, voterAddr sdk.AccAddress) {
	store := ctx.KVStore(keeper.storeKey)
	store.Delete(KeyVote(proposalID, voterAddr))
}

// =====================================================
// Deposits

// Gets the deposit of a specific depositer on a specific proposal
func (keeper Keeper) GetDeposit(ctx sdk.Context, proposalID int64, depositerAddr sdk.AccAddress) (Deposit, bool) {
	store := ctx.KVStore(keeper.storeKey)
	bz := store.Get(KeyDeposit(proposalID, depositerAddr))
	if bz == nil {
		return Deposit{}, false
	}
	var deposit Deposit
	keeper.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &deposit)
	return deposit, true
}

func (keeper Keeper) setDeposit(ctx sdk.Context, proposalID int64, depositerAddr sdk.AccAddress, deposit Deposit) {
	store := ctx.KVStore(keeper.storeKey)
	bz := keeper.cdc.MustMarshalBinaryLengthPrefixed(deposit)
	store.Set(KeyDeposit(proposalID, depositerAddr), bz)
}

// Adds or updates a deposit of a specific depositer on a specific proposal
// Activates voting period when appropriate
func (keeper Keeper) AddDeposit(ctx sdk.Context, proposalID int64, depositerAddr sdk.AccAddress, depositAmount sdk.Coins) (sdk.Error, bool) {
	// Checks to see if proposal exists
	proposal := keeper.GetProposal(ctx, proposalID)
	if proposal == nil {
		return ErrUnknownProposal(keeper.codespace, proposalID), false
	}

	// Check if proposal is still depositable
	if (proposal.GetStatus() != StatusDepositPeriod) && (proposal.GetStatus() != StatusVotingPeriod) {
		return ErrAlreadyFinishedProposal(keeper.codespace, proposalID), false
	}

	// Send coins from depositor's account to DepositedCoinsAccAddr account
	_, err := keeper.ck.SendCoins(ctx, depositerAddr, DepositedCoinsAccAddr, depositAmount)
	if err != nil {
		return err, false
	}

	if ctx.IsDeliverTx() {
		keeper.pool.AddAddrs([]sdk.AccAddress{depositerAddr, DepositedCoinsAccAddr})
	}

	// Update Proposal
	proposal.SetTotalDeposit(proposal.GetTotalDeposit().Plus(depositAmount))
	keeper.SetProposal(ctx, proposal)

	// Check if deposit tipped proposal into voting period
	// Active voting period if so
	activatedVotingPeriod := false
	if proposal.GetStatus() == StatusDepositPeriod && proposal.GetTotalDeposit().IsGTE(keeper.GetDepositParams(ctx).MinDeposit) {
		keeper.ActivateVotingPeriod(ctx, proposal)
		activatedVotingPeriod = true
	}

	// Add or update deposit object
	currDeposit, found := keeper.GetDeposit(ctx, proposalID, depositerAddr)
	if !found {
		newDeposit := Deposit{depositerAddr, proposalID, depositAmount}
		keeper.setDeposit(ctx, proposalID, depositerAddr, newDeposit)
	} else {
		currDeposit.Amount = currDeposit.Amount.Plus(depositAmount)
		keeper.setDeposit(ctx, proposalID, depositerAddr, currDeposit)
	}

	return nil, activatedVotingPeriod
}

// Gets all the deposits on a specific proposal
func (keeper Keeper) GetDeposits(ctx sdk.Context, proposalID int64) sdk.Iterator {
	store := ctx.KVStore(keeper.storeKey)
	return sdk.KVStorePrefixIterator(store, KeyDepositsSubspace(proposalID))
}

// Returns and deletes all the deposits on a specific proposal
func (keeper Keeper) RefundDeposits(ctx sdk.Context, proposalID int64) {
	store := ctx.KVStore(keeper.storeKey)
	depositsIterator := keeper.GetDeposits(ctx, proposalID)
	defer depositsIterator.Close()
	for ; depositsIterator.Valid(); depositsIterator.Next() {
		deposit := &Deposit{}
		keeper.cdc.MustUnmarshalBinaryLengthPrefixed(depositsIterator.Value(), deposit)

		_, err := keeper.ck.SendCoins(ctx, DepositedCoinsAccAddr, deposit.Depositer, deposit.Amount)
		if err != nil {
			panic(fmt.Sprintf("refund error(%s) should not happen", err.Error()))
		}

		keeper.pool.AddAddrs([]sdk.AccAddress{deposit.Depositer, DepositedCoinsAccAddr})
		store.Delete(depositsIterator.Key())
	}
}

// DistributeDeposits distributes deposits to proposer
func (keeper Keeper) DistributeDeposits(ctx sdk.Context, proposalID int64) {
	proposerValAddr := ctx.BlockHeader().ProposerAddress
	proposerValidator := keeper.vs.ValidatorByConsAddr(ctx.DepriveSideChainKeyPrefix(), proposerValAddr)
	proposerAccAddr := proposerValidator.GetFeeAddr()

	store := ctx.KVStore(keeper.storeKey)
	depositsIterator := keeper.GetDeposits(ctx, proposalID)

	depositCoins := sdk.Coins{}
	for ; depositsIterator.Valid(); depositsIterator.Next() {
		deposit := &Deposit{}
		keeper.cdc.MustUnmarshalBinaryLengthPrefixed(depositsIterator.Value(), deposit)

		depositCoins = depositCoins.Plus(deposit.Amount)
		store.Delete(depositsIterator.Key())
	}
	depositsIterator.Close()

	if depositCoins.IsPositive() {
		ctx.Logger().Info("distribute empty deposits")
	}

	_, err := keeper.ck.SendCoins(ctx, DepositedCoinsAccAddr, proposerAccAddr, depositCoins)
	if err != nil {
		panic(fmt.Sprintf("distribute deposits error(%s) should not happen", err.Error()))
	}
	keeper.pool.AddAddrs([]sdk.AccAddress{sdk.AccAddress(proposerAccAddr), DepositedCoinsAccAddr})
}

// =====================================================
// ProposalQueues

func (keeper Keeper) getActiveProposalQueue(ctx sdk.Context) ProposalQueue {
	store := ctx.KVStore(keeper.storeKey)
	bz := store.Get(KeyActiveProposalQueue)
	if bz == nil {
		return nil
	}

	var proposalQueue ProposalQueue
	keeper.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &proposalQueue)

	return proposalQueue
}

func (keeper Keeper) setActiveProposalQueue(ctx sdk.Context, proposalQueue ProposalQueue) {
	store := ctx.KVStore(keeper.storeKey)
	bz := keeper.cdc.MustMarshalBinaryLengthPrefixed(proposalQueue)
	store.Set(KeyActiveProposalQueue, bz)
}

// Return the Proposal at the front of the ProposalQueue
func (keeper Keeper) ActiveProposalQueuePeek(ctx sdk.Context) Proposal {
	proposalQueue := keeper.getActiveProposalQueue(ctx)
	if len(proposalQueue) == 0 {
		return nil
	}
	return keeper.GetProposal(ctx, proposalQueue[0])
}

// Remove and return a Proposal from the front of the ProposalQueue
func (keeper Keeper) ActiveProposalQueuePop(ctx sdk.Context) Proposal {
	proposalQueue := keeper.getActiveProposalQueue(ctx)
	if len(proposalQueue) == 0 {
		return nil
	}
	frontElement, proposalQueue := proposalQueue[0], proposalQueue[1:]
	keeper.setActiveProposalQueue(ctx, proposalQueue)
	return keeper.GetProposal(ctx, frontElement)
}

// Add a proposalID to the ProposalQueue sorted by expire time
func (keeper Keeper) ActiveProposalQueuePush(ctx sdk.Context, proposal Proposal) {
	proposalQueue := keeper.getActiveProposalQueue(ctx)
	if len(proposalQueue) == 0 {
		proposalQueue = append(proposalQueue, proposal.GetProposalID())
	} else {
		votingExpireTime := proposal.GetVotingStartTime().Add(proposal.GetVotingPeriod())

		// sort proposal queue by expire time
		newProposalQueue := make(ProposalQueue, 0, len(proposalQueue)+1)
		for idx, proposalId := range proposalQueue {
			tmpProposal := keeper.GetProposal(ctx, proposalId)
			tmpVotingExpireTime := tmpProposal.GetVotingStartTime().Add(tmpProposal.GetVotingPeriod())
			if tmpVotingExpireTime.After(votingExpireTime) {
				newProposalQueue = append(newProposalQueue, proposal.GetProposalID())
				newProposalQueue = append(newProposalQueue, proposalQueue[idx:]...)
				break
			} else {
				newProposalQueue = append(newProposalQueue, proposalId)
			}
		}
		// insert proposal if there is no proposal in proposal queue which voting expire time after proposal
		if len(newProposalQueue) == len(proposalQueue) {
			newProposalQueue = append(newProposalQueue, proposal.GetProposalID())
		}

		proposalQueue = newProposalQueue
	}
	keeper.setActiveProposalQueue(ctx, proposalQueue)
}

func (keeper Keeper) getInactiveProposalQueue(ctx sdk.Context) ProposalQueue {
	store := ctx.KVStore(keeper.storeKey)
	bz := store.Get(KeyInactiveProposalQueue)
	if bz == nil {
		return nil
	}

	var proposalQueue ProposalQueue

	keeper.cdc.MustUnmarshalBinaryLengthPrefixed(bz, &proposalQueue)

	return proposalQueue
}

func (keeper Keeper) setInactiveProposalQueue(ctx sdk.Context, proposalQueue ProposalQueue) {
	store := ctx.KVStore(keeper.storeKey)
	bz := keeper.cdc.MustMarshalBinaryLengthPrefixed(proposalQueue)
	store.Set(KeyInactiveProposalQueue, bz)
}

// Return the Proposal at the front of the ProposalQueue
func (keeper Keeper) InactiveProposalQueuePeek(ctx sdk.Context) Proposal {
	proposalQueue := keeper.getInactiveProposalQueue(ctx)
	if len(proposalQueue) == 0 {
		return nil
	}
	return keeper.GetProposal(ctx, proposalQueue[0])
}

// Remove and return a Proposal from the front of the ProposalQueue
func (keeper Keeper) InactiveProposalQueuePop(ctx sdk.Context) Proposal {
	proposalQueue := keeper.getInactiveProposalQueue(ctx)
	if len(proposalQueue) == 0 {
		return nil
	}
	frontElement, proposalQueue := proposalQueue[0], proposalQueue[1:]
	keeper.setInactiveProposalQueue(ctx, proposalQueue)
	return keeper.GetProposal(ctx, frontElement)
}

// Add a proposalID to the back of the ProposalQueue
func (keeper Keeper) InactiveProposalQueuePush(ctx sdk.Context, proposal Proposal) {
	proposalQueue := append(keeper.getInactiveProposalQueue(ctx), proposal.GetProposalID())
	keeper.setInactiveProposalQueue(ctx, proposalQueue)
}
