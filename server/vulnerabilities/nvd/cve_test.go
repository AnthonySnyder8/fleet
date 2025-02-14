package nvd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// firefox93WindowsVulnerabilities was manually generated by visiting:
// https://nvd.nist.gov/vuln/search/results?form_type=Advanced&results_type=overview&isCpeNameSearch=true&seach_type=all&query=cpe:2.3:a:mozilla:firefox:93.0:*:*:*:*:*:*:*
var firefox93WindowsVulnerabilities = []string{
	"CVE-2021-43540",
	"CVE-2021-38503",
	"CVE-2021-38504",
	"CVE-2021-38506",
	"CVE-2021-38507",
	"CVE-2021-38508",
	"CVE-2021-38509",
	"CVE-2021-43534",
	"CVE-2021-43532",
	"CVE-2021-43531",
	"CVE-2021-43533",

	"CVE-2021-43538",
	"CVE-2021-43542",
	"CVE-2021-43543",
	"CVE-2021-30547",
	"CVE-2021-43546",
	"CVE-2021-43537",
	"CVE-2021-43541",
	"CVE-2021-43536",
	"CVE-2021-43545",
	"CVE-2021-43539",

	"CVE-2022-34480",
	"CVE-2022-26387",
	"CVE-2022-22759",
	"CVE-2022-28281",
	"CVE-2022-45415",
	"CVE-2022-42930",
	"CVE-2022-0511",
	"CVE-2022-22763",
	"CVE-2022-22737",
	"CVE-2022-22751",
	"CVE-2022-38478",
	"CVE-2022-22761",
	"CVE-2022-34482",
	"CVE-2022-26486",
	"CVE-2022-22739",
	"CVE-2022-22755",
	"CVE-2022-22757",
	"CVE-2022-1097",
	"CVE-2022-22754",
	"CVE-2022-22748",
	"CVE-2022-22736",
	"CVE-2022-22745",
	"CVE-2022-26385",
	"CVE-2022-26383",
	"CVE-2022-3266",
	"CVE-2022-34468",
	"CVE-2022-34481",
	"CVE-2022-28289",
	"CVE-2022-22741",
	"CVE-2022-28284",
	"CVE-2022-34484",
	"CVE-2022-22752",
	"CVE-2022-26485",
	"CVE-2022-28286",
	"CVE-2022-28283",
	"CVE-2022-28285",
	"CVE-2022-0843",
	"CVE-2022-29909",
	"CVE-2022-22749",
	"CVE-2022-26384",
	"CVE-2022-28282",
	"CVE-2022-28287",
	"CVE-2022-40956",
	"CVE-2022-22740",
	"CVE-2022-22743",
	"CVE-2022-22764",
	"CVE-2022-22738",
	"CVE-2022-1529",
	"CVE-2022-22760",
	"CVE-2022-29916",
	"CVE-2022-29917",
	"CVE-2022-22747",
	"CVE-2022-26382",
	"CVE-2022-22742",
	"CVE-2022-28288",
	"CVE-2022-22756",
	"CVE-2022-26381",
	"CVE-2022-1802",
	"CVE-2022-34483",
	"CVE-2022-29915",
}

