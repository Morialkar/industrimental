package sim

import (
	"math"
	"testing"

	"github.com/morialkar/industrimental/internal/content"
	"github.com/morialkar/industrimental/internal/save"
)

func era1(t *testing.T) *content.Era {
	t.Helper()
	b, err := content.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	e, ok := b.Era("e01")
	if !ok {
		t.Fatal("era e01 missing")
	}
	return e
}

// chain is a small economy with one bottleneck: a mine feeds a forge that eats
// faster than the mine produces.
func chain() *content.Era {
	return &content.Era{
		ID: "t", Currency: "cur",
		Resources: []content.Resource{
			{ID: "cur", Kind: content.KindCurrency},
			{ID: "ore", Kind: content.KindBase, BasePrice: 2, Elasticity: 1, Demand: 100},
			{ID: "bar", Kind: content.KindCritical, BasePrice: 10, Elasticity: 1, Demand: 100},
		},
		Stations: []content.Station{
			{ID: "mine", LevelCost: content.Amounts{"ore": 1}, Variants: []content.Variant{
				{ID: "a", Recipe: content.Recipe{Outputs: content.Amounts{"ore": 1}, Seconds: 1}},
			}},
			{ID: "forge", RetoolSeconds: 100, UnlockCost: content.Amounts{"ore": 10}, Variants: []content.Variant{
				{ID: "a", Recipe: content.Recipe{Inputs: content.Amounts{"ore": 4}, Outputs: content.Amounts{"bar": 1}, Seconds: 1}},
				{ID: "b", Recipe: content.Recipe{Inputs: content.Amounts{"ore": 2}, Outputs: content.Amounts{"bar": 1}, Seconds: 1},
					Effects: []content.Effect{{Kind: content.EffectResourceRate, Resource: "ore", Value: 1}}},
			}},
		},
	}
}

func near(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %g, want %g (±%g)", name, got, want, tol)
	}
}

func TestBottleneckLimitsTheChain(t *testing.T) {
	s := New(chain(), 1)
	if err := s.Build("mine", "a"); err != nil {
		t.Fatal(err)
	}
	s.Step(10)
	near(t, "ore after 10 s", s.Stock("ore"), 10, 1e-9)
	if err := s.Build("forge", "a"); err != nil {
		t.Fatal(err)
	}
	near(t, "ore after building", s.Stock("ore"), 0, 1e-9)
	s.Step(100)
	// 1 ore/s feeds a forge that wants 4: it runs at a quarter of its speed.
	near(t, "bars", s.Stock("bar"), 25, 1e-6)
	near(t, "ore", s.Stock("ore"), 0, 1e-6)
	st := s.Station("forge")
	near(t, "forge speed", st.Speed, 0.25, 1e-6)
	if st.Limiting != "ore" {
		t.Errorf("limiting = %q, want ore", st.Limiting)
	}
}

func TestStockRunsOutMidStep(t *testing.T) {
	s := New(chain(), 1)
	if err := s.Build("mine", "a"); err != nil {
		t.Fatal(err)
	}
	s.Step(110)
	if err := s.Build("forge", "a"); err != nil { // leaves 100 ore
		t.Fatal(err)
	}
	// Net -3 ore/s: the stock lasts 33.3 s at full speed, then a quarter speed.
	s.Step(100)
	want := 100.0/3 + (100-100.0/3)/4
	near(t, "bars", s.Stock("bar"), want, 1e-6)
}

func TestOfflineMatchesLivePlay(t *testing.T) {
	setup := func() *Sim {
		s := New(chain(), 1)
		if err := s.Build("mine", "a"); err != nil {
			t.Fatal(err)
		}
		s.Step(60)
		if err := s.Build("forge", "a"); err != nil {
			t.Fatal(err)
		}
		return s
	}
	live, off := setup(), setup()
	for i := 0; i < 36000; i++ { // one hour of live ticks
		live.Tick()
	}
	rep := off.Offline(3600, DefaultOfflineCap)
	near(t, "ore", off.Stock("ore"), live.Stock("ore"), 1e-6)
	near(t, "bars", off.Stock("bar"), live.Stock("bar"), 1e-6)
	if rep.Counted != 3600 || rep.Gains["bar"] <= 0 {
		t.Errorf("unexpected report: %+v", rep)
	}
}

