// Package balance plays an era without a screen, like a regular player would:
// a few short sessions a day, one strategy per ending, offline gains in
// between. It measures how long each ending takes and records the curves.
package balance

import (
	"fmt"
	"math"
	"slices"
	"sort"

	"github.com/morialkar/industrimental/internal/content"
	"github.com/morialkar/industrimental/internal/sim"
)

// Schedule describes the habits of the simulated player.
type Schedule struct {
	SessionHours   []float64 // start of each session within a day
	SessionSeconds float64   // length of a session
	FirstSession   float64   // length of the very first session
	DecisionEvery  float64   // seconds between two decisions
	OfflineCap     float64
	MaxDays        float64
}

// Regular is the target player of the design: 5 short sessions a day.
func Regular() Schedule {
	return Schedule{
		SessionHours:   []float64{0, 3, 6, 9, 12},
		SessionSeconds: 300,
		FirstSession:   1200,
		DecisionEvery:  5,
		OfflineCap:     sim.DefaultOfflineCap,
		MaxDays:        14,
	}
}

// Strategy aims for one ending.
type Strategy struct {
	Axis     content.Axis
	Variants map[string]string // station -> variant chosen when it is built
	Path     string            // tree path followed in the branch
}

// Strategies returns the reference strategy of each ending of era 1.
func Strategies() []Strategy {
	return []Strategy{
		{content.AxisTrade, map[string]string{
			"cueillette": "juteuses", "carriere-silex": "galets", "camp-chasse": "pieges",
			"atelier-taille": "parures", "cercle-troc": "foire", "feu-camp": "four-ocre",
			"source-salee": "mine-sel", "tannerie": "cuir-epais", "fosse-argile": "jarres", "champ": "ble",
		}, "foires"},
		{content.AxisThroughput, map[string]string{
			"cueillette": "fibres", "carriere-silex": "meules", "camp-chasse": "pieges",
			"atelier-taille": "outils-agricoles", "cercle-troc": "pierre-echanges", "feu-camp": "four-ocre",
			"source-salee": "mine-sel", "tannerie": "outres", "fosse-argile": "jarres", "champ": "ble",
		}, "ateliers"},
		{content.AxisIngenuity, map[string]string{
			"cueillette": "fibres", "carriere-silex": "silex-fin", "camp-chasse": "pieges",
			"atelier-taille": "pointes", "cercle-troc": "pierre-echanges", "feu-camp": "four-ocre",
			"source-salee": "sel-marin", "tannerie": "outres", "fosse-argile": "jarres", "champ": "lin",
		}, "pigments"},
		{content.AxisCommunity, map[string]string{
			"cueillette": "fibres", "carriere-silex": "galets", "camp-chasse": "pieges",
			"atelier-taille": "parures", "cercle-troc": "pierre-echanges", "feu-camp": "veillee",
			"source-salee": "sel-marin", "tannerie": "outres", "fosse-argile": "figurines", "champ": "lin",
		}, "veillees"},
	}
}

// Sample is one point of the progression curves.
type Sample struct {
	Hours    float64
	Stations int
	Nodes    int
	Quests   int
	Stock    map[string]float64
}

// Result is the outcome of one simulated game.
type Result struct {
	Axis       content.Axis
	Reached    bool
	Hours      float64 // from the first session to the portal
	KeyHours   float64 // when the Era Key was bought
	Sessions   int
	Milestones []Milestone
	Samples    []Sample
	Left       string // what was still missing, when the portal wasn't reached
	Summary    string // final levels and flows, to read a game at a glance
}

// Milestone is a dated step of a game.
type Milestone struct {
	Hours float64
	What  string
}

type player struct {
	s        *sim.Sim
	strat    Strategy
	path     []string
	order    []string // stations in build order
	needs    map[string]float64
	lastSeen map[string]bool
	res      *Result
	hours    func() float64
}

