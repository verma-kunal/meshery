package app

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/jarcoal/httpmock"
	"github.com/layer5io/meshery/mesheryctl/pkg/utils"
)

func TestOnboardCmd(t *testing.T) {
	// setup current context
	utils.SetupContextEnv(t)

	// initialize mock server for handling requests
	utils.StartMockery(t)

	// create a test helper
	testContext := utils.NewTestHelper(t)

	// get current directory
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Not able to get current working directory")
	}
	currDir := filepath.Dir(filename)
	fixturesDir := filepath.Join(currDir, "fixtures")

	// test scenrios for fetching data
	tests := []struct {
		Name             string
		Args             []string
		ExpectedResponse string
		URLs             []utils.MockURL
		Token            string
		ExpectError      bool
	}{
		{
			Name:             "Onboard Application",
			Args:             []string{"onboard", "-f", filepath.Join(fixturesDir, "sampleApp.golden")},
			ExpectedResponse: "onboard.output.golden",
			URLs: []utils.MockURL{
				{
					Method:       "POST",
					URL:          testContext.BaseURL + "/api/application",
					Response:     "onboard.applicationSave.response.golden",
					ResponseCode: 200,
				},
				{
					Method:       "POST",
					URL:          testContext.BaseURL + "/api/application/deploy",
					Response:     "onboard.applicationdeploy.response.golden",
					ResponseCode: 200,
				},
			},
			Token:       filepath.Join(fixturesDir, "token.golden"),
			ExpectError: false,
		},
		{
			Name:             "Onboard Application with --skip-save",
			Args:             []string{"onboard", "-f", filepath.Join(fixturesDir, "sampleApp.golden"), "--skip-save"},
			ExpectedResponse: "onboard.output.golden",
			URLs: []utils.MockURL{
				{
					Method:       "POST",
					URL:          testContext.BaseURL + "/api/application/deploy",
					Response:     "onboard.applicationdeploy.response.golden",
					ResponseCode: 200,
				},
			},
			Token:       filepath.Join(fixturesDir, "token.golden"),
			ExpectError: false,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			for _, url := range tt.URLs {
				// View api response from golden files
				apiResponse := utils.NewGoldenFile(t, url.Response, fixturesDir).Load()

				// mock response
				httpmock.RegisterResponder(url.Method, url.URL,
					httpmock.NewStringResponder(url.ResponseCode, apiResponse))
			}

			// set token
			utils.TokenFlag = tt.Token

			// Expected response
			testdataDir := filepath.Join(currDir, "testdata")
			golden := utils.NewGoldenFile(t, tt.ExpectedResponse, testdataDir)

			// setting up log to grab logs
			var buf bytes.Buffer
			log.SetOutput(&buf)
			utils.SetupLogrusFormatter()

			AppCmd.SetArgs(tt.Args)
			err := AppCmd.Execute()
			if err != nil {
				// if we're supposed to get an error
				if tt.ExpectError {
					// write it in file
					if *update {
						golden.Write(err.Error())
					}
					expectedResponse := golden.Load()

					utils.Equals(t, expectedResponse, err.Error())
					return
				}
				t.Error(err)
			}

			// response being printed in console
			actualResponse := buf.String()

			// write it in file
			if *update {
				golden.Write(actualResponse)
			}
			expectedResponse := golden.Load()

			utils.Equals(t, expectedResponse, actualResponse)
		})
	}

	// stop mock server
	utils.StopMockery(t)
}
