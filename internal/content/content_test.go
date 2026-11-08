package content

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadEmbedded(t *testing.T) {
	b, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("embedded content is invalid:\n%v", err)
	}
	e, ok := b.Era("e01")
	if !ok {
		t.Fatal("era e01 not found")
	}
	if len(e.Stations) != 10 {
		t.Errorf("era 1 should have 10 stations, has %d", len(e.Stations))
	}
	if got := b.Text("fr", e.NameKey); got != "Troc" {
		t.Errorf("fr name = %q, want Troc", got)
	}
	if got := b.Text("en", "missing.key"); got != "missing.key" {
		t.Errorf("missing key should fall back to the key, got %q", got)
	}
}

const minimalLocales = `{"era.x":"X","res.cur":"C","res.a":"A","st.s":"S","var.s.v1":"V1","var.s.v2":"V2",
"n.root":"R","n.t":"T","n.th":"TH","n.i":"I","n.c":"C","end":"E","pat":"P","pd":"PD"}`

func minimalEra(tree string) string {
	return `{"id":"x","index":1,"nameKey":"era.x","currency":"cur",
"resources":[{"id":"cur","nameKey":"res.cur","kind":"currency"},
             {"id":"a","nameKey":"res.a","kind":"base","basePrice":1,"elasticity":0.5,"demand":10}],
"stations":[{"id":"s","nameKey":"st.s","variants":[
   {"id":"v1","nameKey":"var.s.v1","recipe":{"outputs":{"a":1},"seconds":1}},
   {"id":"v2","nameKey":"var.s.v2","recipe":{"inputs":{"a":1},"outputs":{"cur":1},"seconds":1}}]}],
"tree":` + tree + `,"quests":[],"npcs":[],"contracts":[],"endings":[
 {"axis":"trade","nameKey":"end","patentKey":"pat","patentDescKey":"pd"},
 {"axis":"throughput","nameKey":"end","patentKey":"pat","patentDescKey":"pd"},
 {"axis":"ingenuity","nameKey":"end","patentKey":"pat","patentDescKey":"pd"},
 {"axis":"community","nameKey":"end","patentKey":"pat","patentDescKey":"pd"}]}`
}

const validTree = `[
 {"id":"root","nameKey":"n.root","branch":"trunk","row":1,"col":1,"cost":{"a":1}},
 {"id":"t","nameKey":"n.t","branch":"trade","row":6,"col":2,"cost":{"a":1},"requiresAll":["root"],"eraKey":true},
 {"id":"th","nameKey":"n.th","branch":"throughput","row":6,"col":2,"cost":{"a":1},"requiresAll":["root"],"eraKey":true},
 {"id":"i","nameKey":"n.i","branch":"ingenuity","row":6,"col":2,"cost":{"a":1},"requiresAll":["root"],"eraKey":true},
 {"id":"c","nameKey":"n.c","branch":"community","row":6,"col":2,"cost":{"a":1},"requiresAny":["root"],"eraKey":true}]`

func load(era, fr, en string) error {
	_, err := Load(fstest.MapFS{
		"eras/x.json":     {Data: []byte(era)},
		"locales/fr.json": {Data: []byte(fr)},
		"locales/en.json": {Data: []byte(en)},
	})
	return err
}

func TestValidMinimalEra(t *testing.T) {
	if err := load(minimalEra(validTree), minimalLocales, minimalLocales); err != nil {
		t.Fatal(err)
	}
}

func TestValidationErrors(t *testing.T) {
	cases := []struct {
		name, era, fr, want string
	}{
		{"missing translation", minimalEra(validTree), `{"era.x":"X"}`, `missing in locale "fr"`},
		{"unknown resource", strings.Replace(minimalEra(validTree), `"outputs":{"a":1}`, `"outputs":{"zz":1}`, 1), minimalLocales, `unknown resource "zz"`},
		{"missing era key", strings.Replace(minimalEra(validTree), `"requiresAny":["root"],"eraKey":true`, `"requiresAny":["root"]`, 1), minimalLocales, `branch "community" needs exactly one Era Key`},
		{"cycle", strings.Replace(minimalEra(validTree), `"branch":"trunk","row":1,"col":1,"cost":{"a":1}`, `"branch":"trunk","row":1,"col":1,"cost":{"a":1},"requiresAll":["t"]`, 1), minimalLocales, "cycle"},
		{"unknown field", strings.Replace(minimalEra(validTree), `"index":1`, `"index":1,"typo":2`, 1), minimalLocales, `unknown field "typo"`},
		{"cell outside", strings.Replace(minimalEra(validTree), `"branch":"trade","row":6`, `"branch":"trade","row":7`, 1), minimalLocales, `outside its 3x6 board`},
		{"ranked key", strings.Replace(minimalEra(validTree), `"branch":"trade","row":6,"col":2,`, `"branch":"trade","row":6,"col":2,"maxRank":3,`, 1), minimalLocales, `invalid maxRank`},
		{"row stations", strings.Replace(minimalEra(validTree), `"index":1`, `"index":1,"rowStations":[0,2]`, 1), minimalLocales, `rowStations[1] = 2`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := load(c.era, c.fr, minimalLocales)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want error containing %q, got %v", c.want, err)
			}
		})
	}
}

func TestRankCostsAddUpToTheNodeCost(t *testing.T) {
	n := Node{Cost: Amounts{"a": 150}, MaxRank: 5}
	total := 0.0
	for r := 1; r <= n.Ranks(); r++ {
		total += n.RankCost(r)["a"]
	}
	if total < 149.999 || total > 150.001 {
		t.Fatalf("ranks add up to %v, want 150", total)
	}
	if first, last := n.RankCost(1)["a"], n.RankCost(5)["a"]; first != 10 || last != 50 {
		t.Fatalf("rank 1 costs %v and rank 5 costs %v, want 10 and 50", first, last)
	}
	if single := (&Node{Cost: Amounts{"a": 7}}).RankCost(1)["a"]; single != 7 {
		t.Fatalf("a single-rank node costs %v, want 7", single)
	}
}
