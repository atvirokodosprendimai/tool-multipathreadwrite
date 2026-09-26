package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// ADR-067 T4. mrw is dual-era: a request carrying
// _meta["io.modelcontextprotocol/protocolVersion"] is served per request under
// the 2026-07-28 rules, and a request without it exactly as before.

const modernMeta = `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}`

// session runs requests through one Serve bound to root and returns one decoded
// answer per line written.
func session(t *testing.T, root string, requests ...string) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := Serve(strings.NewReader(strings.Join(requests, "\n")+"\n"), &out, root); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	var got []map[string]any
	for _, l := range strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n") {
		if l != "" {
			got = append(got, decode(t, l))
		}
	}
	return got
}

// rawResultOf returns the bytes of one response's result member as sent.
func rawResultOf(t *testing.T, root, req string) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := Serve(strings.NewReader(req+"\n"), &out, root); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	var resp struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp); err != nil {
		t.Fatalf("response is not JSON: %v\n%s", err, out.String())
	}
	if len(resp.Error) > 0 {
		t.Fatalf("a JSON-RPC error: %s", resp.Error)
	}
	return resp.Result
}

func callReq(tool, args, meta string) string {
	if meta != "" {
		meta = "," + meta
	}
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s%s}}`, tool, args, meta)
}

func errCode(t *testing.T, resp map[string]any) (float64, map[string]any) {
	t.Helper()
	e, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("want a JSON-RPC error, got %v", resp)
	}
	c, _ := e["code"].(float64)
	return c, e
}

func servesMrw(res map[string]any) bool {
	meta, _ := res["_meta"].(map[string]any)
	info, _ := meta["io.modelcontextprotocol/serverInfo"].(map[string]any)
	return info["name"] == "mrw"
}

var wantVersions = []any{"2026-07-28", "2025-11-25", "2025-06-18"}

func TestServerDiscoverNamesEverySupportedVersion(t *testing.T) {
	root, _ := checkout(t, "a.txt", "one\n")
	init := session(t, root, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`)[0]["result"].(map[string]any)
	for _, v := range []string{"2026-07-28", "2025-11-25"} {
		req := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":%q,"io.modelcontextprotocol/clientCapabilities":{}}}}`, v)
		res, ok := session(t, root, req)[0]["result"].(map[string]any)
		if !ok {
			t.Fatalf("discover naming %s answered no result", v)
		}
		if fmt.Sprint(res["supportedVersions"]) != fmt.Sprint(wantVersions) {
			t.Errorf("naming %s: supportedVersions = %v, want %v", v, res["supportedVersions"], wantVersions)
		}
		caps, _ := res["capabilities"].(map[string]any)
		if _, ok := caps["tools"]; !ok || !servesMrw(res) || res["resultType"] != "complete" || res["cacheScope"] != "public" {
			t.Errorf("naming %s: discover is not modern-shaped: %v", v, res)
		}
		if ttl, ok := res["ttlMs"].(float64); !ok || ttl < 0 {
			t.Errorf("naming %s: ttlMs = %v", v, res["ttlMs"])
		}
		if res["instructions"] != init["instructions"] {
			t.Errorf("naming %s: discover's instructions differ from initialize's", v)
		}
	}
}

func TestAModernRequestForAnUnknownVersionIsRefusedWithItsSupportedList(t *testing.T) {
	root, _ := checkout(t, "a.txt", "one\n")
	got := session(t, root, `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"1900-01-01","io.modelcontextprotocol/clientCapabilities":{}}}}`)[0]
	code, e := errCode(t, got)
	if code != -32022 {
		t.Fatalf("an unknown version answered code %v, want -32022", code)
	}
	data, _ := e["data"].(map[string]any)
	if data["requested"] != "1900-01-01" || fmt.Sprint(data["supported"]) != fmt.Sprint(wantVersions) {
		t.Errorf("-32022 data = %v", data)
	}
	// A client retries with a listed version and must be answered, or the
	// refusal is a loop. Decorated only for the modern version.
	for _, v := range wantVersions {
		req := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":%q,"io.modelcontextprotocol/clientCapabilities":{}}}}`, v)
		res, ok := session(t, root, req)[0]["result"].(map[string]any)
		if !ok {
			t.Errorf("retry with listed version %v was not answered", v)
			continue
		}
		if decorated := res["resultType"] == "complete"; decorated != (v == "2026-07-28") {
			t.Errorf("retry with %v: decorated=%v", v, decorated)
		}
	}
}

