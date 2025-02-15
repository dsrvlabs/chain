package keeper_test

import (
	i "github.com/KYVENetwork/chain/testutil/integration"
	bundletypes "github.com/KYVENetwork/chain/x/bundles/types"
	delegationtypes "github.com/KYVENetwork/chain/x/delegation/types"
	pooltypes "github.com/KYVENetwork/chain/x/pool/types"
	stakertypes "github.com/KYVENetwork/chain/x/stakers/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

/*

TEST CASES - logic_end_block_handle_upload_timeout.go

* Next uploader can stay although pool ran out of funds
* First staker who joins gets automatically chosen as next uploader
* Next uploader gets removed due to pool upgrading
* Next uploader gets removed due to pool being disabled
* Next uploader gets removed due to pool not reaching min stake
* Staker is next uploader of genesis bundle and upload interval and timeout does not pass
* Staker is next uploader of genesis bundle and upload timeout does not pass but upload interval passes
* Staker is next uploader of genesis bundle and upload timeout does pass together with upload interval
* Staker is next uploader of bundle proposal and upload interval does not pass
* Staker is next uploader of bundle proposal and upload timeout does not pass
* Staker is next uploader of bundle proposal and upload timeout passes
* Staker with already max points is next uploader of bundle proposal and upload timeout passes
* A bundle proposal with no quorum does not reach the upload interval
* A bundle proposal with no quorum does reach the upload interval
* Staker who just left the pool is next uploader of dropped bundle proposal and upload timeout passes
* Staker who just left the pool is next uploader of valid bundle proposal and upload timeout passes
* Staker who just left the pool is next uploader of invalid bundle proposal and upload timeout passes
* Staker with already max points is next uploader of bundle proposal in a second pool and upload timeout passes

*/

var _ = Describe("logic_end_block_handle_upload_timeout.go", Ordered, func() {
	s := i.NewCleanChain()

	BeforeEach(func() {
		// init new clean chain
		s = i.NewCleanChain()

		// create clean pool for every test case
		s.App().PoolKeeper.AppendPool(s.Ctx(), pooltypes.Pool{
			Name:           "PoolTest",
			MaxBundleSize:  100,
			StartKey:       "0",
			MinDelegation:  100 * i.KYVE,
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
	})

	AfterEach(func() {
		s.PerformValidityChecks()
	})

	It("Next uploader can stay although pool ran out of funds", func() {
		// ARRANGE
		s.RunTxBundlesSuccess(&bundletypes.MsgClaimUploaderRole{
			Creator: i.VALADDRESS_0,
			Staker:  i.STAKER_0,
			PoolId:  0,
		})

		s.RunTxPoolSuccess(&pooltypes.MsgDefundPool{
			Creator: i.ALICE,
			Id:      0,
			Amount:  100 * i.KYVE,
		})

		// ACT
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.StorageId).To(BeEmpty())

		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		Expect(found).To(BeTrue())
		Expect(s.App().DelegationKeeper.GetDelegationAmount(s.Ctx(), i.STAKER_0)).To(Equal(100 * i.KYVE))
	})

	It("First staker who joins gets automatically chosen as next uploader", func() {
		// ACT
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.StorageId).To(BeEmpty())

		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		Expect(found).To(BeTrue())
		Expect(s.App().DelegationKeeper.GetDelegationAmount(s.Ctx(), i.STAKER_0)).To(Equal(100 * i.KYVE))
	})

	It("Next uploader gets removed due to pool upgrading", func() {
		// ARRANGE
		s.RunTxBundlesSuccess(&bundletypes.MsgClaimUploaderRole{
			Creator: i.VALADDRESS_0,
			Staker:  i.STAKER_0,
			PoolId:  0,
		})

		pool, _ := s.App().PoolKeeper.GetPool(s.Ctx(), 0)

		pool.UpgradePlan = &pooltypes.UpgradePlan{
			Version:     "1.0.0",
			Binaries:    "{}",
			ScheduledAt: uint64(s.Ctx().BlockTime().Unix()),
			Duration:    3600,
		}

		s.App().PoolKeeper.SetPool(s.Ctx(), pool)

		// ACT
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(BeEmpty())
		Expect(bundleProposal.StorageId).To(BeEmpty())

		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		Expect(found).To(BeTrue())
		Expect(s.App().DelegationKeeper.GetDelegationAmount(s.Ctx(), i.STAKER_0)).To(Equal(100 * i.KYVE))
	})

	It("Next uploader gets removed due to pool being disabled", func() {
		// ARRANGE
		s.RunTxBundlesSuccess(&bundletypes.MsgClaimUploaderRole{
			Creator: i.VALADDRESS_0,
			Staker:  i.STAKER_0,
			PoolId:  0,
		})

		pool, _ := s.App().PoolKeeper.GetPool(s.Ctx(), 0)

		pool.Disabled = true

		s.App().PoolKeeper.SetPool(s.Ctx(), pool)

		// ACT
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(BeEmpty())
		Expect(bundleProposal.StorageId).To(BeEmpty())

		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		Expect(found).To(BeTrue())
		Expect(s.App().DelegationKeeper.GetDelegationAmount(s.Ctx(), i.STAKER_0)).To(Equal(100 * i.KYVE))
	})

	It("Next uploader gets removed due to pool not reaching min stake", func() {
		// ARRANGE
		s.RunTxBundlesSuccess(&bundletypes.MsgClaimUploaderRole{
			Creator: i.VALADDRESS_0,
			Staker:  i.STAKER_0,
			PoolId:  0,
		})

		s.RunTxDelegatorSuccess(&delegationtypes.MsgUndelegate{
			Creator: i.STAKER_0,
			Staker:  i.STAKER_0,
			Amount:  50 * i.KYVE,
		})

		s.CommitAfterSeconds(s.App().DelegationKeeper.GetUnbondingDelegationTime(s.Ctx()))
		s.CommitAfterSeconds(1)

		// ACT
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(BeEmpty())
		Expect(bundleProposal.StorageId).To(BeEmpty())

		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		Expect(found).To(BeTrue())
		Expect(s.App().DelegationKeeper.GetDelegationAmount(s.Ctx(), i.STAKER_0)).To(Equal(50 * i.KYVE))
	})

	It("Staker is next uploader of genesis bundle and upload interval and timeout does not pass", func() {
		// ARRANGE
		s.RunTxBundlesSuccess(&bundletypes.MsgClaimUploaderRole{
			Creator: i.VALADDRESS_0,
			Staker:  i.STAKER_0,
			PoolId:  0,
		})

		// ACT
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.StorageId).To(BeEmpty())

		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		Expect(found).To(BeTrue())
		Expect(s.App().DelegationKeeper.GetDelegationAmount(s.Ctx(), i.STAKER_0)).To(Equal(100 * i.KYVE))
	})

	It("Staker is next uploader of genesis bundle and upload timeout does not pass but upload interval passes", func() {
		// ARRANGE
		s.RunTxBundlesSuccess(&bundletypes.MsgClaimUploaderRole{
			Creator: i.VALADDRESS_0,
			Staker:  i.STAKER_0,
			PoolId:  0,
		})

		// ACT
		s.CommitAfterSeconds(60)
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.StorageId).To(BeEmpty())

		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		Expect(found).To(BeTrue())
		Expect(s.App().DelegationKeeper.GetDelegationAmount(s.Ctx(), i.STAKER_0)).To(Equal(100 * i.KYVE))
	})

	It("Staker is next uploader of genesis bundle and upload timeout does pass together with upload interval", func() {
		// ARRANGE
		s.RunTxBundlesSuccess(&bundletypes.MsgClaimUploaderRole{
			Creator: i.VALADDRESS_0,
			Staker:  i.STAKER_0,
			PoolId:  0,
		})

		// ACT
		s.CommitAfterSeconds(s.App().BundlesKeeper.GetUploadTimeout(s.Ctx()))
		s.CommitAfterSeconds(60)
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.StorageId).To(BeEmpty())

		// check if next uploader got not removed from pool
		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))

		// check if next uploader received a point
		valaccount, _ := s.App().StakersKeeper.GetValaccount(s.Ctx(), 0, i.STAKER_0)
		Expect(valaccount.Points).To(Equal(uint64(1)))

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		Expect(found).To(BeTrue())

		Expect(s.App().DelegationKeeper.GetDelegationOfPool(s.Ctx(), 0)).To(Equal(100 * i.KYVE))

		// check if next uploader not got slashed
		expectedBalance := 100 * i.KYVE
		Expect(expectedBalance).To(Equal(s.App().DelegationKeeper.GetDelegationAmountOfDelegator(s.Ctx(), i.STAKER_0, i.STAKER_0)))
	})

	It("Staker is next uploader of bundle proposal and upload interval does not pass", func() {
		// ARRANGE
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

		// ACT
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.StorageId).To(Equal("y62A3tfbSNcNYDGoL-eXwzyV-Zc9Q0OVtDvR1biJmNI"))

		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		Expect(found).To(BeTrue())
		Expect(s.App().DelegationKeeper.GetDelegationAmount(s.Ctx(), i.STAKER_0)).To(Equal(100 * i.KYVE))
	})

	It("Staker is next uploader of bundle proposal and upload timeout does not pass", func() {
		// ARRANGE
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

		// ACT
		s.CommitAfterSeconds(60)
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.StorageId).To(Equal("y62A3tfbSNcNYDGoL-eXwzyV-Zc9Q0OVtDvR1biJmNI"))

		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		Expect(found).To(BeTrue())
		Expect(s.App().DelegationKeeper.GetDelegationAmount(s.Ctx(), i.STAKER_0)).To(Equal(100 * i.KYVE))
	})

	It("Staker is next uploader of bundle proposal and upload timeout passes", func() {
		// ARRANGE
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

		// ACT
		s.CommitAfterSeconds(s.App().BundlesKeeper.GetUploadTimeout(s.Ctx()))
		s.CommitAfterSeconds(60)
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.StorageId).To(Equal("y62A3tfbSNcNYDGoL-eXwzyV-Zc9Q0OVtDvR1biJmNI"))

		// check if next uploader got not removed from pool
		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))

		// check if next uploader received a point
		valaccount, _ := s.App().StakersKeeper.GetValaccount(s.Ctx(), 0, i.STAKER_0)
		Expect(valaccount.Points).To(Equal(uint64(1)))

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		Expect(found).To(BeTrue())

		Expect(s.App().DelegationKeeper.GetDelegationOfPool(s.Ctx(), 0)).To(Equal(100 * i.KYVE))

		// check if next uploader not got slashed
		expectedBalance := 100 * i.KYVE
		Expect(expectedBalance).To(Equal(s.App().DelegationKeeper.GetDelegationAmountOfDelegator(s.Ctx(), i.STAKER_0, i.STAKER_0)))
	})

	It("Staker with already max points is next uploader of bundle proposal and upload timeout passes", func() {
		// ARRANGE
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

		s.CommitAfterSeconds(60)

		maxPoints := int(s.App().BundlesKeeper.GetMaxPoints(s.Ctx())) - 1

		for r := 1; r <= maxPoints; r++ {
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

		// overwrite next uploader with staker_1
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		bundleProposal.NextUploader = i.STAKER_1
		s.App().BundlesKeeper.SetBundleProposal(s.Ctx(), bundleProposal)

		// ACT
		s.CommitAfterSeconds(s.App().BundlesKeeper.GetUploadTimeout(s.Ctx()))
		s.CommitAfterSeconds(60)
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ = s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.StorageId).To(Equal("P9edn0bjEfMU_lecFDIPLvGO2v2ltpFNUMWp5kgPddg"))

		// check if next uploader got not removed from pool
		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))

		// check if next uploader received a point
		_, valaccountFound := s.App().StakersKeeper.GetValaccount(s.Ctx(), 0, i.STAKER_1)
		Expect(valaccountFound).To(BeFalse())

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_1)
		Expect(found).To(BeTrue())

		Expect(s.App().DelegationKeeper.GetDelegationOfPool(s.Ctx(), 0)).To(Equal(100 * i.KYVE))

		// check if next uploader not got slashed
		slashAmountRatio := s.App().DelegationKeeper.GetTimeoutSlash(s.Ctx())
		expectedBalance := 50*i.KYVE - uint64(sdk.NewDec(int64(50*i.KYVE)).Mul(slashAmountRatio).TruncateInt64())

		Expect(expectedBalance).To(Equal(s.App().DelegationKeeper.GetDelegationAmountOfDelegator(s.Ctx(), i.STAKER_1, i.STAKER_1)))
	})

	It("A bundle proposal with no quorum does not reach the upload interval", func() {
		// ARRANGE
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

		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  100 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_1,
			PoolId:     0,
			Valaddress: i.VALADDRESS_1,
		})

		// ACT
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.StorageId).To(Equal("y62A3tfbSNcNYDGoL-eXwzyV-Zc9Q0OVtDvR1biJmNI"))

		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(2))

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		Expect(found).To(BeTrue())
		Expect(s.App().DelegationKeeper.GetDelegationAmount(s.Ctx(), i.STAKER_0)).To(Equal(100 * i.KYVE))
	})

	It("A bundle proposal with no quorum does reach the upload interval", func() {
		// ARRANGE
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

		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  100 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_1,
			PoolId:     0,
			Valaddress: i.VALADDRESS_1,
		})

		// ACT
		s.CommitAfterSeconds(60)
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)

		Expect(bundleProposal.StorageId).To(BeEmpty())
		Expect(bundleProposal.Uploader).To(BeEmpty())
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.DataSize).To(BeZero())
		Expect(bundleProposal.DataHash).To(BeEmpty())
		Expect(bundleProposal.BundleSize).To(BeZero())
		Expect(bundleProposal.FromKey).To(BeEmpty())
		Expect(bundleProposal.ToKey).To(BeEmpty())
		Expect(bundleProposal.BundleSummary).To(BeEmpty())
		Expect(bundleProposal.VotersValid).To(BeEmpty())
		Expect(bundleProposal.VotersInvalid).To(BeEmpty())
		Expect(bundleProposal.VotersAbstain).To(BeEmpty())

		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(2))

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		Expect(found).To(BeTrue())
		Expect(s.App().DelegationKeeper.GetDelegationAmount(s.Ctx(), i.STAKER_0)).To(Equal(100 * i.KYVE))
	})

	It("Staker who just left the pool is next uploader of dropped bundle proposal and upload timeout passes", func() {
		// ARRANGE
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

		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  100 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_1,
			PoolId:     0,
			Valaddress: i.VALADDRESS_1,
		})

		// remove valaccount directly from pool
		s.App().StakersKeeper.RemoveValaccountFromPool(s.Ctx(), 0, i.STAKER_0)

		// ACT
		s.CommitAfterSeconds(60)
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_1))
		Expect(bundleProposal.StorageId).To(BeEmpty())

		// check if next uploader got removed from pool
		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		Expect(found).To(BeTrue())

		Expect(s.App().DelegationKeeper.GetDelegationOfPool(s.Ctx(), 0)).To(Equal(100 * i.KYVE))

		// check if next uploader not got slashed
		expectedBalance := 100 * i.KYVE

		Expect(expectedBalance).To(Equal(s.App().DelegationKeeper.GetDelegationAmountOfDelegator(s.Ctx(), i.STAKER_0, i.STAKER_0)))
	})

	It("Staker who just left the pool is next uploader of valid bundle proposal and upload timeout passes", func() {
		// ARRANGE
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

		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  100 * i.KYVE,
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

		// remove valaccount directly from pool
		s.App().StakersKeeper.RemoveValaccountFromPool(s.Ctx(), 0, i.STAKER_0)

		// ACT
		s.CommitAfterSeconds(s.App().BundlesKeeper.GetUploadTimeout(s.Ctx()))
		s.CommitAfterSeconds(60)
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_1))
		Expect(bundleProposal.StorageId).To(Equal("y62A3tfbSNcNYDGoL-eXwzyV-Zc9Q0OVtDvR1biJmNI"))

		// check if next uploader got removed from pool
		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		Expect(found).To(BeTrue())

		Expect(s.App().DelegationKeeper.GetDelegationOfPool(s.Ctx(), 0)).To(Equal(100 * i.KYVE))

		// check if next uploader not got slashed
		expectedBalance := 100 * i.KYVE

		Expect(expectedBalance).To(Equal(s.App().DelegationKeeper.GetDelegationAmountOfDelegator(s.Ctx(), i.STAKER_0, i.STAKER_0)))
	})

	It("Staker who just left the pool is next uploader of invalid bundle proposal and upload timeout passes", func() {
		// ARRANGE
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

		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  100 * i.KYVE,
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

		// remove valaccount directly from pool
		s.App().StakersKeeper.RemoveValaccountFromPool(s.Ctx(), 0, i.STAKER_0)

		// ACT
		s.CommitAfterSeconds(s.App().BundlesKeeper.GetUploadTimeout(s.Ctx()))
		s.CommitAfterSeconds(60)
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_1))
		Expect(bundleProposal.StorageId).To(Equal("y62A3tfbSNcNYDGoL-eXwzyV-Zc9Q0OVtDvR1biJmNI"))

		// check if next uploader got removed from pool
		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 0)
		Expect(poolStakers).To(HaveLen(1))

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_0)
		Expect(found).To(BeTrue())

		Expect(s.App().DelegationKeeper.GetDelegationOfPool(s.Ctx(), 0)).To(Equal(100 * i.KYVE))

		// check if next uploader not got slashed
		expectedBalance := 100 * i.KYVE

		Expect(expectedBalance).To(Equal(s.App().DelegationKeeper.GetDelegationAmountOfDelegator(s.Ctx(), i.STAKER_0, i.STAKER_0)))
	})

	It("Staker with already max points is next uploader of bundle proposal in a second pool and upload timeout passes", func() {
		// ARRANGE
		s.App().PoolKeeper.AppendPool(s.Ctx(), pooltypes.Pool{
			Name:           "PoolTest2",
			MaxBundleSize:  100,
			StartKey:       "0",
			MinDelegation:  100 * i.KYVE,
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
			Id:      1,
			Amount:  100 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  100 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_1,
			PoolId:     1,
			Valaddress: i.VALADDRESS_1,
		})

		s.RunTxBundlesSuccess(&bundletypes.MsgClaimUploaderRole{
			Creator: i.VALADDRESS_1,
			Staker:  i.STAKER_1,
			PoolId:  1,
		})

		s.CommitAfterSeconds(60)

		s.RunTxBundlesSuccess(&bundletypes.MsgSubmitBundleProposal{
			Creator:       i.VALADDRESS_1,
			Staker:        i.STAKER_1,
			PoolId:        1,
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
			Creator: i.STAKER_2,
			Amount:  50 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_2,
			PoolId:     1,
			Valaddress: i.VALADDRESS_2,
		})

		s.CommitAfterSeconds(60)

		maxPoints := int(s.App().BundlesKeeper.GetMaxPoints(s.Ctx())) - 1

		for r := 1; r <= maxPoints; r++ {
			s.RunTxBundlesSuccess(&bundletypes.MsgSubmitBundleProposal{
				Creator:       i.VALADDRESS_1,
				Staker:        i.STAKER_1,
				PoolId:        1,
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
			bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 1)
			bundleProposal.NextUploader = i.STAKER_1
			s.App().BundlesKeeper.SetBundleProposal(s.Ctx(), bundleProposal)

			s.CommitAfterSeconds(60)

			// do not vote
		}

		// overwrite next uploader with staker_1
		bundleProposal, _ := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 1)
		bundleProposal.NextUploader = i.STAKER_2
		s.App().BundlesKeeper.SetBundleProposal(s.Ctx(), bundleProposal)

		// ACT
		s.CommitAfterSeconds(s.App().BundlesKeeper.GetUploadTimeout(s.Ctx()))
		s.CommitAfterSeconds(60)
		s.CommitAfterSeconds(1)

		// ASSERT
		bundleProposal, _ = s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 1)
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_1))
		Expect(bundleProposal.StorageId).To(Equal("P9edn0bjEfMU_lecFDIPLvGO2v2ltpFNUMWp5kgPddg"))

		// check if next uploader got not removed from pool
		poolStakers := s.App().StakersKeeper.GetAllStakerAddressesOfPool(s.Ctx(), 1)
		Expect(poolStakers).To(HaveLen(1))

		// check if next uploader received a point
		_, valaccountFound := s.App().StakersKeeper.GetValaccount(s.Ctx(), 1, i.STAKER_2)
		Expect(valaccountFound).To(BeFalse())

		_, found := s.App().StakersKeeper.GetStaker(s.Ctx(), i.STAKER_2)
		Expect(found).To(BeTrue())

		Expect(s.App().DelegationKeeper.GetDelegationOfPool(s.Ctx(), 1)).To(Equal(100 * i.KYVE))

		// check if next uploader not got slashed
		slashAmountRatio := s.App().DelegationKeeper.GetTimeoutSlash(s.Ctx())
		expectedBalance := 50*i.KYVE - uint64(sdk.NewDec(int64(50*i.KYVE)).Mul(slashAmountRatio).TruncateInt64())

		Expect(expectedBalance).To(Equal(s.App().DelegationKeeper.GetDelegationAmountOfDelegator(s.Ctx(), i.STAKER_2, i.STAKER_2)))
	})
})
