package server

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var logger = zap.NewNop()
var fake = gofakeit.New(0)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}
