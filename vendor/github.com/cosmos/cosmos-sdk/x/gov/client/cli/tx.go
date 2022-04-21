package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/utils"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authtxb "github.com/cosmos/cosmos-sdk/x/auth/client/txbuilder"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/cosmos/cosmos-sdk/x/sidechain/types"
)

const (
	flagProposalID        = "proposal-id"
	flagTitle             = "title"
	flagDescription       = "description"
	flagJustification     = "justification"
	flagProposalType      = "type"
	flagVotingPeriod      = "voting-period"
	flagDeposit           = "deposit"
	flagVoter             = "voter"
	flagOption            = "option"
	flagDepositer         = "depositer"
	flagStatus            = "status"
	flagLatestProposalIDs = "latest"
	flagProposal          = "proposal"
	flagBaseAsset         = "base-asset-symbol"
	flagQuoteAsset        = "quote-asset-symbol"
	flagInitPrice         = "init-price"
	flagExpireTime        = "expire-time"
	flagSideChainId       = "side-chain-id"
)

type proposal struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	VotingPeriod int64  `json:"voting_period"`
	Type         string `json:"type"`
	Deposit      string `json:"deposit"`
	SideChainId  string `json:"side_chain_id, omitempty"`
}

var proposalFlags = []string{
	flagTitle,
	flagDescription,
	flagVotingPeriod,
	flagProposalType,
	flagDeposit,
}