// Play runs one game of an era with a strategy.
func Play(era *content.Era, strat Strategy, sched Schedule, seed uint64) Result {
	s := sim.New(era, seed)
	res := Result{Axis: strat.Axis}
	clock := 0.0
	p := &player{s: s, strat: strat, res: &res, lastSeen: map[string]bool{}, hours: func() float64 { return clock / 3600 }}
	p.path = treePath(era, strat)
	for i := range era.Stations {
		p.order = append(p.order, era.Stations[i].ID)
	}

	day := 86400.0
	for d := 0; float64(d) < sched.MaxDays; d++ {
		for i, h := range sched.SessionHours {
			start := float64(d)*day + h*3600
			if away := start - clock; away > 0 {
				s.Offline(away, sched.OfflineCap)
				clock = start
			}
			length := sched.SessionSeconds
			if d == 0 && i == 0 {
				length = sched.FirstSession
			}
			res.Sessions++
			for t := 0.0; t < length; t += sched.DecisionEvery {
				p.decide(sched.DecisionEvery)
				s.Step(sched.DecisionEvery)
				clock += sched.DecisionEvery
				if p.finish() {
					res.Reached = true
					res.Hours = clock / 3600
					res.Summary = p.summary()
					p.sample()
					return res
				}
			}
			p.sample()
			p.note()
		}
	}
	res.Left = p.missingText()
	res.Summary = p.summary()
	return res
}

// treePath lists the nodes to buy, in order: trunk, the chosen path, what
// joins it to the Era Key.
func treePath(era *content.Era, st Strategy) []string {
	var out []string
	for i := range era.Tree {
		if n := &era.Tree[i]; n.Branch == content.BranchTrunk {
			out = append(out, n.ID)
		}
	}
	for i := range era.Tree {
		n := &era.Tree[i]
		if n.Branch == string(st.Axis) && n.Path == st.Path {
			out = append(out, n.ID)
		}
	}
	for i := range era.Tree {
		n := &era.Tree[i]
		if n.Branch == string(st.Axis) && n.Path == "" && n.BridgeTo == "" {
			out = append(out, n.ID)
		}
	}
	return out
}

func (p *player) finish() bool {
	c := p.s.Completion()
	if !slices.Contains(c.Keys, p.strat.Axis) {
		return false
	}
	if p.res.KeyHours == 0 {
		p.res.KeyHours = p.hours()
	}
	return p.s.Finish(p.strat.Axis) == nil
}

func (p *player) note() {
	s := p.s
	for _, id := range p.order {
		if key := "station " + id; s.Station(id).Built && !p.lastSeen[key] {
			p.lastSeen[key] = true
			p.res.Milestones = append(p.res.Milestones, Milestone{p.hours(), key})
		}
	}
	for _, id := range p.path {
		if key := "node " + id; s.HasNode(id) && !p.lastSeen[key] {
			p.lastSeen[key] = true
			p.res.Milestones = append(p.res.Milestones, Milestone{p.hours(), key})
		}
	}
}

func (p *player) sample() {
	c := p.s.Completion()
	nodes := 0
	for i := range p.s.Era.Tree {
		if p.s.HasNode(p.s.Era.Tree[i].ID) {
			nodes++
		}
	}
	stock := map[string]float64{}
	for _, r := range p.s.Era.Resources {
		stock[r.ID] = p.s.Stock(r.ID)
	}
	p.res.Samples = append(p.res.Samples, Sample{p.hours(), c.StationsBuilt, nodes, c.QuestsDone, stock})
}

func (p *player) summary() string {
	s := p.s
	out := "niveaux:"
	for _, id := range p.order {
		st := s.Station(id)
		out += fmt.Sprintf(" %s=%d(%.0f%%)", id, st.Level, st.Speed*100)
	}
	out += "\nstock:"
	for _, r := range s.Era.Resources {
		out += fmt.Sprintf(" %s=%.0f(%+.1f/s,%.1f$)", r.ID, s.Stock(r.ID), s.Rate(r.ID), s.Price(r.ID))
	}
	return out
}

