package keeper_test

import (
	i "github.com/KYVENetwork/chain/testutil/integration"
	bundletypes "github.com/KYVENetwork/chain/x/bundles/types"
	pooltypes "github.com/KYVENetwork/chain/x/pool/types"
	stakertypes "github.com/KYVENetwork/chain/x/stakers/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

/*

TEST CASES - msg_server_skip_uploader_role.go

* Skip uploader role on data bundle if staker is next uploader
* Skip uploader on data bundle after uploader role has already been skipped
* Skip uploader on data bundle if staker is the only staker in pool
* Skip uploader role on dropped bundle

*/

var _ = Describe("msg_server_skip_uploader_role.go", Ordered, func() {
	s := i.NewCleanChain()

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
	})

	AfterEach(func() {
		s.PerformValidityChecks()
	})

	It("Skip uploader role on data bundle if staker is next uploader", func() {
		// ARRANGE
		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  100 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_1,
			PoolId:     0,
			Valaddress: i.VALADDRESS_1,
		})

		s.CommitAfterSeconds(60)

		// ACT
		s.RunTxBundlesSuccess(&bundletypes.MsgSkipUploaderRole{
			Creator:   i.VALADDRESS_0,
			Staker:    i.STAKER_0,
			PoolId:    0,
			FromIndex: 100,
		})

		// ASSERT
		bundleProposal, found := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(found).To(BeTrue())

		Expect(bundleProposal.PoolId).To(Equal(uint64(0)))
		Expect(bundleProposal.StorageId).To(Equal("y62A3tfbSNcNYDGoL-eXwzyV-Zc9Q0OVtDvR1biJmNI"))
		Expect(bundleProposal.Uploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.DataSize).To(Equal(uint64(100)))
		Expect(bundleProposal.DataHash).To(Equal("test_hash"))
		Expect(bundleProposal.BundleSize).To(Equal(uint64(100)))
		Expect(bundleProposal.FromKey).To(Equal("0"))
		Expect(bundleProposal.ToKey).To(Equal("99"))
		Expect(bundleProposal.BundleSummary).To(Equal("test_value"))
		Expect(bundleProposal.UpdatedAt).NotTo(BeZero())
		Expect(bundleProposal.VotersValid).To(ContainElement(i.STAKER_0))
		Expect(bundleProposal.VotersInvalid).To(BeEmpty())
		Expect(bundleProposal.VotersAbstain).To(BeEmpty())

		// here the next uploader should be always be different after skipping
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_1))
	})

	It("Skip uploader on data bundle after uploader role has already been skipped", func() {
		// ARRANGE
		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  100 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_1,
			PoolId:     0,
			Valaddress: i.VALADDRESS_1,
		})

		s.CommitAfterSeconds(60)

		s.RunTxBundlesSuccess(&bundletypes.MsgSkipUploaderRole{
			Creator:   i.VALADDRESS_0,
			Staker:    i.STAKER_0,
			PoolId:    0,
			FromIndex: 100,
		})

		s.CommitAfterSeconds(60)

		// ACT
		s.RunTxBundlesSuccess(&bundletypes.MsgSkipUploaderRole{
			Creator:   i.VALADDRESS_1,
			Staker:    i.STAKER_1,
			PoolId:    0,
			FromIndex: 100,
		})

		// ASSERT
		bundleProposal, found := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(found).To(BeTrue())

		Expect(bundleProposal.PoolId).To(Equal(uint64(0)))
		Expect(bundleProposal.StorageId).To(Equal("y62A3tfbSNcNYDGoL-eXwzyV-Zc9Q0OVtDvR1biJmNI"))
		Expect(bundleProposal.Uploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.DataSize).To(Equal(uint64(100)))
		Expect(bundleProposal.DataHash).To(Equal("test_hash"))
		Expect(bundleProposal.BundleSize).To(Equal(uint64(100)))
		Expect(bundleProposal.FromKey).To(Equal("0"))
		Expect(bundleProposal.ToKey).To(Equal("99"))
		Expect(bundleProposal.BundleSummary).To(Equal("test_value"))
		Expect(bundleProposal.UpdatedAt).NotTo(BeZero())
		Expect(bundleProposal.VotersValid).To(ContainElement(i.STAKER_0))
		Expect(bundleProposal.VotersInvalid).To(BeEmpty())
		Expect(bundleProposal.VotersAbstain).To(BeEmpty())

		// here the next uploader should be always be different after skipping
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_0))
	})

	It("Skip uploader on data bundle if staker is the only staker in pool", func() {
		// ARRANGE
		s.CommitAfterSeconds(60)

		// ACT
		s.RunTxBundlesSuccess(&bundletypes.MsgSkipUploaderRole{
			Creator:   i.VALADDRESS_0,
			Staker:    i.STAKER_0,
			PoolId:    0,
			FromIndex: 100,
		})

		// ASSERT
		bundleProposal, found := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(found).To(BeTrue())

		Expect(bundleProposal.PoolId).To(Equal(uint64(0)))
		Expect(bundleProposal.StorageId).To(Equal("y62A3tfbSNcNYDGoL-eXwzyV-Zc9Q0OVtDvR1biJmNI"))
		Expect(bundleProposal.Uploader).To(Equal(i.STAKER_0))
		Expect(bundleProposal.DataSize).To(Equal(uint64(100)))
		Expect(bundleProposal.DataHash).To(Equal("test_hash"))
		Expect(bundleProposal.BundleSize).To(Equal(uint64(100)))
		Expect(bundleProposal.FromKey).To(Equal("0"))
		Expect(bundleProposal.ToKey).To(Equal("99"))
		Expect(bundleProposal.BundleSummary).To(Equal("test_value"))
		Expect(bundleProposal.UpdatedAt).NotTo(BeZero())
		Expect(bundleProposal.VotersValid).To(ContainElement(i.STAKER_0))
		Expect(bundleProposal.VotersInvalid).To(BeEmpty())
		Expect(bundleProposal.VotersAbstain).To(BeEmpty())

		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_0))
	})

	It("Skip uploader role on dropped bundle", func() {
		// ARRANGE
		s.RunTxStakersSuccess(&stakertypes.MsgCreateStaker{
			Creator: i.STAKER_1,
			Amount:  200 * i.KYVE,
		})

		s.RunTxStakersSuccess(&stakertypes.MsgJoinPool{
			Creator:    i.STAKER_1,
			PoolId:     0,
			Valaddress: i.VALADDRESS_1,
		})

		// create dropped bundle
		s.CommitAfterSeconds(60)
		s.CommitAfterSeconds(1)

		// wait for upload interval
		s.CommitAfterSeconds(60)

		// ACT
		s.RunTxBundlesSuccess(&bundletypes.MsgSkipUploaderRole{
			Creator:   i.VALADDRESS_0,
			Staker:    i.STAKER_0,
			PoolId:    0,
			FromIndex: 0,
		})

		// ASSERT
		bundleProposal, found := s.App().BundlesKeeper.GetBundleProposal(s.Ctx(), 0)
		Expect(found).To(BeTrue())

		Expect(bundleProposal.PoolId).To(Equal(uint64(0)))
		Expect(bundleProposal.StorageId).To(BeEmpty())
		Expect(bundleProposal.Uploader).To(BeEmpty())
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

		// here the next uploader should be always be different after skipping
		Expect(bundleProposal.NextUploader).To(Equal(i.STAKER_1))
	})
})
