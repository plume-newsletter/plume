package httpapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/plume-newsletter/plume/internal/store/gen"
)

func TestCampaignRespEncodesBodyJsonAsJSONString(t *testing.T) {
	c := gen.Campaign{BodyJson: []byte(`[{"type":"heading","text":"Hi"}]`)}
	out, err := json.Marshal(campaignToResp(c))
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `"body_json":"[{`) {
		t.Fatalf("body_json should be a JSON-array string, got: %s", s)
	}
	// The field value must itself be a JSON string that parses into an array.
	var resp map[string]json.RawMessage
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	var inner string
	if err := json.Unmarshal(resp["body_json"], &inner); err != nil {
		t.Fatalf("body_json is not a JSON string (likely base64 []byte): %v", err)
	}
	var arr []map[string]any
	if err := json.Unmarshal([]byte(inner), &arr); err != nil {
		t.Fatalf("body_json string is not a JSON array: %v", err)
	}
	if len(arr) != 1 {
		t.Fatalf("want 1 block, got %d", len(arr))
	}
}