func TestAModernRequestWithoutClientCapabilitiesIsInvalidParams(t *testing.T) {
	root, _ := checkout(t, "a.txt", "one\n")
	got := session(t, root, `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`)[0]
	if code, _ := errCode(t, got); code != float64(codeInvalidParams) {
		t.Errorf("a modern request without clientCapabilities answered %v, want %d", code, codeInvalidParams)
	}
}

func TestAModernResultCarriesResultTypeAndServerInfo(t *testing.T) {
	root, _ := checkout(t, "a.txt", "hello\n")
	res, ok := session(t, root, callReq("mrw_read", `{"specs":["a.txt"]}`, modernMeta))[0]["result"].(map[string]any)
	if !ok {
		t.Fatal("a modern tools/call answered no result")
	}
	if res["resultType"] != "complete" || !servesMrw(res) {
		t.Errorf("a modern tools/call is not decorated: %v", res)
	}
	if !strings.Contains(served0(t, res), "    1| hello") {
		t.Errorf("content[0] is not the served text: %q", served0(t, res))
	}
}

func TestAModernToolsListCarriesCachingHints(t *testing.T) {
	root, _ := checkout(t, "a.txt", "one\n")
	got := session(t, root,
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{`+modernMeta+`}}`)
	legacy, _ := got[0]["result"].(map[string]any)
	modern, _ := got[1]["result"].(map[string]any)
	if ttl, ok := modern["ttlMs"].(float64); !ok || ttl < 0 || modern["cacheScope"] != "public" || modern["resultType"] != "complete" {
		t.Errorf("a modern tools/list lacks its caching hints: ttlMs=%v cacheScope=%v resultType=%v", modern["ttlMs"], modern["cacheScope"], modern["resultType"])
	}
	if fmt.Sprint(modern["tools"]) != fmt.Sprint(legacy["tools"]) {
		t.Error("the modern tool list differs from the legacy one")
	}
}

var ckID = regexp.MustCompile(`\b[0-9a-f]{16}\b`)

// legacyTranscript is every legacy answer the golden pins, normalised: the
// checkout path and the random checkpoint ids (ack.go) are the only volatile
// bytes, and nothing else is rewritten.
func legacyTranscript(t *testing.T) string {
	t.Helper()
	root, _ := checkout(t, "a.txt", "alpha\nbravo\n")
	var out bytes.Buffer
	reqs := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"ping"}`,
		strings.Replace(callReq("mrw_read", `{"specs":["a.txt"]}`, ""), `"id":1`, `"id":4`, 1),
		strings.Replace(callReq("mrw_read", `{}`, ""), `"id":1`, `"id":5`, 1),
		strings.Replace(callReq("mrw_write", `{"plan":"@@ a.txt 1 replace\nX\n","dry_run":true}`, ""), `"id":1`, `"id":6`, 1),
	}
	if err := Serve(strings.NewReader(strings.Join(reqs, "\n")+"\n"), &out, root); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	// The root appears raw, JSON-escaped once (structuredContent.root) and
	// twice (the JSON text block); on Windows the backslashes differ in each,
	// so all three forms are replaced, the longest first (Codex on #212).
	s := out.String()
	paths := []string{root}
	if real, err := filepath.EvalSymlinks(root); err == nil {
		paths = append(paths, real)
	}
	for _, p := range paths {
		once := jsonInner(p)
		for _, form := range []string{jsonInner(once), once, p} {
			s = strings.ReplaceAll(s, form, "<root>")
		}
	}
	return ckID.ReplaceAllString(s, "<ck>")
}

// jsonInner is s as it appears inside a JSON string, without the quotes.
func jsonInner(s string) string {
	b, _ := json.Marshal(s)
	return string(b[1 : len(b)-1])
}