func (p *player) missingText() string {
	c := p.s.Completion()
	var need []string
	for _, q := range p.s.ActiveQuests() {
		if q.Main {
			need = append(need, q.ID)
		}
	}
	next := ""
	for _, id := range p.path {
		if !p.s.NodeMaxed(id) {
			next = id
			break
		}
	}
	return fmt.Sprintf("keys=%v stations=%d/%d quests=%d/%d next node=%s active=%v",
		c.Keys, c.StationsBuilt, c.StationsTotal, c.QuestsDone, c.QuestsTotal, next, need)
}

// --- decisions ------------------------------------------------------------

func (p *player) decide(dt float64) {
	s := p.s
	// The opening: tap the bush until it can be built and Bob can trade.
	if !s.Station("cueillette").Built || !s.Station("carriere-silex").Built {
		for i := 0; i < int(dt*3); i++ {
			_, _ = s.Harvest("cueillette")
		}
	}
	if !s.Station("carriere-silex").Built && s.Stock("silex") < 1 {
		_ = s.Trade("bob", "silex")
	}
	for round := 0; round < 40; round++ {
		p.computeNeeds()
		if !p.act() {
			break
		}
	}
	// Whatever the investments left over goes to market, for the next ones.
	p.computeNeeds()
	p.sellSurplus()
	for round := 0; round < 10; round++ {
		if !p.act() {
			break
		}
		p.computeNeeds()
	}
}

// act makes the single best affordable move and reports whether there was one.
func (p *player) act() bool {
	s := p.s
	for _, id := range p.order {
		if !s.Station(id).Built && s.Build(id, p.strat.Variants[id]) == nil {
			return true
		}
	}
	for _, id := range p.path {
		if !s.NodeMaxed(id) {
			if s.BuyNode(id) == nil {
				return true
			}
			break
		}
	}
	for _, q := range s.ActiveQuests() {
		if s.CanDeliver(q) && (q.Main || p.spare(s.CurrentStep(q).Goal)) && s.Deliver(q.ID) == nil {
			return true
		}
	}
	for _, c := range s.Contracts() {
		if s.Stock(c.Template.Resource)-p.needs[c.Template.Resource] >= c.Amount && s.DeliverContract(c.Index) == nil {
			return true
		}
	}
	if p.levelForQuests() {
		return true
	}
	if p.buyMissing() {
		return true
	}
	return p.upgrade()
}

// levelForQuests raises the stations an active main quest asks for.
func (p *player) levelForQuests() bool {
	for _, q := range p.s.ActiveQuests() {
		if q.Main && p.levelForGoal(p.s.CurrentStep(q).Goal) {
			return true
		}
	}
	return false
}

func (p *player) levelForGoal(g content.Goal) bool {
	s := p.s
	if len(g.Any) > 0 {
		for _, alt := range g.Any {
			if p.produces(alt) {
				return p.levelForGoal(alt)
			}
		}
		return false
	}
	for _, id := range sortedLevels(g.Levels) {
		if st := s.Station(id); st.Built && st.Level < g.Levels[id] && s.Upgrade(id, 1) == nil {
			return true
		}
	}
	return false
}

