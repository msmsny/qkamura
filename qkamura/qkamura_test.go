package qkamura

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCommand tests command validation
func TestCommand(t *testing.T) {
	var (
		location     = "tateyama"
		stayDates    = "20210731,20210807"
		roomIDs      = "1,7"
		slackChannel = "dummy-channel"
		slackToken   = "dummy-token"
	)
	testCases := map[string]struct {
		location  string
		stayDates string
		roomIDs   string
		isError   bool
	}{
		"normal": {
			location:  location,
			stayDates: stayDates,
			roomIDs:   roomIDs,
			isError:   false,
		},
		"noLocation": {
			location:  "",
			stayDates: stayDates,
			roomIDs:   roomIDs,
			isError:   true,
		},
		"noStayDates": {
			location:  location,
			stayDates: "",
			roomIDs:   roomIDs,
			isError:   true,
		},
		"noRoomIDs": {
			location:  location,
			stayDates: stayDates,
			roomIDs:   "",
			isError:   true,
		},
	}
	for testCase, tt := range testCases {
		t.Run(testCase, func(t *testing.T) {
			cmd := NewQkamuraCommand()
			// noop stub
			cmd.RunE = func(*cobra.Command, []string) error {
				return nil
			}
			require.NoError(t, cmd.Flags().Set("location", tt.location))
			if tt.stayDates != "" {
				require.NoError(t, cmd.Flags().Set("stay-dates", tt.stayDates))
			}
			if tt.roomIDs != "" {
				require.NoError(t, cmd.Flags().Set("room-ids", tt.roomIDs))
			}
			require.NoError(t, cmd.Flags().Set("slack-channel", slackChannel))
			require.NoError(t, cmd.Flags().Set("slack-token", slackToken))

			err := cmd.Execute()
			assert.Equal(t, tt.isError, err != nil, err)
		})
	}
}

// TestRun tests normal test cases and some validation
func TestRun(t *testing.T) {
	var (
		httpClient   = http.DefaultClient
		location     = "dummy-location"
		slackChannel = "dummy-channel"
		slackToken   = "dummy-token"
		debug        = false
	)
	// response has vacancy for stayDates: 20210806, roomID: 1
	qkamuraResponse, err := os.ReadFile("testdata/qkamura_response.txt")
	require.NoError(t, err)

	testCases := map[string]struct {
		stayDates   []int
		roomIDs     []int
		callQkamura bool
		callSlack   bool // found vacancy
		isError     bool
	}{
		"noVacancy": {
			stayDates:   []int{20210731, 20210807},
			roomIDs:     []int{1, 7},
			callQkamura: true,
			callSlack:   false,
			isError:     false,
		},
		"hasVacancy": {
			stayDates:   []int{20210731, 20210806},
			roomIDs:     []int{1, 7},
			callQkamura: true,
			callSlack:   true,
			isError:     false,
		},
		"invalidStartDate": {
			stayDates:   []int{2021731, 20210806},
			roomIDs:     []int{1, 7},
			callQkamura: false,
			callSlack:   false,
			isError:     true,
		},
		"invalidEndDate": {
			stayDates:   []int{20210731, 202186},
			roomIDs:     []int{1, 7},
			callQkamura: false,
			callSlack:   false,
			isError:     true,
		},
	}

	for testCase, tt := range testCases {
		t.Run(testCase, func(t *testing.T) {
			var (
				calledQkamura = false
				calledSlack   = false
			)
			qkamuraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calledQkamura = true
				w.WriteHeader(200)
				fmt.Fprintf(w, "%s", qkamuraResponse)
			}))
			defer qkamuraServer.Close()
			qkamuraURL, err := url.Parse(qkamuraServer.URL)
			require.NoError(t, err)
			qkamuraClient := &qkamuraClient{
				client:    httpClient,
				scheme:    qkamuraURL.Scheme,
				host:      qkamuraURL.Host,
				location:  location,
				stayDates: tt.stayDates,
			}

			slackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calledSlack = true
				w.WriteHeader(200)
				fmt.Fprintf(w, `{"ok":true}`)
			}))
			defer slackServer.Close()
			slackURL, err := url.Parse(slackServer.URL)
			require.NoError(t, err)
			slackClient := &slackClient{
				client: httpClient,
				scheme: slackURL.Scheme,
				host:   slackURL.Host,
			}

			qkamura := &qkamura{
				qkamuraClient: qkamuraClient,
				slackClient:   slackClient,
				location:      location,
				stayDates:     tt.stayDates,
				roomIDs:       tt.roomIDs,
				slackChannel:  slackChannel,
				slackToken:    slackToken,
				debug:         debug,
			}
			err = qkamura.run()
			assert.Equal(t, tt.isError, err != nil, err)
			assert.Equal(t, tt.callQkamura, calledQkamura, "no called qkamura API")
			assert.Equal(t, tt.callSlack, calledSlack, "no called slack API")
		})
	}
}