func TestOfflineIsCapped(t *testing.T) {
	s := New(chain(), 1)
	if err := s.Build("mine", "a"); err != nil {
		t.Fatal(err)
	}
	rep := s.Offline(24*3600, DefaultOfflineCap)
	near(t, "counted", rep.Counted, DefaultOfflineCap, 0)
	near(t, "ore", s.Stock("ore"), DefaultOfflineCap, 1e-6)
	near(t, "clock follows real time", s.Clock(), 24*3600, 0)
}

func TestMarketPriceFormula(t *testing.T) {
	s := New(chain(), 1)
	base := s.Price("bar")
	near(t, "price within the trend", base, 10, 1.5+1e-9)

	s.stock[s.resIdx["bar"]] = 1000
	lot, err := s.Sell("bar", 100)
	if err != nil {
		t.Fatal(err)
	}
	if lot >= base*100 {
		t.Errorf("selling in bulk should pay less than the list price: %g", lot)
	}
	if after := s.Price("bar"); after >= base*0.65 {
		t.Errorf("the price should fall after a big sale: %g from %g", after, base)
	}
	// S halves every hour.
	sold := s.state.Sold["bar"]
	s.Step(3600)
	near(t, "S after one hour", s.state.Sold["bar"], sold/2, 1e-6)
}

func TestNoProfitableRoundTrip(t *testing.T) {
	s := New(chain(), 1)
	s.stock[s.resIdx["bar"]] = 500
	if _, err := s.Sell("bar", 500); err != nil {
		t.Fatal(err)
	}
	s.stock[s.resIdx["cur"]] = 1e9
	before := s.Stock("cur")
	if err := s.Buy("bar", 200); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Sell("bar", 200); err != nil {
		t.Fatal(err)
	}
	if s.Stock("cur") >= before {
		t.Errorf("buying then selling should lose money: %g -> %g", before, s.Stock("cur"))
	}
}

func TestLevelsCostsAndMultiplier(t *testing.T) {
	near(t, "level 24", LevelMultiplier(24), 24, 0)
	near(t, "level 25 doubles", LevelMultiplier(25), 50, 0)
	near(t, "level 50 doubles twice", LevelMultiplier(50), 200, 0)

	s := New(chain(), 1)
	if err := s.Build("mine", "a"); err != nil {
		t.Fatal(err)
	}
	near(t, "first level", s.UpgradeCost("mine", 1)["ore"], 1, 1e-9)
	near(t, "two levels", s.UpgradeCost("mine", 2)["ore"], 2.15, 1e-9)
	s.stock[s.resIdx["ore"]] = 2.15
	if got := s.MaxUpgrades("mine"); got != 2 {
		t.Errorf("max upgrades = %d", got)
	}
	if err := s.Upgrade("mine", 2); err != nil {
		t.Fatal(err)
	}
	near(t, "level", float64(s.Station("mine").Level), 3, 0)
}

func TestRetoolStopsTheStationThenAppliesEffects(t *testing.T) {
	s := New(chain(), 1)
	if err := s.Build("mine", "a"); err != nil {
		t.Fatal(err)
	}
	s.Step(1000)
	if err := s.Build("forge", "a"); err != nil {
		t.Fatal(err)
	}
	if err := s.Retool("forge", "b"); err != nil {
		t.Fatal(err)
	}
	near(t, "half the unlock cost", s.Stock("ore"), 990-5, 1e-9)
	bars := s.Stock("bar")
	s.Step(50)
	near(t, "idle during the works", s.Stock("bar"), bars, 1e-9)
	if s.Station("forge").Retooling <= 0 {
		t.Error("should still be retooling")
	}
	s.Step(60)
	if s.Stock("bar") <= bars {
		t.Error("should run again after the works")
	}
	// Variant b doubles the ore: 2 ore/s, all eaten or stored, and it applies now.
	near(t, "effect active", s.producedRate("ore"), 2, 1e-9)
}

func TestSaveRestoreKeepsTheGame(t *testing.T) {
	s := New(chain(), 1)
	if err := s.Build("mine", "a"); err != nil {
		t.Fatal(err)
	}
	s.Step(500)
	r := Restore(s.Era, s.State())
	r.Step(100)
	s.Step(100)
	near(t, "ore", r.Stock("ore"), s.Stock("ore"), 1e-9)
	near(t, "clock", r.Clock(), s.Clock(), 1e-9)
}

