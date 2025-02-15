package keeper_test

import (
	i "github.com/KYVENetwork/chain/testutil/integration"
	bundletypes "github.com/KYVENetwork/chain/x/bundles/types"
	pooltypes "github.com/KYVENetwork/chain/x/pool/types"
	stakertypes "github.com/KYVENetwork/chain/x/stakers/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

/*

TEST CASES - stakers leave

* Staker leaves, although he is the next uploader and runs into the upload timeout
* Staker leaves, although he was the uploader of the previous round and should receive the uploader reward
* Staker leaves, although he was the uploader of the previous round and should get slashed
* Staker leaves, although he was a voter in the previous round and should get slashed
* Staker leaves, although he was a voter in the previous round and should get a point
* Staker leaves, although he was a voter who did not vote max points in a row should not get slashed

*/

var _ = Describe("stakers leave", Ordered, func() {
	s := i.NewCleanChain()

	initialBalanceStaker0 := s.GetBalanceFromAddress(i.STAKER_0)
	//initialBalanceValaddress0 := s.GetBalanceFromAddress(i.VALADDRESS_0)
	//
	initialBalanceStaker1 := s.GetBalanceFromAddress(i.STAKER_1)
	//initialBalanceValaddress1 := s.GetBalanceFromAddress(i.VALADDRESS_1)
	//
	//initialBalanceStaker2 := s.GetBalanceFromAddress(i.STAKER_2)
	//initialBalanceValaddress2 := s.GetBalanceFromAddress(i.VALADDRESS_2)

	BeforeEach(func() {
		// init new clean chain
		s = i.NewCleanChain()

		// create clean pool for every test case
		s.App().PoolKeeper.AppendPool(s.Ctx(), pooltypes.Pool{
			Name:           "PoolTest",
			MaxBundleSize:  100,
			StartKey:       "0",
			UploadInterval: 60,
			OperatingCost:  10_000,
			Protocol: &pooltypes.Protocol{
				Version:     "0.0.0",
				Binaries:    "{}",
				LastUpgrade: uint64(s.Ctx().BlockTime().Unix()),
			},
			UpgradePlan: &pooltypes.UpgradePlan{},
		})

		s.RunTxPoolSuccess(&pooltypes.MsgFundPool{
			Creator: i.ALICE,
			Id:      0,
			Amount:  100 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_0,
			Amount:  100 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_0,
			PoolId:     0,
			Valaddress: i.VALADDRESS_0,
		})

		s.RunTxBundlesSuccess(&bundletypes.MsgClaimUploaderRole{
			Creator: i.VALADDRESS_0,
			Staker:  i.STAKER_0,
			PoolId:  0,
		})

		initialBalanceStaker0 = s.GetBalanceFromAddress(i.STAKER_0)
		//initialBalanceValaddress0 = s.GetBalanceFromAddress(i.VALADDRESS_0)
		//
		initialBalanceStaker1 = s.GetBalanceFromAddress(i.STAKER_1)
		//initialBalanceValaddress1 = s.GetBalanceFromAddress(i.VALADDRESS_1)
		//
		//initialBalanceStaker2 = s.GetBalanceFromAddress(i.STAKER_2)
		//initialBalanceValaddress2 = s.GetBalanceFromAddress(i.VALADDRESS_2)
	})

	AfterEach(func() {
		s.PerformValidityChecks()
	})

	It("Staker leaves, although he is the next uploader and runs into the upload timeout", func() {
		// ARRANGE
		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  50 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_1,
			PoolId:     0,
			Valaddress: i.VALADDRESS_1,
		})

		// ACT
		s.CommitAfterSeconds(60)

		// leave pool
		s.App().StakersKeeper.RemoveValaccountFromPool(s.Ctx(), 0, i.STAKER_0)

		s.CommitAfterSeconds(s.App().BundlesKeeper.GetUploadTimeout(s.Ctx()))
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_1))
		Expect(bundleProposal.StorageId).To(BeEmpty())

		// check if next uploader is still removed from pool
		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))
		Expect(poolStakers[0]).To(Equal(i.STAKER_1))

		_, valaccountFound := s.App().StakersKeeper.GetValaccount(s.Ctx(), 0, i.STAKER_0)
		Expect(valaccountFound).To(BeFalse())

		Expect(s.App().DelegationKeeper.GetDelegationOfPool(s.Ctx(), 0)).To(Equal(50 * i.KYVE))

		// check if next uploader got not slashed
		Expect(s.App().DelegationKeeper.GetDelegationAmountOfDelegator(s.Ctx(), i.STAKER_0, i.STAKER_0)).To(Equal(100 * i.KYVE))
	})

	It("Staker leaves, although he was the uploader of the previous round and should receive the uploader reward", func() {
		// ARRANGE
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
			FromKey:       "test_key",
			ToKey:         "test_key",
			BundleSummary: "test_value",
		})

		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  50 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_1,
			PoolId:     0,
			Valaddress: i.VALADDRESS_1,
		})

		s.RunTxBundlesSuccess(&bundletypes.MsgVoteBundleProposal{
			Creator:   i.VALADDRESS_1,
			Staker:    i.STAKER_1,
			PoolId:    0,
			StorageId: "y62A3tfbSNcNYDGoL-eXwzyV-Zc9Q0OVtDvR1biJmNI",
			Vote:      bundletypes.VOTE_TYPE_VALID,
		})

		// leave pool
		s.App().StakersKeeper.RemoveValaccountFromPool(s.Ctx(), 0, i.STAKER_0)

		// overwrite next uploader for test purposes
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		bundleProposal.NextUploader = i.STAKER_1
		s.App().BundlesKeeper.SetBundleProposal(s.Ctx(), bundleProposal)

		// ACT
		s.CommitAfterSeconds(60)

		s.RunTxBundlesSuccess(&bundletypes.MsgSubmitBundleProposal{
			Creator:       i.VALADDRESS_1,
			Staker:        i.STAKER_1,
			PoolId:        0,
			StorageId:     "18SRvVuCrB8vy_OCLBaNbXONMVGeflGcw4gGTZ1oUt4",
			DataSize:      100,
			DataHash:      "test_hash",
			FromIndex:     100,
			BundleSize:    100,
			FromKey:       "test_key",
			ToKey:         "test_key",
			BundleSummary: "test_value",
		})

		// ASSERT
		bundleProposal, _ = s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_1))
		Expect(bundleProposal.StorageId).To(Equal("18SRvVuCrB8vy_OCLBaNbXONMVGeflGcw4gGTZ1oUt4"))

		// check if next uploader is still removed from pool
		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))
		Expect(poolStakers[0]).To(Equal(i.STAKER_1))

		_, valaccountFound := s.App().StakersKeeper.GetValaccount(s.Ctx(), 0, i.STAKER_0)
		Expect(valaccountFound).To(BeFalse())

		Expect(s.App().DelegationKeeper.GetDelegationOfPool(s.Ctx(), 0)).To(Equal(50 * i.KYVE))

		// check if next uploader got not slashed
		Expect(s.App().DelegationKeeper.GetDelegationAmountOfDelegator(s.Ctx(), i.STAKER_0, i.STAKER_0)).To(Equal(100 * i.KYVE))

		// check if next uploader received the uploader reward
		pool, _ := s.App().PoolKeeper.GetPool(s.Ctx(), 0)
		uploader, _ := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		balanceUploader := s.GetBalanceFromAddress(i.STAKER_0)

		// calculate uploader rewards
		networkFee := s.App().BundlesKeeper.GetNetworkFee(s.Ctx())
		treasuryReward := uint64(sdk.NewDec(int64(pool.OperatingCost)).Mul(networkFee).TruncateInt64())
		storageReward := uint64(s.App().BundlesKeeper.GetStorageCost(s.Ctx()).MulInt64(100).TruncateInt64())
		totalUploaderReward := pool.OperatingCost - treasuryReward - storageReward

		uploaderPayoutReward := uint64(sdk.NewDec(int64(totalUploaderReward)).Mul(uploader.Commission).TruncateInt64())
		uploaderDelegationReward := totalUploaderReward - uploaderPayoutReward

		// assert payout transfer
		Expect(balanceUploader).To(Equal(initialBalanceStaker0))
		// assert commission rewards
		Expect(uploader.CommissionRewards).To(Equal(uploaderPayoutReward + storageReward))
		// assert uploader self delegation rewards
		Expect(s.App().DelegationKeeper.GetOutstandingRewards(s.Ctx(), i.STAKER_0, i.STAKER_0)).To(Equal(uploaderDelegationReward))
	})

	It("Staker leaves, although he was the uploader of the previous round and should get slashed", func() {
		// ARRANGE
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
			FromKey:       "test_key",
			ToKey:         "test_key",
			BundleSummary: "test_value",
		})

		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  200 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_1,
			PoolId:     0,
			Valaddress: i.VALADDRESS_1,
		})

		s.RunTxBundlesSuccess(&bundletypes.MsgVoteBundleProposal{
			Creator:   i.VALADDRESS_1,
			Staker:    i.STAKER_1,
			PoolId:    0,
			StorageId: "y62A3tfbSNcNYDGoL-eXwzyV-Zc9Q0OVtDvR1biJmNI",
			Vote:      bundletypes.VOTE_TYPE_INVALID,
		})

		// leave pool
		s.App().StakersKeeper.RemoveValaccountFromPool(s.Ctx(), 0, i.STAKER_0)

		// overwrite next uploader for test purposes
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		bundleProposal.NextUploader = i.STAKER_1
		s.App().BundlesKeeper.SetBundleProposal(s.Ctx(), bundleProposal)

		// ACT
		s.CommitAfterSeconds(60)

		s.RunTxBundlesSuccess(&bundletypes.MsgSubmitBundleProposal{
			Creator:       i.VALADDRESS_1,
			Staker:        i.STAKER_1,
			PoolId:        0,
			StorageId:     "18SRvVuCrB8vy_OCLBaNbXONMVGeflGcw4gGTZ1oUt4",
			DataSize:      100,
			DataHash:      "test_hash",
			FromIndex:     100,
			BundleSize:    100,
			FromKey:       "test_key",
			ToKey:         "test_key",
			BundleSummary: "test_value",
		})

		// ASSERT
		bundleProposal, _ = s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_1))
		Expect(bundleProposal.StorageId).To(BeEmpty())

		// check if next uploader is still removed from pool
		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))
		Expect(poolStakers[0]).To(Equal(i.STAKER_1))

		_, valaccountFound := s.App().StakersKeeper.GetValaccount(s.Ctx(), 0, i.STAKER_0)
		Expect(valaccountFound).To(BeFalse())

		// check if next uploader got slashed
		fraction := s.App().DelegationKeeper.GetUploadSlash(s.Ctx())
		slashAmount := uint64(sdk.NewDec(int64(100 * i.KYVE)).Mul(fraction).TruncateInt64())

		Expect(s.App().DelegationKeeper.GetDelegationAmountOfDelegator(s.Ctx(), i.STAKER_0, i.STAKER_0)).To(Equal(100*i.KYVE - slashAmount))
		Expect(s.App().DelegationKeeper.GetDelegationOfPool(s.Ctx(), 0)).To(Equal(200 * i.KYVE))

		// check if next uploader did not receive the uploader reward
		balanceUploader := s.GetBalanceFromAddress(i.STAKER_0)

		// assert payout transfer
		Expect(balanceUploader).To(Equal(initialBalanceStaker0))
		// assert uploader self delegation rewards
		Expect(s.App().DelegationKeeper.GetOutstandingRewards(s.Ctx(), i.STAKER_0, i.STAKER_0)).To(BeZero())
	})

	It("Staker leaves, although he was a voter in the previous round and should get slashed", func() {
		// ARRANGE
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
			FromKey:       "test_key",
			ToKey:         "test_key",
			BundleSummary: "test_value",
		})

		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  50 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_1,
			PoolId:     0,
			Valaddress: i.VALADDRESS_1,
		})

		initialBalanceStaker1 = s.GetBalanceFromAddress(i.STAKER_1)

		s.RunTxBundlesSuccess(&bundletypes.MsgVoteBundleProposal{
			Creator:   i.VALADDRESS_1,
			Staker:    i.STAKER_1,
			PoolId:    0,
			StorageId: "y62A3tfbSNcNYDGoL-eXwzyV-Zc9Q0OVtDvR1biJmNI",
			Vote:      bundletypes.VOTE_TYPE_INVALID,
		})

		// leave pool
		s.App().StakersKeeper.RemoveValaccountFromPool(s.Ctx(), 0, i.STAKER_1)

		// overwrite next uploader for test purposes
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		bundleProposal.NextUploader = i.STAKER_0
		s.App().BundlesKeeper.SetBundleProposal(s.Ctx(), bundleProposal)

		// ACT
		s.CommitAfterSeconds(60)

		s.RunTxBundlesSuccess(&bundletypes.MsgSubmitBundleProposal{
			Creator:       i.VALADDRESS_0,
			Staker:        i.STAKER_0,
			PoolId:        0,
			StorageId:     "18SRvVuCrB8vy_OCLBaNbXONMVGeflGcw4gGTZ1oUt4",
			DataSize:      100,
			DataHash:      "test_hash",
			FromIndex:     100,
			BundleSize:    100,
			FromKey:       "test_key",
			ToKey:         "test_key",
			BundleSummary: "test_value",
		})

		// ASSERT
		bundleProposal, _ = s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.StorageId).To(Equal("18SRvVuCrB8vy_OCLBaNbXONMVGeflGcw4gGTZ1oUt4"))

		// check if next uploader is still removed from pool
		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))
		Expect(poolStakers[0]).To(Equal(i.STAKER_0))

		_, valaccountFound := s.App().StakersKeeper.GetValaccount(s.Ctx(), 0, i.STAKER_1)
		Expect(valaccountFound).To(BeFalse())

		// check if voter got slashed
		fraction := s.App().DelegationKeeper.GetVoteSlash(s.Ctx())
		slashAmount := uint64(sdk.NewDec(int64(50 * i.KYVE)).Mul(fraction).TruncateInt64())

		Expect(s.App().DelegationKeeper.GetDelegationAmountOfDelegator(s.Ctx(), i.STAKER_1, i.STAKER_1)).To(Equal(50*i.KYVE - slashAmount))
		Expect(s.App().DelegationKeeper.GetDelegationOfPool(s.Ctx(), 0)).To(Equal(100 * i.KYVE))

		// check if next uploader did not receive any rewards
		balanceVoter := s.GetBalanceFromAddress(i.STAKER_1)

		// assert payout transfer
		Expect(balanceVoter).To(Equal(initialBalanceStaker1))
		// assert uploader self delegation rewards
		Expect(s.App().DelegationKeeper.GetOutstandingRewards(s.Ctx(), i.STAKER_1, i.STAKER_1)).To(BeZero())
	})

	It("Staker leaves, although he was a voter in the previous round and should get a point", func() {
		// ARRANGE
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
			FromKey:       "test_key",
			ToKey:         "test_key",
			BundleSummary: "test_value",
		})

		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  50 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_1,
			PoolId:     0,
			Valaddress: i.VALADDRESS_1,
		})

		initialBalanceStaker1 = s.GetBalanceFromAddress(i.STAKER_1)

		// leave pool
		s.App().StakersKeeper.RemoveValaccountFromPool(s.Ctx(), 0, i.STAKER_1)

		// do not vote

		// overwrite next uploader for test purposes
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		bundleProposal.NextUploader = i.STAKER_0
		s.App().BundlesKeeper.SetBundleProposal(s.Ctx(), bundleProposal)

		// ACT
		s.CommitAfterSeconds(60)

		s.RunTxBundlesSuccess(&bundletypes.MsgSubmitBundleProposal{
			Creator:       i.VALADDRESS_0,
			Staker:        i.STAKER_0,
			PoolId:        0,
			StorageId:     "18SRvVuCrB8vy_OCLBaNbXONMVGeflGcw4gGTZ1oUt4",
			DataSize:      100,
			DataHash:      "test_hash",
			FromIndex:     100,
			BundleSize:    100,
			FromKey:       "test_key",
			ToKey:         "test_key",
			BundleSummary: "test_value",
		})

		// ASSERT
		bundleProposal, _ = s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.StorageId).To(Equal("18SRvVuCrB8vy_OCLBaNbXONMVGeflGcw4gGTZ1oUt4"))

		// check if next uploader is still removed from pool
		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))
		Expect(poolStakers[0]).To(Equal(i.STAKER_0))

		_, valaccountFound := s.App().StakersKeeper.GetValaccount(s.Ctx(), 0, i.STAKER_1)
		Expect(valaccountFound).To(BeFalse())

		// check if voter status

		Expect(s.App().DelegationKeeper.GetDelegationAmountOfDelegator(s.Ctx(), i.STAKER_1, i.STAKER_1)).To(Equal(50 * i.KYVE))
		Expect(s.App().DelegationKeeper.GetDelegationOfPool(s.Ctx(), 0)).To(Equal(100 * i.KYVE))

		// check if next uploader did not receive any rewards
		balanceVoter := s.GetBalanceFromAddress(i.STAKER_1)

		// assert payout transfer
		Expect(balanceVoter).To(Equal(initialBalanceStaker1))
		// assert uploader self delegation rewards
		Expect(s.App().DelegationKeeper.GetOutstandingRewards(s.Ctx(), i.STAKER_1, i.STAKER_1)).To(BeZero())
	})

	It("Staker leaves, although he was a voter who did not vote max points in a row should not get slashed", func() {
		// ARRANGE
		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  50 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_1,
			PoolId:     0,
			Valaddress: i.VALADDRESS_1,
		})

		s.CommitAfterSeconds(60)

		maxPoints := int(s.App().BundlesKeeper.GetMaxPoints(s.Ctx()))

		for r := 0; r < maxPoints; r++ {
			s.RunTxBundlesSuccess(&bundletypes.MsgSubmitBundleProposal{
				Creator:       i.VALADDRESS_0,
				Staker:        i.STAKER_0,
				PoolId:        0,
				StorageId:     "P9edn0bjEfMU_lecFDIPLvGO2v2ltpFNUMWp5kgPddg",
				DataSize:      100,
				DataHash:      "test_hash",
				FromIndex:     uint64(r * 100),
				BundleSize:    100,
				FromKey:       "test_key",
				ToKey:         "test_key",
				BundleSummary: "test_value",
			})

			// overwrite next uploader for test purposes
			bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
			bundleProposal.NextUploader = i.STAKER_0
			s.App().BundlesKeeper.SetBundleProposal(s.Ctx(), bundleProposal)

			s.CommitAfterSeconds(60)

			// do not vote
		}

		initialBalanceStaker1 = s.GetBalanceFromAddress(i.STAKER_1)

		// leave pool
		s.App().StakersKeeper.RemoveValaccountFromPool(s.Ctx(), 0, i.STAKER_1)

		// ACT
		s.RunTxBundlesSuccess(&bundletypes.MsgSubmitBundleProposal{
			Creator:       i.VALADDRESS_0,
			Staker:        i.STAKER_0,
			PoolId:        0,
			StorageId:     "18SRvVuCrB8vy_OCLBaNbXONMVGeflGcw4gGTZ1oUt4",
			DataSize:      100,
			DataHash:      "test_hash",
			FromIndex:     2400,
			BundleSize:    100,
			FromKey:       "test_key",
			ToKey:         "test_key",
			BundleSummary: "test_value",
		})

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.StorageId).To(Equal("18SRvVuCrB8vy_OCLBaNbXONMVGeflGcw4gGTZ1oUt4"))

		// check if next uploader is still removed from pool
		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))
		Expect(poolStakers[0]).To(Equal(i.STAKER_0))

		_, valaccountFound := s.App().StakersKeeper.GetValaccount(s.Ctx(), 0, i.STAKER_1)
		Expect(valaccountFound).To(BeFalse())

		// check if voter not got slashed
		Expect(s.App().DelegationKeeper.GetDelegationAmountOfDelegator(s.Ctx(), i.STAKER_1, i.STAKER_1)).To(Equal(50 * i.KYVE))
		Expect(s.App().DelegationKeeper.GetDelegationOfPool(s.Ctx(), 0)).To(Equal(100 * i.KYVE))

		// check if next uploader did not receive any rewards
		balanceVoter := s.GetBalanceFromAddress(i.STAKER_1)

		// assert payout transfer
		Expect(balanceVoter).To(Equal(initialBalanceStaker1))
		// assert uploader self delegation rewards
		Expect(s.App().DelegationKeeper.GetOutstandingRewards(s.Ctx(), i.STAKER_1, i.STAKER_1)).To(BeZero())
	})
})
