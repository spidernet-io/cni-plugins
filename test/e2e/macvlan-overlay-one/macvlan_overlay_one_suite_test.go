package macvlan_overlay_one_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMacvlanOverlayOne(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MacvlanOverlayOne Suite")
}
