package graphql

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/traggo/server/auth"
	"github.com/traggo/server/model"
	"github.com/traggo/server/test"
)

func TestHandler_jsonOverHtml(t *testing.T) {
	db := test.InMemoryDB(t)
	defer db.Close()
	resolver := NewResolver(db.DB, 4, model.Version{})
	handler := Handler("/gql", resolver, NewDirective())
	req := httptest.NewRequest("GET", "/gql?query="+url.QueryEscape("query {currentUser {name}}"), strings.NewReader(""))
	req.Header.Set("Accept", "text/html;application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	require.Equal(t, "application/graphql-response+json", recorder.Header().Get("Content-Type"))
	require.JSONEq(t, `
{
   "data": { "currentUser": null }
}
`, recorder.Body.String())
}

func TestHandler_htmlIfNotJson(t *testing.T) {
	db := test.InMemoryDB(t)
	defer db.Close()
	resolver := NewResolver(db.DB, 4, model.Version{})
	handler := Handler("/gql", resolver, NewDirective())
	req := httptest.NewRequest("get", "/gql", strings.NewReader(""))
	req.Header.Set("Accept", "text/html;application/xml")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	require.Contains(t, recorder.Body.String(), "Traggo Playground")
}

// TestHandler_stats_fractionalBoundsPreservedOnWire drives the real stats
// resolver through the HTTP GraphQL handler (with the auth middleware and a
// tagged span seeded in the database) and asserts the serialized JSON response
// preserves the accepted fractional range bounds exactly. A direct resolver
// result would not prove the wire contract, so this asserts the actual
// serialized strings and their decoded instants.
func TestHandler_stats_fractionalBoundsPreservedOnWire(t *testing.T) {
	db := test.InMemoryDB(t)
	defer db.Close()

	udb := db.User(1)
	udb.NewDevice(1, "token1", "device1")
	udb.TimeSpan("2019-06-11T10:00:00Z", "2019-06-11T10:00:10Z").Tag("type", "review")

	resolver := NewResolver(db.DB, 4, model.Version{})
	handler := auth.Middleware(db.DB)(Handler("/gql", resolver, NewDirective()))

	// One completed span 10:00:00Z-10:00:10Z clipped to a fractional range
	// [10:00:00.123456789Z, 10:00:01.123456789Z] => 1 second. The fractional
	// bounds must survive serialization byte-for-byte.
	query := `{ stats(ranges: [{start: "2019-06-11T10:00:00.123456789Z", end: "2019-06-11T10:00:01.123456789Z"}], tags: ["type"]) { start end entries { key value timeSpendInSeconds } } }`
	body := `{"query":` + strconvQuote(query) + `}`
	req := httptest.NewRequest("POST", "/gql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "traggo token1")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	require.Equal(t, 200, recorder.Code)
	require.JSONEq(t, `{
	  "data": {
	    "stats": [
	      {
	        "start": "2019-06-11T10:00:00.123456789Z",
	        "end": "2019-06-11T10:00:01.123456789Z",
	        "entries": [{ "key": "type", "value": "review", "timeSpendInSeconds": 1 }]
	      }
	    ]
	  }
	}`, recorder.Body.String())

	// Decode the JSON and verify the fractional start/end survive decoding as
	// the exact accepted instants (not truncated to whole seconds).
	var decoded struct {
		Data struct {
			Stats []struct {
				Start string `json:"start"`
				End   string `json:"end"`
			} `json:"stats"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &decoded))
	require.Len(t, decoded.Data.Stats, 1)
	gotStart, err := time.Parse(time.RFC3339Nano, decoded.Data.Stats[0].Start)
	require.NoError(t, err)
	gotEnd, err := time.Parse(time.RFC3339Nano, decoded.Data.Stats[0].End)
	require.NoError(t, err)
	wantStart, _ := time.Parse(time.RFC3339Nano, "2019-06-11T10:00:00.123456789Z")
	wantEnd, _ := time.Parse(time.RFC3339Nano, "2019-06-11T10:00:01.123456789Z")
	require.True(t, gotStart.Equal(wantStart), "decoded start %s != %s", gotStart, wantStart)
	require.True(t, gotEnd.Equal(wantEnd), "decoded end %s != %s", gotEnd, wantEnd)
}

// strconvQuote wraps a GraphQL query in a JSON string literal so it can be
// embedded in a request body without depending on the query content.
func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
