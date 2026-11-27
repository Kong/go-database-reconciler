package konnect

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient spins up a stub server and returns a Client pointed at it,
// along with a pointer to the slice recording every request URI received.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *[]string) {
	t.Helper()

	var gotURIs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURIs = append(gotURIs, r.URL.RequestURI())
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(server.Client(), ClientOpts{BaseURL: server.URL})
	require.NoError(t, err)

	return client, &gotURIs
}

func TestRuntimeGroupServiceGetByName(t *testing.T) {
	client, gotURIs := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"data": [{"id": "a203e83d", "name": "fffffff"}],
			"meta": {"page": {"number": 1, "size": 100, "total": 1}}
		}`)
	})

	rg, err := client.RuntimeGroups.GetByName(context.Background(), "fffffff")
	require.NoError(t, err)
	require.NotNil(t, rg)
	assert.Equal(t, "a203e83d", *rg.ID)
	assert.Equal(t, "fffffff", *rg.Name)

	require.Len(t, *gotURIs, 1, "exact-name lookup must take a single request")
	assert.Equal(t,
		"/v2/control-planes?filter%5Bname%5D%5Beq%5D=fffffff&page%5Bsize%5D=100",
		(*gotURIs)[0],
		"must filter server-side via filter[name][eq]")
}

func TestRuntimeGroupServiceGetByNameNotFound(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data": [], "meta": {"page": {"number": 1, "size": 100, "total": 0}}}`)
	})

	rg, err := client.RuntimeGroups.GetByName(context.Background(), "nope")
	require.NoError(t, err)
	assert.Nil(t, rg)
}

// TestRuntimeGroupServiceListPagination guards against the regression where
// the v2 response envelope (meta.page.*) was read as the legacy top-level
// page/pageCount, which made every page look like the last one -- or, when
// the request used the legacy flat page/size params the v2 API ignores,
// made the caller loop on page 2 forever.
func TestRuntimeGroupServiceListPagination(t *testing.T) {
	const total = 250

	client, gotURIs := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// Serve the page the client actually asked for via page[number].
		// If the client sent the wrong param name we would always answer
		// page 1 here, and the loop below would never terminate.
		number := 1
		if v := r.URL.Query().Get("page[number]"); v != "" {
			_, err := fmt.Sscanf(v, "%d", &number)
			assert.NoError(t, err)
		}
		size := 100
		if v := r.URL.Query().Get("page[size]"); v != "" {
			_, err := fmt.Sscanf(v, "%d", &size)
			assert.NoError(t, err)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"data": [{"id": "id-%d", "name": "cp-%d"}],
			"meta": {"page": {"number": %d, "size": %d, "total": %d}}
		}`, number, number, number, size, total)
	})

	var all []*RuntimeGroup
	var opt *ListOpt
	for i := 0; ; i++ {
		require.Less(t, i, 10, "pagination did not terminate")

		groups, next, err := client.RuntimeGroups.List(context.Background(), opt)
		require.NoError(t, err)
		all = append(all, groups...)
		if next == nil {
			break
		}
		opt = next
	}

	require.Len(t, all, 3, "250 items at 100/page is 3 pages")
	assert.Equal(t, "cp-1", *all[0].Name)
	assert.Equal(t, "cp-3", *all[2].Name)

	require.Equal(t, []string{
		"/v2/control-planes?page%5Bsize%5D=100",
		"/v2/control-planes?page%5Bnumber%5D=2&page%5Bsize%5D=100",
		"/v2/control-planes?page%5Bnumber%5D=3&page%5Bsize%5D=100",
	}, *gotURIs, "must page with bracketed page[number]/page[size]")
}

func TestRuntimeGroupServiceListClampsPageSize(t *testing.T) {
	client, gotURIs := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data": [], "meta": {"page": {"number": 1, "size": 100, "total": 0}}}`)
	})

	_, _, err := client.RuntimeGroups.List(context.Background(), &ListOpt{Size: 500})
	require.NoError(t, err)

	require.Len(t, *gotURIs, 1)
	assert.Equal(t, "/v2/control-planes?page%5Bsize%5D=100", (*gotURIs)[0],
		"page[size] is capped at 100 by the API")
}
