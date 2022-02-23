package cmd

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/spf13/cobra"
	clientquery "github.com/strangelove-ventures/lens/client/query"
)

func bankSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send [from] [to] [amount]",
		Short: "send coins from one address to another",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := config.GetDefaultClient()
			fromAddr, err := cl.AccountFromKeyOrAddress(args[0])
			if err != nil {
				return err
			}

			toAddr, err := cl.DecodeBech32AccAddr(args[1])
			if err != nil {
				return err
			}

			coins, err := sdk.ParseCoinsNormalized(args[2])
			if err != nil {
				return fmt.Errorf("parsing coin string (i.e. 20000uatom): %s", err)
			}

			req := &banktypes.MsgSend{
				FromAddress: cl.MustEncodeAccAddr(fromAddr),
				ToAddress:   cl.MustEncodeAccAddr(toAddr),
				Amount:      coins,
			}

			res, err := cl.SendMsg(cmd.Context(), req)
			if err != nil {
				if res != nil {
					return fmt.Errorf("failed to send coins: code(%d) msg(%s)", res.Code, res.Logs)
				}
				return fmt.Errorf("failed to send coins: err(%w)", err)
			}
			return cl.PrintTxResponse(res)

		},
	}
	return cmd
}

// ========== Querier Functions ==========

func bankBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "balance [key-or-address]",
		Aliases: []string{"bal", "b"},
		Short:   "query the account balance for a key or address (if none is specified, the balance of the default account is returned)",
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := config.GetDefaultClient()
			_, err := ReadPageRequest(cmd.Flags())
			keyNameOrAddress := ""
			if len(args) == 0 {
				keyNameOrAddress = cl.Config.Key
			} else {
				keyNameOrAddress = args[0]
			}
			address, err := cl.AccountFromKeyOrAddress(keyNameOrAddress)
			if err != nil {
				return err
			}
			encodedAddr := cl.MustEncodeAccAddr(address)
			query := clientquery.Query{Client: cl, Options: clientquery.DefaultOptions()}
			balance, err := query.Balances(encodedAddr)
			if err != nil {
				return err
			}
			return cl.PrintObject(balance)
		},
	}
	return cmd
}

func bankTotalSupplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "total-supply",
		Aliases: []string{"totalsupply", "tot", "ts", "totsupplys"},
		Short:   "query the total supply of coins in the chain",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := config.GetDefaultClient()
			query := clientquery.Query{Client: cl, Options: clientquery.DefaultOptions()}
			totalSupply, err := query.TotalSupply()
			if err != nil {
				return err
			}
			return cl.PrintObject(totalSupply)
		},
	}
	return cmd
}

func bankDenomsMetadataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "denoms-metadata",
		Aliases: []string{"denoms", "d"},
		Short:   "query the denoms metadata",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := config.GetDefaultClient()
			query := clientquery.Query{Client: cl, Options: clientquery.DefaultOptions()}
			denoms, err := query.DenomsMetadata()
			if err != nil {
				return err
			}
			return cl.PrintObject(denoms)
		},
	}
	return cmd
}