// GetCmdSubmitProposal implements submitting a proposal transaction command.
func GetCmdSubmitProposal(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-proposal",
		Short: "Submit a proposal along with an initial deposit",
		Long: strings.TrimSpace(`
Submit a proposal along with an initial deposit. Proposal title, description, type and deposit can be given directly or through a proposal JSON file. For example:

$ CLI gov submit-proposal --proposal="path/to/proposal.json"

where proposal.json contains:

{
  "title": "Test Proposal",
  "description": "My awesome proposal",
  "voting_period": 1000,
  "type": "Text",
  "deposit": "1000:test"
}

is equivalent to

$ CLI gov submit-proposal --title="Test Proposal" --description="My awesome proposal" --type="Text" --deposit="1000:test" --voting-period=1000
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			proposal, err := parseSubmitProposalFlags()
			if err != nil {
				return err
			}

			if proposal.Title == "" {
				return errors.New("Title should not be empty")
			}

			if len(proposal.Title) > gov.MaxTitleLength {
				return errors.New(fmt.Sprintf("Proposal title is longer than max length of %d", gov.MaxTitleLength))
			}

			if len(proposal.Description) > gov.MaxDescriptionLength {
				return errors.New(fmt.Sprintf("Proposal description is longer than max length of %d", gov.MaxDescriptionLength))
			}

			if proposal.SideChainId != gov.NativeChainID && len(proposal.SideChainId) > types.MaxSideChainIdLength {
				return errors.New(fmt.Sprintf("chain-id exceed the length limit %d", types.MaxSideChainIdLength))
			}

			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			if proposal.VotingPeriod <= 0 {
				return errors.New("voting period should be positive")
			}

			sideChainId := proposal.SideChainId
			votingPeriod := time.Duration(proposal.VotingPeriod) * time.Second
			if votingPeriod > gov.MaxVotingPeriod {
				return errors.New(fmt.Sprintf("voting period should be less than %d seconds", gov.MaxVotingPeriod/time.Second))
			}

			fromAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			amount, err := sdk.ParseCoins(proposal.Deposit)
			if err != nil {
				return err
			}

			proposalType, err := gov.ProposalTypeFromString(proposal.Type)
			if err != nil {
				return err
			}
			var msg sdk.Msg
			if sideChainId == gov.NativeChainID {
				msg = gov.NewMsgSubmitProposal(proposal.Title, proposal.Description, proposalType, fromAddr, amount, votingPeriod)
			} else {
				msg = gov.NewMsgSideChainSubmitProposal(proposal.Title, proposal.Description, proposalType, fromAddr, amount, votingPeriod, sideChainId)
			}
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			if cliCtx.GenerateOnly {
				return utils.PrintUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{msg})
			}

			// Build and sign the transaction, then broadcast to Tendermint
			// proposalID must be returned, and it is a part of response.
			cliCtx.PrintResponse = true
			return utils.CompleteAndBroadcastTxCli(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().String(flagTitle, "", "title of proposal")
	cmd.Flags().String(flagDescription, "", "description of proposal")
	cmd.Flags().Int64(flagVotingPeriod, 7*24*60*60, "voting period in seconds")
	cmd.Flags().String(flagProposalType, "", "proposalType of proposal, types: text/parameter_change/software_upgrade")
	cmd.Flags().String(flagDeposit, "", "deposit of proposal")
	cmd.Flags().String(flagProposal, "", "proposal file path (if this path is given, other proposal flags are ignored)")
	cmd.Flags().String(flagSideChainId, gov.NativeChainID, "the id of side chain, default is native chain")
	return cmd
}

func parseSubmitProposalFlags() (*proposal, error) {
	proposal := &proposal{}
	proposalFile := viper.GetString(flagProposal)

	if proposalFile == "" {
		proposal.Title = viper.GetString(flagTitle)
		proposal.Description = viper.GetString(flagDescription)
		proposal.VotingPeriod = viper.GetInt64(flagVotingPeriod)
		proposal.Type = client.NormalizeProposalType(viper.GetString(flagProposalType))
		proposal.Deposit = viper.GetString(flagDeposit)
		proposal.SideChainId = viper.GetString(flagSideChainId)
		return proposal, nil
	}

	for _, flag := range proposalFlags {
		if viper.GetString(flag) != "" {
			return nil, fmt.Errorf("--%s flag provided alongside --proposal, which is a noop", flag)
		}
	}

	contents, err := ioutil.ReadFile(proposalFile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(contents, proposal)
	if err != nil {
		return nil, err
	}

	return proposal, nil
}

// GetCmdDeposit implements depositing tokens for an active proposal.
func GetCmdDeposit(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit",
		Short: "Deposit tokens for activing proposal",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			depositerAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			proposalID := viper.GetInt64(flagProposalID)
			sideChainId := viper.GetString(flagSideChainId)
			if len(sideChainId) > types.MaxSideChainIdLength {
				return fmt.Errorf("side-chain-id exceed the max length %d", types.MaxSideChainIdLength)
			}

			amount, err := sdk.ParseCoins(viper.GetString(flagDeposit))
			if err != nil {
				return err
			}
			var msg sdk.Msg
			if sideChainId == gov.NativeChainID {
				msg = gov.NewMsgDeposit(depositerAddr, proposalID, amount)
			} else {
				msg = gov.NewMsgSideChainDeposit(depositerAddr, proposalID, amount, sideChainId)
			}

			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			if cliCtx.GenerateOnly {
				return utils.PrintUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{msg})
			}

			// Build and sign the transaction, then broadcast to a Tendermint
			// node.
			return utils.CompleteAndBroadcastTxCli(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().String(flagProposalID, "", "proposalID of proposal depositing on")
	cmd.Flags().String(flagDeposit, "", "amount of deposit")
	cmd.Flags().String(flagSideChainId, gov.NativeChainID, "the id of side chain, default is native chain")

	return cmd
}

// GetCmdVote implements creating a new vote command.
func GetCmdVote(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vote",
		Short: "Vote for an active proposal, options: yes/no/no_with_veto/abstain",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			voterAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			proposalID := viper.GetInt64(flagProposalID)
			option := viper.GetString(flagOption)
			sideChainId := viper.GetString(flagSideChainId)

			if len(sideChainId) > types.MaxSideChainIdLength {
				return fmt.Errorf("side-chain-id exceed the max length %d", types.MaxSideChainIdLength)
			}

			byteVoteOption, err := gov.VoteOptionFromString(client.NormalizeVoteOption(option))
			if err != nil {
				return err
			}
			var msg sdk.Msg
			if sideChainId == gov.NativeChainID {
				msg = gov.NewMsgVote(voterAddr, proposalID, byteVoteOption)
			} else {
				msg = gov.NewMsgSideChainVote(voterAddr, proposalID, byteVoteOption, sideChainId)
			}
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			if cliCtx.GenerateOnly {
				return utils.PrintUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{msg})
			}
			if sideChainId == gov.NativeChainID {
				fmt.Printf("Vote[Voter:%s,ProposalID:%d,Option:%s]",
					voterAddr.String(), proposalID, option,
				)
			} else {
				fmt.Printf("Vote[Voter:%s,ProposalID:%d,Option:%s, sideChainId:%s]",
					voterAddr.String(), proposalID, option, sideChainId,
				)
			}

			// Build and sign the transaction, then broadcast to a Tendermint
			// node.
			return utils.CompleteAndBroadcastTxCli(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().String(flagProposalID, "", "proposalID of proposal voting on")
	cmd.Flags().String(flagOption, "", "vote option {yes, no, no_with_veto, abstain}")
	cmd.Flags().String(flagSideChainId, gov.NativeChainID, "the id of side chain, default is native chain")

	return cmd
}

// GetCmdQueryProposal implements the query proposal command.
func GetCmdQueryProposal(queryRoute string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-proposal",
		Short: "Query details of a single proposal",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)
			proposalID := viper.GetInt64(flagProposalID)
			sideChainId := viper.GetString(flagSideChainId)

			params := gov.QueryProposalParams{
				BaseParams: gov.NewBaseParams(sideChainId),
				ProposalID: proposalID,
			}

			bz, err := cdc.MarshalJSON(params)
			if err != nil {
				return err
			}

			res, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/proposal", queryRoute), bz)
			if err != nil {
				return err
			}

			fmt.Println(string(res))
			return nil
		},
	}

	cmd.Flags().String(flagProposalID, "", "proposalID of proposal being queried")
	cmd.Flags().String(flagSideChainId, "", "the id of side chain, default is native chain")

	return cmd
}

// GetCmdQueryProposals implements a query proposals command.
func GetCmdQueryProposals(queryRoute string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-proposals",
		Short: "Query proposals with optional filters",
		RunE: func(cmd *cobra.Command, args []string) error {
			bechDepositerAddr := viper.GetString(flagDepositer)
			bechVoterAddr := viper.GetString(flagVoter)
			strProposalStatus := viper.GetString(flagStatus)
			latestProposalsIDs := viper.GetInt64(flagLatestProposalIDs)
			sideChainId := viper.GetString(flagSideChainId)

			params := gov.QueryProposalsParams{
				BaseParams:         gov.NewBaseParams(sideChainId),
				NumLatestProposals: latestProposalsIDs,
			}

			if len(bechDepositerAddr) != 0 {
				depositerAddr, err := sdk.AccAddressFromBech32(bechDepositerAddr)
				if err != nil {
					return err
				}
				params.Depositer = depositerAddr
			}

			if len(bechVoterAddr) != 0 {
				voterAddr, err := sdk.AccAddressFromBech32(bechVoterAddr)
				if err != nil {
					return err
				}
				params.Voter = voterAddr
			}

			if len(strProposalStatus) != 0 {
				proposalStatus, err := gov.ProposalStatusFromString(client.NormalizeProposalStatus(strProposalStatus))
				if err != nil {
					return err
				}
				params.ProposalStatus = proposalStatus
			}

			bz, err := cdc.MarshalJSON(params)
			if err != nil {
				return err
			}

			cliCtx := context.NewCLIContext().WithCodec(cdc)

			res, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/proposals", queryRoute), bz)
			if err != nil {
				return err
			}

			var matchingProposals []gov.Proposal
			err = cdc.UnmarshalJSON(res, &matchingProposals)
			if err != nil {
				return err
			}

			if len(matchingProposals) == 0 {
				fmt.Println("No matching proposals found")
				return nil
			}

			for _, proposal := range matchingProposals {
				fmt.Printf("  %d - %s\n", proposal.GetProposalID(), proposal.GetTitle())
			}

			return nil
		},
	}

	cmd.Flags().String(flagLatestProposalIDs, "", "(optional) limit to latest [number] proposals. Defaults to all proposals")
	cmd.Flags().String(flagDepositer, "", "(optional) filter by proposals deposited on by depositer")
	cmd.Flags().String(flagVoter, "", "(optional) filter by proposals voted on by voted")
	cmd.Flags().String(flagStatus, "", "(optional) filter proposals by proposal status, status: deposit_period/voting_period/passed/rejected")
	cmd.Flags().String(flagSideChainId, "", "the id of side chain, default is native chain")

	return cmd
}

// Command to Get a Proposal Information
// GetCmdQueryVote implements the query proposal vote command.
func GetCmdQueryVote(queryRoute string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-vote",
		Short: "Query details of a single vote",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)
			proposalID := viper.GetInt64(flagProposalID)
			sideChainId := viper.GetString(flagSideChainId)

			voterAddr, err := sdk.AccAddressFromBech32(viper.GetString(flagVoter))
			if err != nil {
				return err
			}

			params := gov.QueryVoteParams{
				BaseParams: gov.NewBaseParams(sideChainId),
				Voter:      voterAddr,
				ProposalID: proposalID,
			}
			bz, err := cdc.MarshalJSON(params)
			if err != nil {
				return err
			}

			res, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/vote", queryRoute), bz)
			if err != nil {
				return err
			}

			fmt.Println(string(res))
			return nil
		},
	}

	cmd.Flags().String(flagProposalID, "", "proposalID of proposal voting on")
	cmd.Flags().String(flagVoter, "", "bech32 voter address")
	cmd.Flags().String(flagSideChainId, "", "the id of side chain, default is native chain")

	return cmd
}

// GetCmdQueryVotes implements the command to query for proposal votes.
func GetCmdQueryVotes(queryRoute string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-votes",
		Short: "Query votes on a proposal",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)
			proposalID := viper.GetInt64(flagProposalID)
			sideChainId := viper.GetString(flagSideChainId)

			params := gov.QueryVotesParams{
				BaseParams: gov.NewBaseParams(sideChainId),
				ProposalID: proposalID,
			}
			bz, err := cdc.MarshalJSON(params)
			if err != nil {
				return err
			}

			res, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/votes", queryRoute), bz)
			if err != nil {
				return err
			}

			fmt.Println(string(res))
			return nil
		},
	}

	cmd.Flags().String(flagProposalID, "", "proposalID of which proposal's votes are being queried")
	cmd.Flags().String(flagSideChainId, "", "the id of side chain, default is native chain")

	return cmd
}

// Command to Get a specific Deposit Information
// GetCmdQueryDeposit implements the query proposal deposit command.
func GetCmdQueryDeposit(queryRoute string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-deposit",
		Short: "Query details of a deposit",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)
			proposalID := viper.GetInt64(flagProposalID)
			sideChainId := viper.GetString(flagSideChainId)

			depositerAddr, err := sdk.AccAddressFromBech32(viper.GetString(flagDepositer))
			if err != nil {
				return err
			}

			params := gov.QueryDepositParams{
				BaseParams: gov.NewBaseParams(sideChainId),
				Depositer:  depositerAddr,
				ProposalID: proposalID,
			}
			bz, err := cdc.MarshalJSON(params)
			if err != nil {
				return err
			}

			res, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/deposit", queryRoute), bz)
			if err != nil {
				return err
			}

			fmt.Println(string(res))
			return nil
		},
	}

	cmd.Flags().String(flagProposalID, "", "proposalID of proposal deposited on")
	cmd.Flags().String(flagDepositer, "", "bech32 depositer address")
	cmd.Flags().String(flagSideChainId, "", "the id of side chain, default is native chain")
	return cmd
}

// GetCmdQueryDeposits implements the command to query for proposal deposits.
func GetCmdQueryDeposits(queryRoute string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query-deposits",
		Short: "Query deposits on a proposal",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)
			proposalID := viper.GetInt64(flagProposalID)
			sideChainId := viper.GetString(flagSideChainId)

			params := gov.QueryDepositsParams{
				BaseParams: gov.NewBaseParams(sideChainId),
				ProposalID: proposalID,
			}
			bz, err := cdc.MarshalJSON(params)
			if err != nil {
				return err
			}

			res, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/deposits", queryRoute), bz)
			if err != nil {
				return err
			}

			fmt.Println(string(res))
			return nil
		},
	}

	cmd.Flags().String(flagProposalID, "", "proposalID of which proposal's deposits are being queried")
	cmd.Flags().String(flagSideChainId, "", "the id of side chain, default is native chain")

	return cmd
}

// GetCmdQueryDeposits implements the command to query for proposal deposits.
func GetCmdQueryTally(queryRoute string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tally",
		Short: "Get the tally of a proposal vote",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)
			proposalID := viper.GetInt64(flagProposalID)
			sideChainId := viper.GetString(flagSideChainId)

			params := gov.QueryTallyParams{
				BaseParams: gov.NewBaseParams(sideChainId),
				ProposalID: proposalID,
			}
			bz, err := cdc.MarshalJSON(params)
			if err != nil {
				return err
			}

			res, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/tally", queryRoute), bz)
			if err != nil {
				return err
			}

			fmt.Println(string(res))
			return nil
		},
	}

	cmd.Flags().String(flagProposalID, "", "proposalID of which proposal is being tallied")
	cmd.Flags().String(flagSideChainId, "", "the id of side chain, default is native chain")

	return cmd
}

// GetCmdSubmitListProposal implements submitting a proposal transaction command.
func GetCmdSubmitListProposal(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-list-proposal",
		Short: "Submit a list proposal along with an initial deposit",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			title := viper.GetString(flagTitle)
			description := viper.GetString(flagDescription)
			initialDeposit := viper.GetString(flagDeposit)
			tradeAsset := viper.GetString(flagBaseAsset)
			quoteAsset := viper.GetString(flagQuoteAsset)
			initPrice := viper.GetInt64(flagInitPrice)
			expireTimestamp := viper.GetInt64(flagExpireTime)
			votingPeriodInSeconds := viper.GetInt64(flagVotingPeriod)

			if title == "" {
				return errors.New("Title should not be empty")
			}

			if len(title) > gov.MaxTitleLength {
				return errors.New(fmt.Sprintf("Proposal title is longer than max length of %d", gov.MaxTitleLength))
			}

			if tradeAsset == "" {
				return errors.New("base asset should not be empty")
			}

			if quoteAsset == "" {
				return errors.New("quote asset should not be empty")
			}

			if initPrice <= 0 {
				return errors.New("init price should greater than 0")
			}

			expireTime := time.Unix(expireTimestamp, 0)
			if expireTime.Before(time.Now()) {
				return errors.New("expire time should after now")
			}

			if votingPeriodInSeconds <= 0 {
				return errors.New("voting period should be positive")
			}

			votingPeriod := time.Duration(votingPeriodInSeconds) * time.Second
			if votingPeriod > gov.MaxVotingPeriod {
				return errors.New(fmt.Sprintf("voting period should be less than %d seconds", gov.MaxVotingPeriod/time.Second))
			}

			fromAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			amount, err := sdk.ParseCoins(initialDeposit)
			if err != nil {
				return err
			}

			listParams := gov.ListTradingPairParams{
				BaseAssetSymbol:  tradeAsset,
				QuoteAssetSymbol: quoteAsset,
				InitPrice:        initPrice,
				Description:      description,
				ExpireTime:       expireTime,
			}

			listParamsBz, err := json.Marshal(listParams)
			if err != nil {
				return err
			}
			msg := gov.NewMsgSubmitProposal(title, string(listParamsBz), gov.ProposalTypeListTradingPair, fromAddr, amount, votingPeriod)

			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			if cliCtx.GenerateOnly {
				return utils.PrintUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{msg})
			}

			cliCtx.PrintResponse = true
			return utils.CompleteAndBroadcastTxCli(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().String(flagTitle, "", "title of proposal")
	cmd.Flags().String(flagDescription, "", "description of proposal")
	cmd.Flags().Int64(flagVotingPeriod, 7*24*60*60, "voting period in seconds")
	cmd.Flags().String(flagDeposit, "", "deposit of proposal")
	cmd.Flags().String(flagBaseAsset, "", "base asset symbol")
	cmd.Flags().String(flagQuoteAsset, "", "quote asset symbol")
	cmd.Flags().Int64(flagInitPrice, 0, "init price")
	cmd.Flags().Int64(flagExpireTime, 0, "expire time")

	return cmd
}

// GetCmdSubmitDelistProposal implements submitting a delist proposal transaction command.
func GetCmdSubmitDelistProposal(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-delist-proposal",
		Short: "Submit a delist proposal along with an initial deposit",
		RunE: func(cmd *cobra.Command, args []string) error {
			txBldr := authtxb.NewTxBuilderFromCLI().WithCodec(cdc)
			cliCtx := context.NewCLIContext().
				WithCodec(cdc).
				WithAccountDecoder(authcmd.GetAccountDecoder(cdc))

			title := viper.GetString(flagTitle)
			justification := viper.GetString(flagJustification)
			initialDeposit := viper.GetString(flagDeposit)
			baseAsset := viper.GetString(flagBaseAsset)
			quoteAsset := viper.GetString(flagQuoteAsset)
			votingPeriodInSeconds := viper.GetInt64(flagVotingPeriod)

			if title == "" {
				return errors.New("Title should not be empty")
			}

			if len(title) > gov.MaxTitleLength {
				return errors.New(fmt.Sprintf("Proposal title is longer than max length of %d", gov.MaxTitleLength))
			}

			if baseAsset == "" {
				return errors.New("base asset should not be empty")
			}

			if quoteAsset == "" {
				return errors.New("quote asset should not be empty")
			}

			if justification == "" {
				return errors.New("justification should not be empty")
			}

			if votingPeriodInSeconds <= 0 {
				return errors.New("voting period should be positive")
			}

			votingPeriod := time.Duration(votingPeriodInSeconds) * time.Second
			if votingPeriod > gov.MaxVotingPeriod {
				return fmt.Errorf("voting period should be less than %d seconds", gov.MaxVotingPeriod/time.Second)
			}

			fromAddr, err := cliCtx.GetFromAddress()
			if err != nil {
				return err
			}

			amount, err := sdk.ParseCoins(initialDeposit)
			if err != nil {
				return err
			}

			delistParams := gov.DelistTradingPairParams{
				BaseAssetSymbol:  baseAsset,
				QuoteAssetSymbol: quoteAsset,
				Justification:    justification,
				IsExecuted:       false,
			}

			delistParamsBz, err := json.Marshal(delistParams)
			if err != nil {
				return err
			}
			msg := gov.NewMsgSubmitProposal(title, string(delistParamsBz), gov.ProposalTypeDelistTradingPair, fromAddr, amount, votingPeriod)

			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			if cliCtx.GenerateOnly {
				return utils.PrintUnsignedStdTx(txBldr, cliCtx, []sdk.Msg{msg})
			}

			cliCtx.PrintResponse = true
			return utils.CompleteAndBroadcastTxCli(txBldr, cliCtx, []sdk.Msg{msg})
		},
	}

	cmd.Flags().String(flagTitle, "", "title of proposal")
	cmd.Flags().String(flagJustification, "", "justification of delist trading pair")
	cmd.Flags().Int64(flagVotingPeriod, 7*24*60*60, "voting period in seconds")
	cmd.Flags().String(flagDeposit, "", "deposit of proposal")
	cmd.Flags().String(flagBaseAsset, "", "base asset symbol")
	cmd.Flags().String(flagQuoteAsset, "", "quote asset symbol")

	return cmd
}