var cvetests = []struct {
	cpe          string
	excludedCVEs []string
	includedCVEs []string
	// continuesToUpdate indicates if the product/software
	// continues to register new CVE vulnerabilities.
	continuestoUpdate bool
}{
	{
		cpe:               "cpe:2.3:a:1password:1password:3.9.9:*:*:*:*:macos:*:*",
		includedCVEs:      []string{"CVE-2012-6369"},
		continuestoUpdate: false,
	},
	{
		cpe:               "cpe:2.3:a:1password:1password:3.9.9:*:*:*:*:*:*:*",
		includedCVEs:      []string{"CVE-2012-6369"},
		continuestoUpdate: false,
	},
	{
		cpe: "cpe:2.3:a:pypa:pip:9.0.3:*:*:*:*:python:*:*",
		includedCVEs: []string{
			"CVE-2019-20916",
			"CVE-2021-3572",
		},
		continuestoUpdate: false,
	},
	{
		cpe:               "cpe:2.3:a:mozilla:firefox:93.0:*:*:*:*:windows:*:*",
		includedCVEs:      firefox93WindowsVulnerabilities,
		continuestoUpdate: true,
	},
	{
		cpe:               "cpe:2.3:a:mozilla:firefox:93.0.100:*:*:*:*:windows:*:*",
		includedCVEs:      firefox93WindowsVulnerabilities,
		continuestoUpdate: true,
	},
	{
		cpe: "cpe:2.3:a:apple:icloud:1.0:*:*:*:*:macos:*:*",
		excludedCVEs: []string{
			"CVE-2017-13797",
			"CVE-2017-2383",
			"CVE-2017-2366",
			"CVE-2016-4613",
			"CVE-2016-4692",
			"CVE-2016-4743",
			"CVE-2016-7578",
			"CVE-2016-7583",
			"CVE-2016-7586",
			"CVE-2016-7587",
			"CVE-2016-7589",
			"CVE-2016-7592",
			"CVE-2016-7598",
			"CVE-2016-7599",
			"CVE-2016-7610",
			"CVE-2016-7611",
			"CVE-2016-7614",
			"CVE-2016-7632",
			"CVE-2016-7635",
			"CVE-2016-7639",
			"CVE-2016-7640",
			"CVE-2016-7641",
			"CVE-2016-7642",
			"CVE-2016-7645",
			"CVE-2016-7646",
			"CVE-2016-7648",
			"CVE-2016-7649",
			"CVE-2016-7652",
			"CVE-2016-7654",
			"CVE-2016-7656",
			"CVE-2017-2383",
		},
		continuestoUpdate: true,
	},
}

func printMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

type threadSafeDSMock struct {
	mu sync.Mutex
	*mock.Store
}

func (d *threadSafeDSMock) ListSoftwareCPEs(ctx context.Context) ([]fleet.SoftwareCPE, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.Store.ListSoftwareCPEs(ctx)
}

func (d *threadSafeDSMock) InsertSoftwareVulnerability(ctx context.Context, vuln fleet.SoftwareVulnerability, src fleet.VulnerabilitySource) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.Store.InsertSoftwareVulnerability(ctx, vuln, src)
}