// A guard, captured on T1–T3's tree before any T4 edit: legacy answers are
// byte-identical after normalisation, and none carries resultType.
// MRW_UPDATE_LEGACY_GOLDEN=1 rewrites the file. It was done before ADR-067 T4, once by ADR-070,
// whose worked plan gained " body=4": the diff was that token and nothing else, and once by
// ADR-075, whose routing stopped selling serialized writes: the diff was that sentence and the two
// tool descriptions' matching clause; and once by ADR-076, whose receipt gained `target` and
// `dirs_created` and whose `hunks.path` stopped claiming to be "as written": the schema, nothing else.
func TestALegacyResultIsUnchangedByTheModernPath(t *testing.T) {
	got := legacyTranscript(t)
	golden := filepath.Join("testdata", "legacy_golden.jsonl")
	if os.Getenv("MRW_UPDATE_LEGACY_GOLDEN") != "" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("a legacy answer changed:\n got: %s\nwant: %s", got, want)
	}
	if strings.Contains(got, `"resultType"`) {
		t.Error("a legacy answer carries resultType")
	}
}

// decorationSize measures what the modern era adds to a tools/call result, on
// the wire, from a deterministic refusal answered in both eras.
func decorationSize(t *testing.T, root string) int {
	t.Helper()
	legacy := rawResultOf(t, root, callReq("mrw_read", `{}`, ""))
	modern := rawResultOf(t, root, callReq("mrw_read", `{}`, modernMeta))
	d := len(modern) - len(legacy)
	if d <= 0 {
		t.Fatalf("the modern era adds %d bytes; nothing to reserve", d)
	}
	return d
}

func TestAModernReadStaysWithinTheCeiling(t *testing.T) {
	// A whole-file read that fits undecorated: only an open-ended spec can
	// degrade to a page (firstPage), so a closed range would be refused instead.
	root, name, _ := ceilingFixture(t, 200)
	d := decorationSize(t, root)
	legacy := rawResultOf(t, root, callReq("mrw_read", fmt.Sprintf(`{"specs":[%q]}`, name), ""))
	// A ceiling the undecorated read fits under and the decorated one would not.
	withCeiling(t, len(legacy)+d/2)
	raw := rawResultOf(t, root, callReq("mrw_read", fmt.Sprintf(`{"specs":[%q]}`, name), modernMeta))
	if len(raw) > MaxResultChars {
		t.Fatalf("the modern read is %d bytes against a %d ceiling", len(raw), MaxResultChars)
	}
	var res map[string]any
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatal(err)
	}
	if res["resultType"] != "complete" {
		t.Errorf("the degraded modern answer is not decorated: %v", res)
	}
	text := served0(t, res)
	if !strings.Contains(text, "-- PARTIAL:") {
		t.Fatalf("the modern read did not degrade to a page:\n%.400s", text)
	}
	// The page holds pending checkpoints; acknowledging them licenses a write.
	withCeiling(t, DefaultMaxResultChars)
	ack, _ := json.Marshal(checkpointsIn(text))
	w := rawResultOf(t, root, callReq("mrw_write", fmt.Sprintf(`{"plan":"@@ %s 1 replace\nX\n","ack":%s}`, name, ack), modernMeta))
	if strings.Contains(string(w), `"isError":true`) {
		t.Errorf("a write to an acknowledged line of the modern page was refused: %s", w)
	}
}

