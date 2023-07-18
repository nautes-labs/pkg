package configs_test

import (
	"os"
	"testing"

	configs "github.com/nautes-labs/pkg/pkg/nautesconfigs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestNautesconfigs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Nautesconfigs Suite")
}

var configString string
var configPath = "/tmp/config.yaml"

var _ = Describe("New nautes config", func() {
	BeforeEach(func() {
		configString = `
nautes:
  namespace: "testNamespace"
`
		err := os.WriteFile(configPath, []byte(configString), 0600)
		Expect(err).Should(BeNil())
	})

	AfterEach(func() {
		err := os.Remove(configPath)
		Expect(err).Should(BeNil())
		os.Unsetenv(configs.EnvNautesConfigPath)
	})

	It("will read from input if input is not empty.", func() {
		cfg, err := configs.NewNautesConfigFromFile(configs.FilePath(configPath))
		Expect(err).Should(BeNil())
		Expect(cfg.Nautes.Namespace).Should(Equal("testNamespace"))
	})

	It("will read from env when env is set.", func() {
		os.Setenv(configs.EnvNautesConfigPath, configPath)
		cfg, err := configs.NewNautesConfigFromFile()
		Expect(err).Should(BeNil())
		Expect(cfg.Nautes.Namespace).Should(Equal("testNamespace"))
	})

	It("will throw an error if file not exist", func() {
		_, err := configs.NewNautesConfigFromFile()
		Expect(os.IsNotExist(err)).Should(BeTrue())
	})
})
