package config

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ty "github.com/spidernet-io/cni-plugins/pkg/types"
	"k8s.io/utils/pointer"
)

var _ = Describe("config", func() {
	Context("Test ValidateRoutes", func() {
		It("overlaySubnet can't be empty", func() {
			overlaySubnet := []string{}
			serviceSubnet := []string{"1.1.1.0/24"}
			_, _, err := ValidateRoutes(overlaySubnet, serviceSubnet)
			Expect(err).To(HaveOccurred())
		})

		It("serviceSubnet can't be empty", func() {
			overlaySubnet := []string{"1.1.1.0/24"}
			serviceSubnet := []string{}
			_, _, err := ValidateRoutes(overlaySubnet, serviceSubnet)
			Expect(err).To(HaveOccurred())
		})

		It("ignore leading or trailing spaces", func() {
			overlaySubnet := []string{" 1.1.1.0/24"}
			serviceSubnet := []string{" 2.2.2.0/24 "}
			_, _, err := ValidateRoutes(overlaySubnet, serviceSubnet)
			Expect(err).NotTo(HaveOccurred())
		})

		It("invalid cidr return err", func() {
			overlaySubnet := []string{"abcd"}
			serviceSubnet := []string{"abcd"}
			_, _, err := ValidateRoutes(overlaySubnet, serviceSubnet)
			Expect(err).To(HaveOccurred())
		})

		It("correct cidr config", func() {
			overlaySubnet := []string{"10.69.0.0/12", "fd00:10:244::/64"}
			serviceSubnet := []string{"10.244.0.0/12", "fd00:10:69::/64"}
			_, _, err := ValidateRoutes(overlaySubnet, serviceSubnet)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Test ValidateRPFilterConfig", func() {
		It("no rp_filter config", func() {
			var config *ty.RPFilter
			var want = &ty.RPFilter{
				Enable: pointer.Bool(true),
				Value:  pointer.Int32(2),
			}
			got := ValidateRPFilterConfig(config)
			Expect(got).To(Equal(want))
		})

		It("enable rp_filter but no value given, we give default value to it", func() {
			var config = &ty.RPFilter{
				Enable: pointer.Bool(true),
				Value:  nil,
			}
			var want = &ty.RPFilter{
				Enable: pointer.Bool(true),
				Value:  pointer.Int32(2),
			}
			got := ValidateRPFilterConfig(config)
			Expect(got).To(Equal(want))
		})

		It("give value but disable rp_filter", func() {
			var config = &ty.RPFilter{
				Enable: nil,
				Value:  pointer.Int32(2),
			}
			var want = &ty.RPFilter{
				Enable: nil,
				Value:  pointer.Int32(2),
			}
			got := ValidateRPFilterConfig(config)
			Expect(got).To(Equal(want))
		})

		It("value must be 0/1/2, if not we set it to 2", func() {
			var config = &ty.RPFilter{
				Enable: pointer.Bool(true),
				Value:  pointer.Int32(10),
			}
			var want = &ty.RPFilter{
				Enable: pointer.Bool(true),
				Value:  pointer.Int32(2),
			}
			got := ValidateRPFilterConfig(config)
			Expect(got).To(Equal(want))
		})

		It("correct rp_filter config", func() {
			var config = &ty.RPFilter{
				Enable: pointer.Bool(true),
				Value:  pointer.Int32(1),
			}
			var want = &ty.RPFilter{
				Enable: pointer.Bool(true),
				Value:  pointer.Int32(1),
			}
			got := ValidateRPFilterConfig(config)
			Expect(got).To(Equal(want))
		})
	})

	Context("Test ValidateMigrateRouteConfig", func() {
		It("no config but we give default value -1", func() {
			var config *ty.MigrateRoute
			var want = (*ty.MigrateRoute)(pointer.Int32(-1))
			got := ValidateMigrateRouteConfig(config)
			Expect(got).To(Equal(want))
		})

		It("value must be in -1,0,2, if not we set it to -1", func() {
			var config = (*ty.MigrateRoute)(pointer.Int32(1000))
			var want = (*ty.MigrateRoute)(pointer.Int32(-1))
			got := ValidateMigrateRouteConfig(config)
			Expect(got).To(Equal(want))
		})

		It("correct config", func() {
			var config = (*ty.MigrateRoute)(pointer.Int32(-1))
			var want = (*ty.MigrateRoute)(pointer.Int32(-1))
			got := ValidateMigrateRouteConfig(config)
			Expect(got).To(Equal(want))
		})
	})
})