func TestAModernWriteReceiptIsBudgetedWithItsDecoration(t *testing.T) {
	root, name, planText := ceilingFixture(t, 300)
	d := decorationSize(t, root)
	// The read must be ACKNOWLEDGED: an MCP serve licenses nothing until then.
	read := rawResultOf(t, root, callReq("mrw_read", fmt.Sprintf(`{"specs":[%q]}`, name), ""))
	var readRes map[string]any
	if err := json.Unmarshal(read, &readRes); err != nil {
		t.Fatal(err)
	}
	plan := strings.Replace(planText, "line 00001 of the fixture", "MUTATED", 1)
	args, _ := json.Marshal(map[string]any{"plan": plan, "ack": checkpointsIn(served0(t, readRes))})
	// The floor message prints the ceiling, so the floor depends on the
	// ceiling's digit count: iterate to the ceiling that sits exactly one byte
	// under its own decorated floor.
	withCeiling(t, 1000) // its cleanup restores the default after the loop below
	c := 1000
	for i := 0; i < 8; i++ {
		MaxResultChars = c
		next := writeFloor() + d - 1
		if next == c {
			break
		}
		c = next
	}
	floor := c + 1 - d

	// Below the decorated floor: refused before applying, the tree unchanged.
	withCeiling(t, floor+d-1)
	got := session(t, root, callReq("mrw_write", string(args), modernMeta))[0]
	if _, refused := got["error"]; !refused {
		t.Errorf("a modern write below its decorated floor was not refused: %v", got)
	}
	if b, _ := os.ReadFile(filepath.Join(root, name)); strings.Contains(string(b), "MUTATED") {
		t.Fatal("a modern write refused for its ceiling changed the tree")
	}

	// The refusal names the ceiling that works: retrying at exactly that
	// number applies, and the answer — too small for the receipt — still says
	// the write happened (Codex on #212: a generic refusal passed this before).
	e, _ := got["error"].(map[string]any)
	msg, _ := e["message"].(string)
	named := regexp.MustCompile(`at least (\d+)`).FindStringSubmatch(msg)
	if named == nil {
		t.Fatalf("the refusal names no ceiling to retry at: %q", msg)
	}
	c2, _ := strconv.Atoi(named[1])
	withCeiling(t, c2)
	raw := rawResultOf(t, root, callReq("mrw_write", string(args), modernMeta))
	if len(raw) > MaxResultChars {
		t.Fatalf("the modern write answered %d bytes against a %d ceiling", len(raw), MaxResultChars)
	}
	if b, _ := os.ReadFile(filepath.Join(root, name)); !strings.Contains(string(b), "MUTATED") {
		t.Fatalf("a modern write at the ceiling its refusal named (%d) did not apply", c2)
	}
	if !strings.Contains(string(raw), "the write HAPPENED") || !strings.Contains(string(raw), `"resultType":"complete"`) {
		t.Errorf("the modern write's answer does not say it applied, or is undecorated: %s", raw)
	}

	// Where the undecorated receipt fits and the decorated one would not, the
	// answer is still the receipt — elided, applied — never a refusal. This is
	// the case that proves receipt sizing reads the reserve.
	withCeiling(t, DefaultMaxResultChars)
	legacy := rawResultOf(t, root, callReq("mrw_write", string(args), ""))
	withCeiling(t, len(legacy)+d-1)
	modern := rawResultOf(t, root, callReq("mrw_write", string(args), modernMeta))
	if len(modern) > MaxResultChars {
		t.Fatalf("the modern receipt is %d bytes against a %d ceiling", len(modern), MaxResultChars)
	}
	if strings.Contains(string(modern), `"isError":true`) || !strings.Contains(string(modern), `"applied":true`) {
		t.Errorf("a modern write whose undecorated receipt fits was not answered with its receipt: %.400s", modern)
	}
}

func TestTheEraPrecedenceIsFixed(t *testing.T) {
	root, _ := checkout(t, "a.txt", "one\n")
	// initialize is always legacy, whatever _meta it carries.
	init := session(t, root, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25",`+modernMeta+`}}`)[0]
	res, _ := init["result"].(map[string]any)
	if res["protocolVersion"] != "2025-11-25" || res["resultType"] != nil {
		t.Errorf("initialize with modern _meta did not answer the legacy shape: %v", init)
	}
	// A notification is never answered, before any _meta is read.
	quiet := session(t, root,
		`{"jsonrpc":"2.0","method":"notifications/initialized","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"1900-01-01"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`)
	if len(quiet) != 0 {
		t.Errorf("a notification was answered: %v", quiet)
	}
	// Both eras interleaved on one Serve each answer in their own shape.
	mixed := session(t, root,
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{`+modernMeta+`}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`)
	for i, want := range []bool{false, true, false} {
		r, _ := mixed[i]["result"].(map[string]any)
		if (r["resultType"] == "complete") != want {
			t.Errorf("answer %d: decorated=%v, want %v", i+1, r["resultType"] == "complete", want)
		}
	}
}
