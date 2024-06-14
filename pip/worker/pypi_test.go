package worker

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/opensbom-generator/parsers/internal/helper"
	"github.com/stretchr/testify/require"
)

func TestGetPackageDataFromPyPi(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimSpace(r.URL.Path) {
		case "/pypi/requests/jso":
			byteData, err := os.ReadFile("../testdata/requests_pypi_data.json")
			if err != nil {
				panic(err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(byteData)
		default:
			http.NotFoundHandler().ServeHTTP(w, r)
		}
	}))
	defer server.Close()

	for name, tc := range map[string]struct {
		packageJSONUrl string
		expectedErr    error
	}{
		"valid package url": {
			packageJSONUrl: "/pypi/requests/jso",
			expectedErr:    nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			mockClient := helper.NewClient(server.URL)
			factory := NewPypiPackageDataFactory(mockClient)
			packageInfo, err := factory.GetPackageData(tc.packageJSONUrl)
			require.ErrorIs(t, tc.expectedErr, err)
			require.NotNil(t, packageInfo)
		})
	}
}
