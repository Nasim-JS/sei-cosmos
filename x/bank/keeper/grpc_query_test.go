package keeper_test

import (
	gocontext "context"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

func (suite *IntegrationTestSuite) TestQueryBalance() {
	app, ctx, queryClient := suite.app, suite.ctx, suite.queryClient
	_, _, addr := testdata.KeyTestPubAddr()

	_, err := queryClient.Balance(gocontext.Background(), &types.QueryBalanceRequest{})
	suite.Require().Error(err)

	_, err = queryClient.Balance(gocontext.Background(), &types.QueryBalanceRequest{Address: addr.String()})
	suite.Require().Error(err)

	req := types.NewQueryBalanceRequest(addr, fooDenom)
	res, err := queryClient.Balance(gocontext.Background(), req)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.True(res.Balance.IsZero())

	origCoins := sdk.NewCoins(newFooCoin(50), newBarCoin(30))
	acc := app.AccountKeeper.NewAccountWithAddress(ctx, addr)

	app.AccountKeeper.SetAccount(ctx, acc)
	suite.Require().NoError(simapp.FundAccount(app.BankKeeper, ctx, acc.GetAddress(), origCoins))

	res, err = queryClient.Balance(gocontext.Background(), req)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.True(res.Balance.IsEqual(newFooCoin(50)))
}

func (suite *IntegrationTestSuite) TestQueryAllBalances() {
	app, ctx, queryClient := suite.app, suite.ctx, suite.queryClient
	_, _, addr := testdata.KeyTestPubAddr()
	_, err := queryClient.AllBalances(gocontext.Background(), &types.QueryAllBalancesRequest{})
	suite.Require().Error(err)

	pageReq := &query.PageRequest{
		Key:        nil,
		Limit:      1,
		CountTotal: false,
	}
	req := types.NewQueryAllBalancesRequest(addr, pageReq)
	res, err := queryClient.AllBalances(gocontext.Background(), req)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.True(res.Balances.IsZero())

	fooCoins := newFooCoin(50)
	barCoins := newBarCoin(30)

	origCoins := sdk.NewCoins(fooCoins, barCoins)
	acc := app.AccountKeeper.NewAccountWithAddress(ctx, addr)

	app.AccountKeeper.SetAccount(ctx, acc)
	suite.Require().NoError(simapp.FundAccount(app.BankKeeper, ctx, acc.GetAddress(), origCoins))

	res, err = queryClient.AllBalances(gocontext.Background(), req)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Equal(res.Balances.Len(), 1)
	suite.NotNil(res.Pagination.NextKey)

	suite.T().Log("query second page with nextkey")
	pageReq = &query.PageRequest{
		Key:        res.Pagination.NextKey,
		Limit:      1,
		CountTotal: true,
	}
	req = types.NewQueryAllBalancesRequest(addr, pageReq)
	res, err = queryClient.AllBalances(gocontext.Background(), req)
	suite.Equal(res.Balances.Len(), 1)
	suite.Nil(res.Pagination.NextKey)
}

func (suite *IntegrationTestSuite) TestSpendableBalances() {
	app, ctx, queryClient := suite.app, suite.ctx, suite.queryClient
	_, _, addr := testdata.KeyTestPubAddr()
	ctx = ctx.WithBlockTime(time.Now())

	_, err := queryClient.SpendableBalances(sdk.WrapSDKContext(ctx), &types.QuerySpendableBalancesRequest{})
	suite.Require().Error(err)

	pageReq := &query.PageRequest{
		Key:        nil,
		Limit:      2,
		CountTotal: false,
	}
	req := types.NewQuerySpendableBalancesRequest(addr, pageReq)

	res, err := queryClient.SpendableBalances(sdk.WrapSDKContext(ctx), req)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.True(res.Balances.IsZero())

	fooCoins := newFooCoin(50)
	barCoins := newBarCoin(30)

	origCoins := sdk.NewCoins(fooCoins, barCoins)
	acc := app.AccountKeeper.NewAccountWithAddress(ctx, addr)
	acc = vestingtypes.NewContinuousVestingAccount(
		acc.(*authtypes.BaseAccount),
		sdk.NewCoins(fooCoins),
		ctx.BlockTime().Unix(),
		ctx.BlockTime().Add(time.Hour).Unix(),
		nil,
	)

	app.AccountKeeper.SetAccount(ctx, acc)
	suite.Require().NoError(simapp.FundAccount(app.BankKeeper, ctx, acc.GetAddress(), origCoins))

	// move time forward for some tokens to vest
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(30 * time.Minute))
	queryHelper := baseapp.NewQueryServerTestHelper(ctx, app.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, app.BankKeeper)
	queryClient = types.NewQueryClient(queryHelper)

	res, err = queryClient.SpendableBalances(sdk.WrapSDKContext(ctx), req)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Equal(2, res.Balances.Len())
	suite.Nil(res.Pagination.NextKey)
	suite.EqualValues(30, res.Balances[0].Amount.Int64())
	suite.EqualValues(25, res.Balances[1].Amount.Int64())
}

func (suite *IntegrationTestSuite) TestQueryTotalSupply() {
	app, ctx, queryClient := suite.app, suite.ctx, suite.queryClient
	expectedTotalSupply := sdk.NewCoins(sdk.NewInt64Coin("test", 400000000))
	suite.
		Require().
		NoError(app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, expectedTotalSupply))

	res, err := queryClient.TotalSupply(gocontext.Background(), &types.QueryTotalSupplyRequest{})
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	suite.Require().Equal(expectedTotalSupply, res.Supply)
}