func sortedLevels(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (p *player) spare(g content.Goal) bool {
	for id, q := range g.Deliver {
		if p.s.Stock(id)-p.needs[id] < q {
			return false
		}
	}
	return len(g.Any) == 0
}

// computeNeeds lists what the next purchases and the active quests still ask
// for: the player sells nothing of it.
func (p *player) computeNeeds() {
	s := p.s
	needs := map[string]float64{}
	add := func(a content.Amounts) {
		for id, q := range a {
			needs[id] += q
		}
	}
	for _, id := range p.order {
		if !s.Station(id).Built {
			def := stationDef(s.Era, id)
			add(def.UnlockCost)
			if !s.StationAvailable(id) {
				break // later stations can wait
			}
		}
	}
	bought := 0
	for _, id := range p.path {
		if s.NodeMaxed(id) {
			continue
		}
		add(remainingCost(s, id))
		if bought++; bought == 2 {
			break
		}
	}
	key := keyNode(s.Era, p.strat.Axis)
	add(key.Cost)
	for _, q := range s.ActiveQuests() {
		if !q.Main {
			continue
		}
		p.goalNeeds(s.CurrentStep(q).Goal, needs)
	}
	p.needs = needs
}

func (p *player) goalNeeds(g content.Goal, needs map[string]float64) {
	s := p.s
	if len(g.Any) > 0 {
		best := 0
		for i := range g.Any {
			if p.produces(g.Any[i]) {
				best = i
				break
			}
		}
		p.goalNeeds(g.Any[best], needs)
		return
	}
	for id, q := range g.Have {
		needs[id] += q
	}
	for id, q := range g.Deliver {
		needs[id] += q
	}
	for id, q := range g.Produced {
		if left := q - s.Produced(id); left > 0 {
			needs[id] += left
		}
	}
	if left := g.Earned - s.State().Earned; left > 0 {
		needs[s.Era.Currency] += 0 // earned by selling: see sellSurplus
	}
	for id, lvl := range g.Levels {
		if st := s.Station(id); st.Built && st.Level < lvl {
			for r, c := range s.UpgradeCost(id, lvl-st.Level) {
				needs[r] += c
			}
		}
	}
	for _, id := range g.Nodes {
		if !s.HasNode(id) {
			for r, c := range nodeDef(s.Era, id).RankCost(1) {
				needs[r] += c
			}
		}
	}
}

func (p *player) produces(g content.Goal) bool {
	for id := range g.Deliver {
		if p.rate(id) > 0 {
			return true
		}
	}
	for id := range g.Produced {
		if p.rate(id) > 0 {
			return true
		}
	}
	for id := range g.Levels {
		if p.s.Station(id).Built {
			return true
		}
	}
	return false
}

// rate is the gross output of a resource when its stations run at full speed.
func (p *player) rate(id string) float64 {
	s := p.s
	total := 0.0
	for i := range s.Era.Stations {
		def := &s.Era.Stations[i]
		st := s.Station(def.ID)
		if !st.Built {
			continue
		}
		for j := range def.Variants {
			if def.Variants[j].ID == st.Variant {
				total += def.Variants[j].Recipe.Outputs[id] * st.Cycles
			}
		}
	}
	return total
}

// consumption is what the stations eat of a resource at full speed, per second.
func (p *player) consumption(id string) float64 {
	s := p.s
	total := 0.0
	for i := range s.Era.Stations {
		def := &s.Era.Stations[i]
		st := s.Station(def.ID)
		if st.Built {
			total += variantOf(def, st.Variant).Recipe.Inputs[id] * st.Cycles
		}
	}
	return total
}

// sellSurplus sells what no target needs, as long as the price stays decent.
func (p *player) sellSurplus() bool {
	s := p.s
	if !s.MarketOpen() {
		return false
	}
	sold := false
	for _, r := range s.Era.Resources {
		if r.Kind == content.KindCurrency {
			continue
		}
		// Keep what the next purchases need and an hour of the recipes' food.
		surplus := s.Stock(r.ID) - p.needs[r.ID]*1.1 - p.consumption(r.ID)*3600
		if surplus < 1 || s.Price(r.ID) < r.BasePrice*0.2 {
			continue
		}
		// Sell a slice at a time so that the price falls gradually.
		lot := math.Max(1, surplus*0.25)
		if _, err := s.Sell(r.ID, lot); err == nil {
			sold = true
		}
	}
	return sold
}

// buyMissing buys small amounts the player's stations cannot make.
func (p *player) buyMissing() bool {
	s := p.s
	if !s.MarketOpen() {
		return false
	}
	bought := false
	for _, id := range sortedKeys(p.needs) {
		if id == s.Era.Currency || p.rate(id) > 0 {
			continue
		}
		key := keyNode(s.Era, p.strat.Axis)
		if _, inKey := key.Cost[id]; inKey {
			continue
		}
		missing := p.needs[id] - s.Stock(id)
		if missing < 1 {
			continue
		}
		if cost := s.BuyCost(id, missing); cost < s.Stock(s.Era.Currency)*0.5 && s.Buy(id, missing) == nil {
			bought = true
		}
	}
	return bought
}

// upgrade buys a level of whichever station would best relieve the slowest
// need: the producer of the resource furthest from its target.
func (p *player) upgrade() bool {
	s := p.s
	type lag struct {
		res  string
		time float64
	}
	var lags []lag
	for id, q := range p.needs {
		missing := q - s.Stock(id)
		if missing <= 0 || id == s.Era.Currency {
			continue
		}
		r := p.rate(id)
		if r <= 0 {
			continue
		}
		lags = append(lags, lag{id, missing / r})
	}
	sort.Slice(lags, func(i, j int) bool {
		if lags[i].time != lags[j].time {
			return lags[i].time > lags[j].time
		}
		return lags[i].res < lags[j].res
	})
	tried := map[string]bool{}
	var try func(res string, depth int) bool
	try = func(res string, depth int) bool {
		if depth > 4 {
			return false
		}
		for i := range s.Era.Stations {
			def := &s.Era.Stations[i]
			st := s.Station(def.ID)
			if !st.Built || tried[def.ID] {
				continue
			}
			recipe := variantOf(def, st.Variant).Recipe
			if recipe.Outputs[res] <= 0 {
				continue
			}
			tried[def.ID] = true
			// A starved station needs its supplier first.
			if st.Limiting != "" && try(st.Limiting, depth+1) {
				return true
			}
			if p.affordableUpgrade(def.ID) {
				return true
			}
			for in := range recipe.Inputs {
				if try(in, depth+1) {
					return true
				}
			}
		}
		return false
	}
	for _, l := range lags {
		if try(l.res, 0) {
			return true
		}
	}
	return false
}

func (p *player) affordableUpgrade(id string) bool {
	s := p.s
	cost := s.UpgradeCost(id, 1)
	for r, q := range cost {
		// Spend at most half of a stock on levels, and leave the current
		// purchases alone when the resource is scarce.
		if s.Stock(r)*0.5 < q {
			return false
		}
	}
	return s.Upgrade(id, 1) == nil
}

func stationDef(e *content.Era, id string) *content.Station {
	for i := range e.Stations {
		if e.Stations[i].ID == id {
			return &e.Stations[i]
		}
	}
	return nil
}

func nodeDef(e *content.Era, id string) *content.Node {
	for i := range e.Tree {
		if e.Tree[i].ID == id {
			return &e.Tree[i]
		}
	}
	return nil
}

func keyNode(e *content.Era, axis content.Axis) *content.Node {
	for i := range e.Tree {
		if e.Tree[i].EraKey && e.Tree[i].Branch == string(axis) {
			return &e.Tree[i]
		}
	}
	return nil
}

func variantOf(def *content.Station, id string) *content.Variant {
	for i := range def.Variants {
		if def.Variants[i].ID == id {
			return &def.Variants[i]
		}
	}
	return &def.Variants[0]
}

func sortedKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// remainingCost is the price of every rank of a node not bought yet.
func remainingCost(s *sim.Sim, id string) content.Amounts {
	n := nodeDef(s.Era, id)
	out := content.Amounts{}
	for r := s.Rank(id) + 1; r <= n.Ranks(); r++ {
		for res, q := range n.RankCost(r) {
			out[res] += q
		}
	}
	return out
}
