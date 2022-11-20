package acnil_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAcnil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Acnil Suite")
}