func (suite *IntegrationTestSuite) TestQueryTotalSupplyOf() {
	app, ctx, queryClient := suite.app, suite.ctx, suite.queryClient

	test1Supply := sdk.NewInt64Coin("test1", 4000000)
	test2Supply := sdk.NewInt64Coin("test2", 700000000)
	expectedTotalSupply := sdk.NewCoins(test1Supply, test2Supply)
	suite.
		Require().
		NoError(app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, expectedTotalSupply))

	_, err := queryClient.SupplyOf(gocontext.Background(), &types.QuerySupplyOfRequest{})
	suite.Require().Error(err)

	res, err := queryClient.SupplyOf(gocontext.Background(), &types.QuerySupplyOfRequest{Denom: test1Supply.Denom})
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	suite.Require().Equal(test1Supply, res.Amount)
}

func (suite *IntegrationTestSuite) TestQueryParams() {
	res, err := suite.queryClient.Params(gocontext.Background(), &types.QueryParamsRequest{})
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Equal(suite.app.BankKeeper.GetParams(suite.ctx), res.GetParams())
}

func (suite *IntegrationTestSuite) TestQueryDenomsMetadataRequest() {
	var (
		req         *types.QueryDenomsMetadataRequest
		expMetadata []types.Metadata
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty pagination",
			func() {
				req = &types.QueryDenomsMetadataRequest{}
			},
			true,
		},
		{
			"success, no results",
			func() {
				req = &types.QueryDenomsMetadataRequest{
					Pagination: &query.PageRequest{
						Limit:      3,
						CountTotal: true,
					},
				}
			},
			true,
		},
		{
			"success",
			func() {
				metadataAtom := types.Metadata{
					Description: "The native staking token of the Cosmos Hub.",
					DenomUnits: []*types.DenomUnit{
						{
							Denom:    "uatom",
							Exponent: 0,
							Aliases:  []string{"microatom"},
						},
						{
							Denom:    "atom",
							Exponent: 6,
							Aliases:  []string{"ATOM"},
						},
					},
					Base:    "uatom",
					Display: "atom",
				}

				metadataEth := types.Metadata{
					Description: "Ethereum native token",
					DenomUnits: []*types.DenomUnit{
						{
							Denom:    "wei",
							Exponent: 0,
						},
						{
							Denom:    "eth",
							Exponent: 18,
							Aliases:  []string{"ETH", "ether"},
						},
					},
					Base:    "wei",
					Display: "eth",
				}

				suite.app.BankKeeper.SetDenomMetaData(suite.ctx, metadataAtom)
				suite.app.BankKeeper.SetDenomMetaData(suite.ctx, metadataEth)
				expMetadata = []types.Metadata{metadataAtom, metadataEth}
				req = &types.QueryDenomsMetadataRequest{
					Pagination: &query.PageRequest{
						Limit:      7,
						CountTotal: true,
					},
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(suite.ctx)

			res, err := suite.queryClient.DenomsMetadata(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expMetadata, res.Metadatas)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestQueryDenomMetadataRequest() {
	var (
		req         *types.QueryDenomMetadataRequest
		expMetadata = types.Metadata{}
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty denom",
			func() {
				req = &types.QueryDenomMetadataRequest{}
			},
			false,
		},
		{
			"not found denom",
			func() {
				req = &types.QueryDenomMetadataRequest{
					Denom: "foo",
				}
			},
			false,
		},
		{
			"success",
			func() {
				expMetadata = types.Metadata{
					Description: "The native staking token of the Cosmos Hub.",
					DenomUnits: []*types.DenomUnit{
						{
							Denom:    "uatom",
							Exponent: 0,
							Aliases:  []string{"microatom"},
						},
						{
							Denom:    "atom",
							Exponent: 6,
							Aliases:  []string{"ATOM"},
						},
					},
					Base:    "uatom",
					Display: "atom",
				}

				suite.app.BankKeeper.SetDenomMetaData(suite.ctx, expMetadata)
				req = &types.QueryDenomMetadataRequest{
					Denom: expMetadata.Base,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(suite.ctx)

			res, err := suite.queryClient.DenomMetadata(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expMetadata, res.Metadata)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestQueryTokenFactoryDenomMetadataRequest() {
	var (
		req         *types.QueryTokenFactoryDenomMetadataRequest
		expMetadata = types.Metadata{}
	)
	tokenFactoryDenom := "factory/sei1gxskuzvhr4s8sdm2rpruaf7yx2dnmjn0zfdu9q/NEWCOIN"
	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"empty denom",
			func() {
				req = &types.QueryTokenFactoryDenomMetadataRequest{}
			},
			false,
		},
		{
			"not found denom",
			func() {
				req = &types.QueryTokenFactoryDenomMetadataRequest{
					Denom: tokenFactoryDenom,
				}
			},
			false,
		},
		{
			"success",
			func() {

				expMetadata = types.Metadata{
					Description: "Token factory custom token",
					DenomUnits: []*types.DenomUnit{
						{
							Denom:    tokenFactoryDenom,
							Exponent: 0,
							Aliases:  []string{tokenFactoryDenom},
						},
					},
					Base:    tokenFactoryDenom,
					Display: tokenFactoryDenom,
				}

				suite.app.BankKeeper.SetDenomMetaData(suite.ctx, expMetadata)
				req = &types.QueryTokenFactoryDenomMetadataRequest{
					Denom: expMetadata.Base,
				}
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate()
			ctx := sdk.WrapSDKContext(suite.ctx)

			res, err := suite.queryClient.TokenFactoryDenomMetadata(ctx, req)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(expMetadata, res.Metadata)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