func TestEra1OpensLikeTheDesign(t *testing.T) {
	e := era1(t)
	if len(e.Stations) != 10 || len(e.Tree) != 50 {
		t.Errorf("stations = %d, nodes = %d", len(e.Stations), len(e.Tree))
	}
	s := New(e, 1)
	if err := s.Trade("bob", "silex"); err != ErrAfford {
		t.Errorf("no berries yet: %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, err := s.Harvest("cueillette"); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Trade("bob", "silex"); err != nil {
		t.Fatal(err)
	}
	if s.Stock("silex") != 1 {
		t.Errorf("silex = %g", s.Stock("silex"))
	}
	s.Step(1)
	if !s.QuestDone("premier-troc") {
		t.Error("the first barter should close the first quest")
	}
	if err := s.Build("carriere-silex", "silex-fin"); err != nil {
		t.Fatal(err)
	}
	if err := s.Build("feu-camp", "fumoir"); err != ErrLocked {
		t.Errorf("the campfire needs a tree node: %v", err)
	}
	if err := s.BuyNode("le-cercle"); err != ErrLocked {
		t.Errorf("le-cercle needs the trunk: %v", err)
	}
	if s.MarketOpen() {
		t.Error("the market opens with the barter circle")
	}
}

func TestAtMostThreeMainQuests(t *testing.T) {
	s := New(era1(t), 1)
	main := 0
	for _, q := range s.ActiveQuests() {
		if q.Main {
			main++
		}
	}
	if main != 3 {
		t.Errorf("active main quests = %d", main)
	}
}

func TestMissedContractComesBack(t *testing.T) {
	s := New(era1(t), 1)
	var q *content.Quest
	for i := range s.Era.Quests {
		if s.Era.Quests[i].Kind == content.QuestContract && s.Era.Quests[i].Main {
			q = &s.Era.Quests[i]
			break
		}
	}
	s.state.Quests = append(s.state.Quests, q.Requires...)
	for i := range s.Era.Quests {
		if q2 := &s.Era.Quests[i]; q2 == q {
			break
		} else {
			s.state.Quests = append(s.state.Quests, q2.ID)
		}
	}
	s.updateQuests()
	if _, running := s.QuestDeadline(q.ID); !running {
		t.Fatal("the delay should run once the contract is active")
	}
	s.state.Clock += q.Seconds + 1
	s.updateQuests()
	if s.QuestDone(q.ID) {
		t.Fatal("a missed contract is not done")
	}
	for _, a := range s.ActiveQuests() {
		if a == q {
			t.Fatal("a missed contract should wait before coming back")
		}
	}
	s.state.Clock += contractRetryDelay + 1
	s.updateQuests()
	found := false
	for _, a := range s.ActiveQuests() {
		found = found || a == q
	}
	if !found {
		t.Error("a missed main contract should come back")
	}
}

func TestDeliverContractPaysAndCloses(t *testing.T) {
	s := New(era1(t), 7)
	s.stock[s.resIdx["baies"]] = 100
	s.stock[s.resIdx["silex"]] = 100
	if err := s.Build("cercle-troc", "conseil"); err != nil {
		t.Fatal(err)
	}
	s.Step(1)
	open := s.Contracts()
	if len(open) == 0 {
		t.Fatal("the market should offer a contract")
	}
	c := open[0]
	s.stock[s.resIdx[c.Template.Resource]] += c.Amount
	before := s.Stock(s.Era.Currency)
	if err := s.DeliverContract(c.Index); err != nil {
		t.Fatal(err)
	}
	if s.Stock(s.Era.Currency)-before < c.Reward-1e-9 {
		t.Error("the contract should pay its reward")
	}
	if len(s.Contracts()) != len(open)-1 {
		t.Error("the contract should be closed")
	}
}

func TestRanksScaleEffectsAndCosts(t *testing.T) {
	e := era1(t)
	s := New(e, 1)
	s.Grant("baies", 1e6)
	s.Grant("silex", 1e6)
	n := s.nodes["main-habile"]
	if n.Ranks() != 5 {
		t.Fatalf("main-habile has %d ranks, want 5", n.Ranks())
	}
	if got := s.NextRankCost("main-habile")["baies"]; math.Abs(got-n.Cost["baies"]/15) > 1e-9 {
		t.Errorf("rank 1 costs %g baies, want %g", got, n.Cost["baies"]/15)
	}
	if err := s.BuyNode("main-habile"); err != nil {
		t.Fatal(err)
	}
	if got := s.NextRankCost("main-habile")["baies"]; math.Abs(got-2*n.Cost["baies"]/15) > 1e-9 {
		t.Errorf("rank 2 costs %g baies, want %g", got, 2*n.Cost["baies"]/15)
	}
	if got, want := s.HarvestShare(), e.HarvestShare+0.05; math.Abs(got-want) > 1e-9 {
		t.Errorf("harvest share after one rank = %g, want %g", got, want)
	}
	if err := s.BuyNodeFully("main-habile"); err != nil {
		t.Fatal(err)
	}
	if !s.NodeMaxed("main-habile") || s.BuyNode("main-habile") != ErrBuilt {
		t.Error("main-habile should be maxed")
	}
	if got, want := s.HarvestShare(), e.HarvestShare+0.25; math.Abs(got-want) > 1e-9 {
		t.Errorf("harvest share at rank 5 = %g, want %g", got, want)
	}
}

func TestOldSavesKeepTheirNodesFullyRanked(t *testing.T) {
	e := era1(t)
	s := Restore(e, &save.EraState{Nodes: []string{"main-habile"}})
	if !s.NodeMaxed("main-habile") {
		t.Errorf("an old save's node should be maxed, rank = %d", s.Rank("main-habile"))
	}
}

func TestRowsOpenWithBuiltStations(t *testing.T) {
	e := era1(t)
	s := New(e, 1)
	for _, id := range []string{"baies", "silex", "bois", "coquillages", "sel", "poteries-crues", "outils-silex", "peaux-tannees", "parures"} {
		s.Grant(id, 1e7)
	}
	for _, id := range []string{"main-habile", "feu-apprivoise", "premiers-paniers", "langage-troc", "sedentarisation", "le-cercle"} {
		if err := s.BuyNode(id); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
	}
	if !s.NodeAvailable("foires") {
		t.Error("row 1 is open from the start")
	}
	if s.NodeAvailable("marche-grains") {
		t.Error("row 2 needs 2 stations")
	}
	_ = s.Build("cueillette", "juteuses")
	_ = s.Build("carriere-silex", "silex-fin")
	if err := s.BuyNode("foires"); err != nil {
		t.Fatal(err)
	}
	if !s.RowOpen(2) || !s.NodeAvailable("marche-grains") {
		t.Errorf("row 2 should open with %d stations", s.StationsBuilt())
	}
}

func TestHarvestGrowsWithProduction(t *testing.T) {
	e := era1(t)
	s := New(e, 1)
	got, err := s.Harvest("cueillette")
	if err != nil || got["baies"] != 1 {
		t.Fatalf("first tap = %v, %v; want 1 baie", got, err)
	}
	for _, r := range e.Resources {
		s.Grant(r.ID, 1e12)
	}
	if err := s.Build("cueillette", "juteuses"); err != nil {
		t.Fatal(err)
	}
	if err := s.Upgrade("cueillette", 60); err != nil {
		t.Fatal(err)
	}
	s.Step(1)
	rate := 0.0
	for _, r := range e.Resources {
		if r.Kind != content.KindCurrency && s.Rate(r.ID) > 0 {
			rate += s.Rate(r.ID)
		}
	}
	want := math.Round(e.HarvestShare*rate*10) / 10
	if v := s.HarvestValue("cueillette")["baies"]; rate < 40 || v != want {
		t.Errorf("tap = %g for %g units/s, want %g", v, rate, want)
	}
}

func TestByProductsWaitUntilUnlocked(t *testing.T) {
	e := era1(t)
	s := New(e, 1)
	s.Grant("baies", 10)
	if err := s.Build("cueillette", "juteuses"); err != nil {
		t.Fatal(err)
	}
	s.Step(10)
	if s.ResourceUnlocked("bois") || s.Stock("bois") != 0 {
		t.Fatalf("the bush should not make wood nobody needs yet: %g", s.Stock("bois"))
	}
	if s.Stock("baies") <= 0 {
		t.Fatal("berries are its main output")
	}
	// The flint workshop's levels cost wood: building it unlocks wood.
	s.Grant("silex", 200)
	if err := s.Build("atelier-taille", "pointes"); err != nil {
		t.Fatal(err)
	}
	s.Step(10)
	if !s.ResourceUnlocked("bois") || s.Stock("bois") <= 0 {
		t.Errorf("wood should flow once needed: unlocked %v, stock %g", s.ResourceUnlocked("bois"), s.Stock("bois"))
	}
}
