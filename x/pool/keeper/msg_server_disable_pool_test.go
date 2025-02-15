package keeper_test

import (
	i "github.com/KYVENetwork/chain/testutil/integration"
	bundletypes "github.com/KYVENetwork/chain/x/bundles/types"
	globalTypes "github.com/KYVENetwork/chain/x/global/types"
	stakertypes "github.com/KYVENetwork/chain/x/stakers/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	// Gov
	govV1Types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	// Pool
	"github.com/KYVENetwork/chain/x/pool/types"
)

/*

TEST CASES - msg_server_disabled_pool.go

* Invalid authority (transaction)
* Invalid authority (proposal)
* Disable a non-existing pool
* Disable pool which is active
* Disable pool which is active and has a balance
* Disable pool which is already disabled
* Disable multiple pools
* Kick out all stakers from pool
* Kick out all stakers from pool which are still members of another pool
* Drop current bundle proposal when pool gets disabled

*/

var _ = Describe("msg_server_disable_pool.go", Ordered, func() {
	s := i.NewCleanChain()

	gov := s.App().GovKeeper.GetGovernanceAccount(s.Ctx()).GetAddress().String()
	votingPeriod := s.App().GovKeeper.GetVotingParams(s.Ctx()).VotingPeriod

	BeforeEach(func() {
		s = i.NewCleanChain()

		// create clean pool for every test case
		s.App().PoolKeeper.AppendPool(s.Ctx(), types.Pool{
			Name:           "PoolTest",
			MaxBundleSize:  100,
			StartKey:       "0",
			UploadInterval: 60,
			OperatingCost:  10_000,
			Protocol: &types.Protocol{
				Version:     "0.0.0",
				Binaries:    "{}",
				LastUpgrade: uint64(s.Ctx().BlockTime().Unix()),
			},
			UpgradePlan: &types.UpgradePlan{},
		})

		s.RunTxPoolSuccess(&types.MsgFundPool{
			Creator: i.ALICE,
			Id:      0,
			Amount:  100 * i.KYVE,
		})

		s.App().PoolKeeper.AppendPool(s.Ctx(), types.Pool{
			Name:           "PoolTest2",
			MaxBundleSize:  100,
			StartKey:       "0",
			UploadInterval: 60,
			OperatingCost:  10_000,
			Protocol: &types.Protocol{
				Version:     "0.0.0",
				Binaries:    "{}",
				LastUpgrade: uint64(s.Ctx().BlockTime().Unix()),
			},
			UpgradePlan: &types.UpgradePlan{},
		})

		s.RunTxPoolSuccess(&types.MsgFundPool{
			Creator: i.ALICE,
			Id:      1,
			Amount:  100 * i.KYVE,
		})
	})

	AfterEach(func() {
		s.PerformValidityChecks()
	})

	It("Invalid authority (transaction).", func() {
		// ARRANGE
		msg := &types.MsgDisablePool{
			Authority: i.DUMMY[0],
			Id:        0,
		}

		// ACT
		_, err := s.RunTx(msg)

		// ASSERT
		Expect(err).To(HaveOccurred())

		pool, _ := s.App().PoolKeeper.GetPool(s.Ctx(), 0)
		Expect(pool.Disabled).To(BeFalse())

		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.StorageId).To(BeEmpty())
	})

	It("Invalid authority (proposal).", func() {
		// ARRANGE
		msg := &types.MsgDisablePool{
			Authority: i.DUMMY[0],
			Id:        0,
		}

		proposal, _ := BuildGovernanceTxs(s, []sdk.Msg{msg})

		// ACT
		_, err := s.RunTx(&proposal)

		// ASSERT
		Expect(err).To(HaveOccurred())

		pool, _ := s.App().PoolKeeper.GetPool(s.Ctx(), 0)
		Expect(pool.Disabled).To(BeFalse())

		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.StorageId).To(BeEmpty())
	})

	It("Disable a non-existing pool", func() {
		// ARRANGE
		msg := &types.MsgDisablePool{
			Authority: gov,
			Id:        42,
		}

		p, v := BuildGovernanceTxs(s, []sdk.Msg{msg})

		// ACT
		_, submitErr := s.RunTx(&p)
		_, voteErr := s.RunTx(&v)

		s.CommitAfter(*votingPeriod)
		s.Commit()

		// ASSERT
		proposal, _ := s.App().GovKeeper.GetProposal(s.Ctx(), 1)

		Expect(submitErr).To(Not(HaveOccurred()))
		Expect(voteErr).To(Not(HaveOccurred()))

		Expect(proposal.Status).To(Equal(govV1Types.StatusFailed))
	})

	It("Disable pool which is active", func() {
		// ARRANGE
		msg := &types.MsgDisablePool{
			Authority: gov,
			Id:        0,
		}

		p, v := BuildGovernanceTxs(s, []sdk.Msg{msg})

		// ACT
		_, submitErr := s.RunTx(&p)
		_, voteErr := s.RunTx(&v)

		s.CommitAfter(*votingPeriod)
		s.Commit()

		// ASSERT
		proposal, _ := s.App().GovKeeper.GetProposal(s.Ctx(), 1)
		pool, _ := s.App().PoolKeeper.GetPool(s.Ctx(), 0)

		Expect(submitErr).To(Not(HaveOccurred()))
		Expect(voteErr).To(Not(HaveOccurred()))

		Expect(proposal.Status).To(Equal(govV1Types.StatusPassed))
		Expect(pool.Disabled).To(BeTrue())

		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.StorageId).To(BeEmpty())

		// assert empty pool balance
		b := s.App().BankKeeper.GetBalance(s.Ctx(), pool.GetPoolAccount(), globalTypes.Denom).Amount.Uint64()
		Expect(b).To(BeZero())
	})

	It("Disable pool which is active and has a balance", func() {
		// ARRANGE
		pool, _ := s.App().PoolKeeper.GetPool(s.Ctx(), 0)

		s.App().PoolKeeper.SetParams(s.Ctx(), types.Params{
			ProtocolInflationShare:  sdk.MustNewDecFromStr("0.1"),
			PoolInflationPayoutRate: sdk.MustNewDecFromStr("0.05"),
		})

		for i := 0; i < 100; i++ {
			s.Commit()
		}

		b := s.App().BankKeeper.GetBalance(s.Ctx(), pool.GetPoolAccount(), globalTypes.Denom).Amount.Uint64()
		Expect(b).To(BeNumerically(">", uint64(0)))

		msg := &types.MsgDisablePool{
			Authority: gov,
			Id:        0,
		}

		p, v := BuildGovernanceTxs(s, []sdk.Msg{msg})

		// ACT
		_, submitErr := s.RunTx(&p)
		_, voteErr := s.RunTx(&v)

		s.CommitAfter(*votingPeriod)
		s.Commit()

		// ASSERT
		proposal, _ := s.App().GovKeeper.GetProposal(s.Ctx(), 1)
		pool, _ = s.App().PoolKeeper.GetPool(s.Ctx(), 0)

		Expect(submitErr).To(Not(HaveOccurred()))
		Expect(voteErr).To(Not(HaveOccurred()))

		Expect(proposal.Status).To(Equal(govV1Types.StatusPassed))
		Expect(pool.Disabled).To(BeTrue())

		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.StorageId).To(BeEmpty())

		// assert empty pool balance
		b = s.App().BankKeeper.GetBalance(s.Ctx(), pool.GetPoolAccount(), globalTypes.Denom).Amount.Uint64()
		Expect(b).To(BeZero())
	})

	It("Disable pool which is active", func() {
		// ARRANGE
		msg := &types.MsgDisablePool{
			Authority: gov,
			Id:        0,
		}

		p, v := BuildGovernanceTxs(s, []sdk.Msg{msg})

		// ACT
		_, submitErr := s.RunTx(&p)
		_, voteErr := s.RunTx(&v)

		s.CommitAfter(*votingPeriod)
		s.Commit()

		// ASSERT
		proposal, _ := s.App().GovKeeper.GetProposal(s.Ctx(), 1)
		pool, _ := s.App().PoolKeeper.GetPool(s.Ctx(), 0)

		Expect(submitErr).To(Not(HaveOccurred()))
		Expect(voteErr).To(Not(HaveOccurred()))

		Expect(proposal.Status).To(Equal(govV1Types.StatusPassed))
		Expect(pool.Disabled).To(BeTrue())

		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.StorageId).To(BeEmpty())
	})

	It("Disable pool which is already disabled", func() {
		// ARRANGE
		pool, _ := s.App().PoolKeeper.GetPool(s.Ctx(), 0)
		pool.Disabled = true
		s.App().PoolKeeper.SetPool(s.Ctx(), pool)

		msg := &types.MsgDisablePool{
			Authority: gov,
			Id:        0,
		}

		p, v := BuildGovernanceTxs(s, []sdk.Msg{msg})

		// ACT
		_, submitErr := s.RunTx(&p)
		_, voteErr := s.RunTx(&v)

		s.CommitAfter(*votingPeriod)
		s.Commit()

		// ASSERT
		proposal, _ := s.App().GovKeeper.GetProposal(s.Ctx(), 1)
		pool, _ = s.App().PoolKeeper.GetPool(s.Ctx(), 0)

		Expect(submitErr).To(Not(HaveOccurred()))
		Expect(voteErr).To(Not(HaveOccurred()))

		Expect(proposal.Status).To(Equal(govV1Types.StatusFailed))
		Expect(pool.Disabled).To(BeTrue())

		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.StorageId).To(BeEmpty())
	})

	It("Disable multiple pools", func() {
		// ARRANGE
		msgFirstPool := &types.MsgDisablePool{
			Authority: gov,
			Id:        0,
		}
		msgSecondPool := &types.MsgDisablePool{
			Authority: gov,
			Id:        1,
		}

		p, v := BuildGovernanceTxs(s, []sdk.Msg{msgFirstPool, msgSecondPool})

		// ACT
		_, submitErr := s.RunTx(&p)
		_, voteErr := s.RunTx(&v)

		s.CommitAfter(*votingPeriod)
		s.Commit()

		// ASSERT
		proposal, _ := s.App().GovKeeper.GetProposal(s.Ctx(), 1)
		firstPool, _ := s.App().PoolKeeper.GetPool(s.Ctx(), 0)
		secondPool, _ := s.App().PoolKeeper.GetPool(s.Ctx(), 1)

		Expect(submitErr).To(Not(HaveOccurred()))
		Expect(voteErr).To(Not(HaveOccurred()))

		Expect(proposal.Status).To(Equal(govV1Types.StatusPassed))
		Expect(firstPool.Disabled).To(BeTrue())
		Expect(secondPool.Disabled).To(BeTrue())

		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.StorageId).To(BeEmpty())

		bundleProposal, _ = s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 1)
		Expect(bundleProposal.StorageId).To(BeEmpty())
	})

	It("Kick out all stakers from pool", func() {
		// ARRANGE
		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_0,
			Amount:  100 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_0,
			PoolId:     0,
			Valaddress: i.VALADDRESS_0,
			Amount:     0,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  100 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_1,
			PoolId:     0,
			Valaddress: i.VALADDRESS_1,
			Amount:     0,
		})

		msgFirstPool := &types.MsgDisablePool{
			Authority: gov,
			Id:        0,
		}

		Expect(s.App().StakersKeeper.GetAllValaccounts(s.Ctx())).To(HaveLen(2))
		Expect(s.App().StakersKeeper.GetActiveStakers(s.Ctx())).To(HaveLen(2))

		p, v := BuildGovernanceTxs(s, []sdk.Msg{msgFirstPool})

		msgVoteStaker0 := govV1Types.NewMsgVote(sdk.MustAccAddressFromBech32(i.STAKER_0), 1, govV1Types.VoteOption_VOTE_OPTION_YES, "")
		msgVoteStaker1 := govV1Types.NewMsgVote(sdk.MustAccAddressFromBech32(i.STAKER_0), 1, govV1Types.VoteOption_VOTE_OPTION_YES, "")

		// ACT
		_, submitErr := s.RunTx(&p)
		_, voteErr := s.RunTx(&v)
		_, voteErr0 := s.RunTx(msgVoteStaker0)
		_, voteErr1 := s.RunTx(msgVoteStaker1)

		s.CommitAfter(*votingPeriod)
		s.Commit()

		// ASSERT
		proposal, _ := s.App().GovKeeper.GetProposal(s.Ctx(), 1)

		Expect(s.App().StakersKeeper.GetAllValaccounts(s.Ctx())).To(HaveLen(0))
		Expect(s.App().StakersKeeper.GetActiveStakers(s.Ctx())).To(HaveLen(0))

		firstPool, _ := s.App().PoolKeeper.GetPool(s.Ctx(), 0)

		Expect(submitErr).To(Not(HaveOccurred()))
		Expect(voteErr).To(Not(HaveOccurred()))
		Expect(voteErr0).To(Not(HaveOccurred()))
		Expect(voteErr1).To(Not(HaveOccurred()))

		Expect(proposal.Status).To(Equal(govV1Types.StatusPassed))
		Expect(firstPool.Disabled).To(BeTrue())

		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.StorageId).To(BeEmpty())
	})

	It("Kick out all stakers from pool which are still members of another pool", func() {
		// ARRANGE
		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_0,
			Amount:  100 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_0,
			PoolId:     0,
			Valaddress: i.VALADDRESS_0,
			Amount:     0,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_0,
			PoolId:     1,
			Valaddress: i.VALADDRESS_2,
			Amount:     0,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  100 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_1,
			PoolId:     0,
			Valaddress: i.VALADDRESS_1,
			Amount:     0,
		})

		msgFirstPool := &types.MsgDisablePool{
			Authority: gov,
			Id:        0,
		}

		Expect(s.App().StakersKeeper.GetAllValaccounts(s.Ctx())).To(HaveLen(3))
		Expect(s.App().StakersKeeper.GetActiveStakers(s.Ctx())).To(HaveLen(2))

		p, v := BuildGovernanceTxs(s, []sdk.Msg{msgFirstPool})

		msgVoteStaker0 := govV1Types.NewMsgVote(sdk.MustAccAddressFromBech32(i.STAKER_0), 1, govV1Types.VoteOption_VOTE_OPTION_YES, "")
		msgVoteStaker1 := govV1Types.NewMsgVote(sdk.MustAccAddressFromBech32(i.STAKER_0), 1, govV1Types.VoteOption_VOTE_OPTION_YES, "")

		// ACT
		_, submitErr := s.RunTx(&p)
		_, voteErr := s.RunTx(&v)
		_, voteErr0 := s.RunTx(msgVoteStaker0)
		_, voteErr1 := s.RunTx(msgVoteStaker1)

		s.CommitAfter(*votingPeriod)
		s.Commit()

		// ASSERT
		proposal, _ := s.App().GovKeeper.GetProposal(s.Ctx(), 1)

		Expect(s.App().StakersKeeper.GetAllValaccounts(s.Ctx())).To(HaveLen(1))
		Expect(s.App().StakersKeeper.GetActiveStakers(s.Ctx())).To(HaveLen(1))

		firstPool, _ := s.App().PoolKeeper.GetPool(s.Ctx(), 0)

		Expect(submitErr).To(Not(HaveOccurred()))
		Expect(voteErr).To(Not(HaveOccurred()))
		Expect(voteErr0).To(Not(HaveOccurred()))
		Expect(voteErr1).To(Not(HaveOccurred()))

		Expect(proposal.Status).To(Equal(govV1Types.StatusPassed))
		Expect(firstPool.Disabled).To(BeTrue())

		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.StorageId).To(BeEmpty())
	})

	It("Drop current bundle proposal when pool gets disabled", func() {
		// ARRANGE
		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_0,
			Amount:  100 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_0,
			PoolId:     0,
			Valaddress: i.VALADDRESS_0,
			Amount:     0,
		})

		s.RunTxBundlesSuccess(&bundletypes.MsgClaimUploaderRole{
			Creator: i.VALADDRESS_0,
			Staker:  i.STAKER_0,
			PoolId:  0,
		})

		s.CommitAfterSeconds(60)

		s.RunTxBundlesSuccess(&bundletypes.MsgSubmitBundleProposal{
			Creator:       i.VALADDRESS_0,
			Staker:        i.STAKER_0,
			PoolId:        0,
			StorageId:     "y62A3tfbSNcNYDGoL-eXwzyV-Zc9Q0OVtDvR1biJmNI",
			DataSize:      100,
			DataHash:      "test_hash",
			FromIndex:     0,
			BundleSize:    100,
			FromKey:       "0",
			ToKey:         "99",
			BundleSummary: "test_value",
		})

		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.StorageId).To(Equal("y62A3tfbSNcNYDGoL-eXwzyV-Zc9Q0OVtDvR1biJmNI"))

		msgFirstPool := &types.MsgDisablePool{
			Authority: gov,
			Id:        0,
		}

		p, v := BuildGovernanceTxs(s, []sdk.Msg{msgFirstPool})

		msgVoteStaker0 := govV1Types.NewMsgVote(sdk.MustAccAddressFromBech32(i.STAKER_0), 1, govV1Types.VoteOption_VOTE_OPTION_YES, "")
		msgVoteStaker1 := govV1Types.NewMsgVote(sdk.MustAccAddressFromBech32(i.STAKER_0), 1, govV1Types.VoteOption_VOTE_OPTION_YES, "")

		// ACT
		_, submitErr := s.RunTx(&p)
		_, voteErr := s.RunTx(&v)
		_, voteErr0 := s.RunTx(msgVoteStaker0)
		_, voteErr1 := s.RunTx(msgVoteStaker1)

		s.CommitAfter(*votingPeriod)
		s.Commit()

		// ASSERT
		proposal, _ := s.App().GovKeeper.GetProposal(s.Ctx(), 1)

		pool, _ := s.App().PoolKeeper.GetPool(s.Ctx(), 0)

		Expect(submitErr).To(Not(HaveOccurred()))
		Expect(voteErr).To(Not(HaveOccurred()))
		Expect(voteErr0).To(Not(HaveOccurred()))
		Expect(voteErr1).To(Not(HaveOccurred()))

		Expect(proposal.Status).To(Equal(govV1Types.StatusPassed))
		Expect(pool.Disabled).To(BeTrue())

		// check if bundle proposal got dropped
		bundleProposal, bundleProposalFound := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposalFound).To(BeTrue())

		Expect(bundleProposal.PoolId).To(Equal(uint64(0)))
		Expect(bundleProposal.StorageId).To(BeEmpty())
		Expect(bundleProposal.Uploader).To(BeEmpty())
		Expect(bundleProposal.NextUploader).To(BeEmpty())
		Expect(bundleProposal.DataSize).To(BeZero())
		Expect(bundleProposal.DataHash).To(BeEmpty())
		Expect(bundleProposal.BundleSize).To(BeZero())
		Expect(bundleProposal.FromKey).To(BeEmpty())
		Expect(bundleProposal.ToKey).To(BeEmpty())
		Expect(bundleProposal.BundleSummary).To(BeEmpty())
		Expect(bundleProposal.UpdatedAt).NotTo(BeZero())
		Expect(bundleProposal.VotersValid).To(BeEmpty())
		Expect(bundleProposal.VotersInvalid).To(BeEmpty())
		Expect(bundleProposal.VotersAbstain).To(BeEmpty())
	})
})
