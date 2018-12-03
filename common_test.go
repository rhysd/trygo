package trygo_test

import (
	"github.com/rhysd/trygo"
	"os"
	"testing"
)

var (
	cwd        string
	onTravisCI bool
	onAppveyor bool
	onCI       bool
)

func init() {
	var err error
	cwd, err = os.Getwd()
	if err != nil {
		panic(err)
	}

	onTravisCI = os.Getenv("TRAVIS") != ""
	onAppveyor = os.Getenv("APPVEYOR") != ""
	onCI = onTravisCI || onAppveyor

	// On CI, enabling log would help failure analysis
	if onCI || os.Getenv("ENABLE_TEST_LOG") != "" {
		trygo.InitLog(true)
	}
}

func skipOnCI(t *testing.T) {
	if onCI {
		t.Skip("because this test case cannot be run on CI")
	}
}