func TestTranslateCPEToCVE(t *testing.T) {
	nettest.Run(t)

	tempDir := t.TempDir()

	ds := new(mock.Store)
	ctx := context.Background()

	// download the CVEs once for all sub-tests, and then disable syncing
	err := nettest.RunWithNetRetry(t, func() error {
		return DownloadNVDCVEFeed(tempDir, "")
	})
	require.NoError(t, err)

	for _, tt := range cvetests {
		t.Run(tt.cpe, func(t *testing.T) {
			ds.ListSoftwareCPEsFunc = func(ctx context.Context) ([]fleet.SoftwareCPE, error) {
				return []fleet.SoftwareCPE{
					{CPE: tt.cpe},
				}, nil
			}

			cveLock := &sync.Mutex{}
			var cvesFound []string
			ds.InsertSoftwareVulnerabilityFunc = func(ctx context.Context, vuln fleet.SoftwareVulnerability, src fleet.VulnerabilitySource) (bool, error) {
				cveLock.Lock()
				defer cveLock.Unlock()
				cvesFound = append(cvesFound, vuln.CVE)
				return false, nil
			}
			ds.DeleteOutOfDateVulnerabilitiesFunc = func(ctx context.Context, source fleet.VulnerabilitySource, duration time.Duration) error {
				return nil
			}

			_, err := TranslateCPEToCVE(ctx, ds, tempDir, kitlog.NewLogfmtLogger(os.Stdout), false, 1*time.Hour)
			require.NoError(t, err)

			printMemUsage()

			if tt.continuestoUpdate {
				// Given that new vulnerabilities can be found on these
				// packages/products, we check that at least the
				// known ones are found.
				for _, cve := range tt.includedCVEs {
					require.Contains(t, cvesFound, cve, tt.cpe)
				}
			} else {
				// Check for exact match of CVEs found.
				require.ElementsMatch(t, cvesFound, tt.includedCVEs, tt.cpe)
			}

			for _, cve := range tt.excludedCVEs {
				require.NotContains(t, cvesFound, cve, tt.cpe)
			}

			require.True(t, ds.DeleteOutOfDateVulnerabilitiesFuncInvoked)
		})
	}

	t.Run("recent_vulns", func(t *testing.T) {
		safeDS := &threadSafeDSMock{Store: ds}

		softwareCPEs := []fleet.SoftwareCPE{
			{CPE: "cpe:2.3:a:google:chrome:-:*:*:*:*:*:*:*", ID: 1, SoftwareID: 1},
			{CPE: "cpe:2.3:a:mozilla:firefox:-:*:*:*:*:*:*:*", ID: 2, SoftwareID: 2},
			{CPE: "cpe:2.3:a:haxx:curl:-:*:*:*:*:*:*:*", ID: 3, SoftwareID: 3},
		}
		ds.ListSoftwareCPEsFunc = func(ctx context.Context) ([]fleet.SoftwareCPE, error) {
			return softwareCPEs, nil
		}

		ds.InsertSoftwareVulnerabilityFunc = func(ctx context.Context, vuln fleet.SoftwareVulnerability, src fleet.VulnerabilitySource) (bool, error) {
			return true, nil
		}
		recent, err := TranslateCPEToCVE(ctx, safeDS, tempDir, kitlog.NewNopLogger(), true, 1*time.Hour)
		require.NoError(t, err)

		byCPE := make(map[uint]int)
		for _, cpe := range recent {
			byCPE[cpe.Affected()]++
		}

		// even if it's somewhat far in the past, I've seen the exact numbers
		// change a bit between runs with different downloads, so allow for a bit
		// of wiggle room.
		assert.Greater(t, byCPE[softwareCPEs[0].SoftwareID], 150, "google chrome CVEs")
		assert.Greater(t, byCPE[softwareCPEs[1].SoftwareID], 280, "mozilla firefox CVEs")
		assert.Greater(t, byCPE[softwareCPEs[2].SoftwareID], 10, "curl CVEs")

		// call it again but now return false from this call, simulating CVE-CPE pairs
		// that already existed in the DB.
		ds.InsertSoftwareVulnerabilityFunc = func(ctx context.Context, vuln fleet.SoftwareVulnerability, src fleet.VulnerabilitySource) (bool, error) {
			return false, nil
		}
		recent, err = TranslateCPEToCVE(ctx, safeDS, tempDir, kitlog.NewNopLogger(), true, 1*time.Hour)
		require.NoError(t, err)

		// no recent vulnerability should be reported
		assert.Len(t, recent, 0)
	})
}

func TestSyncsCVEFromURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.RequestURI, ".meta") {
			fmt.Fprint(w, "lastModifiedDate:2021-08-04T11:10:30-04:00\r\n")
			fmt.Fprint(w, "size:20967174\r\n")
			fmt.Fprint(w, "zipSize:1453429\r\n")
			fmt.Fprint(w, "gzSize:1453293\r\n")
			fmt.Fprint(w, "sha256:10D7338A1E2D8DB344C381793110B67FCA7D729ADA21624EF089EBA78CCE7B53\r\n")
		}
	}))
	defer ts.Close()

	tempDir := t.TempDir()
	cveFeedPrefixURL := ts.URL + "/feeds/json/cve/1.1/"
	err := DownloadNVDCVEFeed(tempDir, cveFeedPrefixURL)
	require.Error(t, err)
	require.Contains(t,
		err.Error(),
		fmt.Sprintf("1 synchronisation error:\n\tunexpected size for \"%s/feeds/json/cve/1.1/nvdcve-1.1-2002.json.gz\" (200 OK): want 1453293, have 0", ts.URL),
	)
}
