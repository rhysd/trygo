package trygo_test

import (
	"github.com/rhysd/trygo"
	"os"
	"testing"
)

var cwd string

func init() {
	var err error
	cwd, err = os.Getwd()
	if err != nil {
		panic(err)
	}

	// On CI, enabling log would help failure analysis
	if onCI() {
		trygo.InitLog(true)
	}
}

func onTravis() bool {
	return os.Getenv("TRAVIS") != ""
}

func onAppveyor() bool {
	return os.Getenv("APPVEYOR") != ""
}

func onCI() bool {
	return onTravis() || onAppveyor()
}

func skipOnCI(t *testing.T) {
	if onCI() {
		t.Skip("because this test case cannot be run on CI")
	}
}
