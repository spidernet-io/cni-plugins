package macvlan_standalone_two_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMacvlanStandaloneTwo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MacvlanStandaloneTwo Suite")
}
