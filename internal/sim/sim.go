// Package sim is the economy of an era: stock, stations, recipes, bottlenecks,
// market, tree and quests. It is pure Go, deterministic and free of any
// rendering, so that a whole era can be played and balanced without a screen.
//
// Production is a continuous flow. Between two events (a stock running out, a
// retooling ending) every rate is constant, so Advance computes the exact
// result in one step, whatever the duration. The live 10 Hz tick and the
// offline gains therefore share the same code and always agree.
package sim

import (
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/morialkar/industrimental/internal/content"
	"github.com/morialkar/industrimental/internal/save"
)

const (
	// TickSeconds is the length of a live tick (10 Hz).
	TickSeconds = 0.1
	// DefaultOfflineCap is how much absence counts by default.
	DefaultOfflineCap = 8 * 3600.0

	levelCostGrowth = 1.15
	levelsPerDouble = 25
	stockEpsilon    = 1e-9
	retoolCostShare = 0.5
	maxActiveMain   = 3
	questCheckEvery = 1.0
)

// Errors returned by the player's actions.
var (
	ErrLocked      = errors.New("sim: not available yet")
	ErrAfford      = errors.New("sim: not enough resources")
	ErrUnknown     = errors.New("sim: unknown id")
	ErrBuilt       = errors.New("sim: already done")
	ErrMarketClose = errors.New("sim: the market is not open")
	ErrIncomplete  = errors.New("sim: the era is not complete")
)

type flow struct {
	res    int
	amount float64 // per cycle, bonuses included
}

// station is the runtime form of a station.
type station struct {
	def     *content.Station
	state   *save.StationState // nil while not built
	variant *content.Variant
	inputs  []flow
	outputs []flow
	cycles  float64 // cycles per second at full speed
	theta   float64 // share of full speed reached after bottlenecks
}

func (st *station) running(clock float64) bool {
	return st.state != nil && st.variant != nil && clock >= st.state.RetoolUntil
}

// Sim is one era being played.
type Sim struct {
	Era *content.Era

	state    *save.EraState
	resIdx   map[string]int
	stock    []float64
	produced []float64
	net      []float64
	stations []*station
	byID     map[string]*station
	nodes    map[string]*content.Node
	unlocked []bool // by resource index, see refreshUnlocks
	quests   map[string]*content.Quest
	npcs     map[string]*content.NPC
	bonus    bonuses

	questTimer float64
}

// New starts an era from scratch.
func New(era *content.Era, seed uint64) *Sim {
	return Restore(era, &save.EraState{Seed: seed})
}

// Restore resumes an era from a saved state.
func Restore(era *content.Era, st *save.EraState) *Sim {
	s := &Sim{
		Era:    era,
		state:  st,
		resIdx: map[string]int{},
		byID:   map[string]*station{},
		nodes:  map[string]*content.Node{},
		quests: map[string]*content.Quest{},
		npcs:   map[string]*content.NPC{},
	}
	ensureMaps(st)
	s.stock = make([]float64, len(era.Resources))
	s.unlocked = make([]bool, len(era.Resources))
	s.produced = make([]float64, len(era.Resources))
	s.net = make([]float64, len(era.Resources))
	for i, r := range era.Resources {
		s.resIdx[r.ID] = i
		s.stock[i] = st.Resources[r.ID]
		s.produced[i] = st.Produced[r.ID]
	}
	for _, id := range st.Unlocked {
		if i, ok := s.resIdx[id]; ok {
			s.unlocked[i] = true
		}
	}
	for i := range era.Stations {
		def := &era.Stations[i]
		rt := &station{def: def, state: st.Stations[def.ID]}
		s.stations = append(s.stations, rt)
		s.byID[def.ID] = rt
	}
	for i := range era.Tree {
		s.nodes[era.Tree[i].ID] = &era.Tree[i]
	}
	for _, id := range st.Nodes {
		if _, ok := st.NodeRanks[id]; !ok {
			if n, known := s.nodes[id]; known {
				st.NodeRanks[id] = n.Ranks()
			}
		}
	}
	for i := range era.Quests {
		s.quests[era.Quests[i].ID] = &era.Quests[i]
	}
	for i := range era.NPCs {
		s.npcs[era.NPCs[i].ID] = &era.NPCs[i]
	}
	s.recompute()
	return s
}

func ensureMaps(st *save.EraState) {
	if st.Resources == nil {
		st.Resources = map[string]float64{}
	}
	if st.Stations == nil {
		st.Stations = map[string]*save.StationState{}
	}
	if st.Produced == nil {
		st.Produced = map[string]float64{}
	}
	if st.Sold == nil {
		st.Sold = map[string]float64{}
	}
	if st.TotalSold == nil {
		st.TotalSold = map[string]float64{}
	}
	if st.Affinity == nil {
		st.Affinity = map[string]float64{}
	}
	if st.QuestSteps == nil {
		st.QuestSteps = map[string]int{}
	}
	if st.QuestDeadlines == nil {
		st.QuestDeadlines = map[string]float64{}
	}
	if st.QuestRetry == nil {
		st.QuestRetry = map[string]float64{}
	}
	if st.NodeRanks == nil {
		st.NodeRanks = map[string]int{}
	}
}

// State returns the state to persist. It stays owned by the Sim.
func (s *Sim) State() *save.EraState {
	for i, r := range s.Era.Resources {
		s.state.Resources[r.ID] = s.stock[i]
		s.state.Produced[r.ID] = s.produced[i]
	}
	return s.state
}

// Clock returns the seconds lived in this era, offline included.
func (s *Sim) Clock() float64 { return s.state.Clock }

// Stock returns the quantity held of a resource.
func (s *Sim) Stock(id string) float64 {
	if i, ok := s.resIdx[id]; ok {
		return s.stock[i]
	}
	return 0
}

// Produced returns the total produced of a resource.
func (s *Sim) Produced(id string) float64 {
	if i, ok := s.resIdx[id]; ok {
		return s.produced[i]
	}
	return 0
}

// Rate returns the current net flow of a resource, per second.
func (s *Sim) Rate(id string) float64 {
	if i, ok := s.resIdx[id]; ok {
		return s.net[i]
	}
	return 0
}

// Tick advances the live game by one 10 Hz step.
func (s *Sim) Tick() { s.Step(TickSeconds) }

// Step advances the live game by dt seconds: production, market and quests.
func (s *Sim) Step(dt float64) {
	s.advance(dt)
	s.state.Clock += dt
	s.state.MarketClock += dt
	s.decayMarket(dt)
	s.refreshEffectsAfterRetool()
	s.updateContracts()
	s.questTimer += dt
	if s.questTimer >= questCheckEvery {
		s.questTimer = 0
		s.updateQuests()
	}
}

// OfflineReport tells what happened during an absence.
type OfflineReport struct {
	Away    float64            // real duration of the absence, in seconds
	Counted float64            // part of it that produced (the cap applies)
	Gains   map[string]float64 // net change per resource
}

// Offline applies an absence: production in steady state up to the cap, market
// frozen, no event. Deadlines keep running with the real time.
func (s *Sim) Offline(away, limit float64) OfflineReport {
	before := slices.Clone(s.stock)
	counted := math.Min(away, limit)
	s.advance(counted)
	s.state.Clock += away
	s.refreshEffectsAfterRetool()
	s.updateContracts()
	s.updateQuests()
	gains := map[string]float64{}
	for i, r := range s.Era.Resources {
		if d := s.stock[i] - before[i]; d != 0 {
			gains[r.ID] = d
		}
	}
	return OfflineReport{Away: away, Counted: counted, Gains: gains}
}

// advance moves stocks forward by dt seconds, stopping at each event where the
// flows change: a stock running out or a retooling ending. Deadlines of
// retooling are compared with a clock local to this call.
func (s *Sim) advance(dt float64) {
	clock := s.state.Clock
	for i := 0; dt > 1e-12 && i < 256; i++ {
		s.solve(clock)
		step := dt
		for r, rate := range s.net {
			if rate < 0 && s.stock[r] > stockEpsilon {
				step = math.Min(step, s.stock[r]/-rate)
			}
		}
		for _, st := range s.stations {
			if st.state != nil && st.state.RetoolUntil > clock {
				step = math.Min(step, st.state.RetoolUntil-clock)
			}
		}
		for r, rate := range s.net {
			s.stock[r] = math.Max(0, s.stock[r]+rate*step)
			if s.stock[r] < stockEpsilon {
				s.stock[r] = 0
			}
		}
		for _, st := range s.stations {
			if st.theta > 0 {
				for _, o := range st.outputs {
					s.produced[o.res] += o.amount * st.cycles * st.theta * step
				}
			}
		}
		clock += step
		dt -= step
	}
	s.solve(clock)
}

// solve computes each station's real speed: a station whose input has run out
// only runs as fast as that input arrives, and that shortage carries down the
// recipe chain. It fills theta and the net flow of every resource.
func (s *Sim) solve(clock float64) {
	for _, st := range s.stations {
		st.theta = 0
		if st.running(clock) {
			st.theta = 1
		}
	}
	inflow := make([]float64, len(s.stock))
	demand := make([]float64, len(s.stock))
	for pass := 0; pass < 12; pass++ {
		clear(inflow)
		clear(demand)
		for _, st := range s.stations {
			for _, o := range st.outputs {
				inflow[o.res] += o.amount * st.cycles * st.theta
			}
			for _, in := range st.inputs {
				demand[in.res] += in.amount * st.cycles * st.theta
			}
		}
		changed := false
		for r := range s.stock {
			if s.stock[r] > stockEpsilon || demand[r] <= inflow[r]*(1+1e-12) {
				continue
			}
			f := inflow[r] / demand[r]
			for _, st := range s.stations {
				for _, in := range st.inputs {
					if in.res == r {
						st.theta *= f
						break
					}
				}
			}
			changed = true
		}
		if !changed {
			break
		}
	}
	clear(s.net)
	for _, st := range s.stations {
		for _, o := range st.outputs {
			s.net[o.res] += o.amount * st.cycles * st.theta
		}
		for _, in := range st.inputs {
			s.net[in.res] -= in.amount * st.cycles * st.theta
		}
	}
}

// --- effects -------------------------------------------------------------

type bonuses struct {
	stationRate    map[string]float64
	resourceRate   map[string]float64
	inputSaving    map[string]float64
	sellPrice      map[string]float64
	demand         map[string]float64
	contractReward float64
	questReward    float64
	retoolDiscount float64
	harvestShare   float64
	retooling      int // stations under works when the bonuses were computed
}

// apply adds an effect, scaled by f (the share of a node's ranks owned).
func (b *bonuses) apply(fx content.Effect, f float64) {
	v := fx.Value * f
	switch fx.Kind {
	case content.EffectStationRate:
		b.stationRate[fx.Station] += v
	case content.EffectResourceRate:
		b.resourceRate[fx.Resource] += v
	case content.EffectInputSaving:
		b.inputSaving[fx.Resource] += v
	case content.EffectSellPrice:
		b.sellPrice[fx.Resource] += v
	case content.EffectDemand:
		b.demand[fx.Resource] += v
	case content.EffectContractReward:
		b.contractReward += v
	case content.EffectQuestReward:
		b.questReward += v
	case content.EffectRetoolDiscount:
		b.retoolDiscount += v
	case content.EffectHarvestShare:
		b.harvestShare += v
	}
}

// recompute rebuilds bonuses and station rates after any change of levels,
// variants or nodes.
func (s *Sim) recompute() {
	s.refreshUnlocks()
	b := bonuses{
		stationRate:  map[string]float64{},
		resourceRate: map[string]float64{},
		inputSaving:  map[string]float64{},
		sellPrice:    map[string]float64{},
		demand:       map[string]float64{},
	}
	for _, id := range s.state.Nodes {
		if n, ok := s.nodes[id]; ok {
			f := float64(s.Rank(id)) / float64(n.Ranks())
			for _, fx := range n.Effects {
				b.apply(fx, f)
			}
		}
	}
	clock := s.state.Clock
	for _, st := range s.stations {
		st.variant = nil
		if st.state == nil {
			continue
		}
		for i := range st.def.Variants {
			if st.def.Variants[i].ID == st.state.Variant {
				st.variant = &st.def.Variants[i]
			}
		}
		if st.state.RetoolUntil > clock {
			b.retooling++
		} else if st.variant != nil {
			for _, fx := range st.variant.Effects {
				b.apply(fx, 1)
			}
		}
	}
	for _, st := range s.stations {
		st.inputs, st.outputs, st.cycles = nil, nil, 0
		if st.variant == nil {
			continue
		}
		r := st.variant.Recipe
		speed := 1 + b.stationRate[st.def.ID] + b.stationRate["*"]
		st.cycles = LevelMultiplier(st.state.Level) * speed / r.Seconds
		for _, id := range sortedKeys(r.Inputs) {
			saving := math.Min(0.75, b.inputSaving[id])
			st.inputs = append(st.inputs, flow{s.resIdx[id], r.Inputs[id] * (1 - saving)})
		}
		for _, id := range sortedKeys(r.Outputs) {
			if !s.unlocked[s.resIdx[id]] {
				continue // a by-product nobody needs yet
			}
			st.outputs = append(st.outputs, flow{s.resIdx[id], r.Outputs[id] * (1 + b.resourceRate[id])})
		}
	}
	s.bonus = b
	s.solve(clock)
}

// ResourceUnlocked tells whether the player has come across a resource.
func (s *Sim) ResourceUnlocked(id string) bool {
	i, ok := s.resIdx[id]
	return ok && s.unlocked[i]
}

// refreshUnlocks unlocks the resources the player now needs or makes on
// purpose: the currency, the costs of what can be built or bought, the
// inputs and main outputs of the running stations, what active quests, NPC
// deals and contracts ask for. It reports whether anything new unlocked.
func (s *Sim) refreshUnlocks() bool {
	changed := false
	mark := func(id string) {
		if i, ok := s.resIdx[id]; ok && !s.unlocked[i] {
			s.unlocked[i] = true
			s.state.Unlocked = append(s.state.Unlocked, id)
			changed = true
		}
	}
	markAll := func(a content.Amounts) {
		for id := range a {
			mark(id)
		}
	}
	mark(s.Era.Currency)
	for _, st := range s.stations {
		if st.state == nil {
			if s.StationAvailable(st.def.ID) {
				markAll(st.def.UnlockCost)
			}
			continue
		}
		markAll(st.def.LevelCost)
		for i := range st.def.Variants {
			if v := &st.def.Variants[i]; v.ID == st.state.Variant {
				markAll(v.Recipe.Inputs)
				top := 0.0
				for _, q := range v.Recipe.Outputs {
					top = math.Max(top, q)
				}
				for id, q := range v.Recipe.Outputs {
					if q >= top {
						mark(id)
					}
				}
			}
		}
	}
	for i := range s.Era.Tree {
		if n := &s.Era.Tree[i]; s.NodeReqsMet(n.ID) && !s.NodeMaxed(n.ID) {
			markAll(n.Cost)
		}
	}
	var goal func(g content.Goal)
	goal = func(g content.Goal) {
		markAll(g.Have)
		markAll(g.Deliver)
		markAll(g.Produced)
		for _, alt := range g.Any {
			goal(alt)
		}
	}
	for _, q := range s.ActiveQuests() {
		goal(s.CurrentStep(q).Goal)
	}
	for i := range s.Era.NPCs {
		if n := &s.Era.NPCs[i]; s.NPCPresent(n.ID) {
			for j := range n.Deals {
				markAll(n.Deals[j].Give)
				markAll(n.Deals[j].Get)
			}
		}
	}
	for _, c := range s.state.Contracts {
		for _, t := range s.Era.Contracts {
			if t.ID == c.Template {
				mark(t.Resource)
			}
		}
	}
	return changed
}

// refreshEffectsAfterRetool switches a variant's effects on when its retooling
// has just ended.
func (s *Sim) refreshEffectsAfterRetool() {
	if s.bonus.retooling == 0 {
		return
	}
	left := 0
	for _, st := range s.stations {
		if st.state != nil && st.state.RetoolUntil > s.state.Clock {
			left++
		}
	}
	if left != s.bonus.retooling {
		s.recompute()
	}
}

// LevelMultiplier is the production of a level relative to level 1: linear,
// doubled every 25 levels to give short-term goals.
func LevelMultiplier(level int) float64 {
	return float64(level) * math.Pow(2, float64(level/levelsPerDouble))
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// --- stock helpers --------------------------------------------------------

// CanAfford tells whether the stock covers an amount.
func (s *Sim) CanAfford(cost content.Amounts) bool {
	for id, q := range cost {
		if s.Stock(id) < q {
			return false
		}
	}
	return true
}

func (s *Sim) pay(cost content.Amounts) error {
	if !s.CanAfford(cost) {
		return ErrAfford
	}
	for id, q := range cost {
		s.stock[s.resIdx[id]] -= q
	}
	return nil
}

func (s *Sim) gain(amounts content.Amounts, factor float64) {
	for id, q := range amounts {
		if i, ok := s.resIdx[id]; ok {
			s.stock[i] += q * factor
			s.produced[i] += q * factor
		}
	}
}

// Grant adds a resource out of thin air: for tools and captures.
func (s *Sim) Grant(id string, qty float64) {
	if i, ok := s.resIdx[id]; ok {
		s.stock[i] += qty
	}
}

// --- stations -------------------------------------------------------------

// StationStatus describes a station for the interface.
type StationStatus struct {
	Built     bool
	Variant   string
	Level     int
	Retooling float64 // seconds left, 0 when running
	Speed     float64 // real share of full speed, 0 to 1
	Cycles    float64 // cycles per second at full speed
	Limiting  string  // resource holding the station back, if any
}

// Station returns the status of a station.
func (s *Sim) Station(id string) StationStatus {
	st, ok := s.byID[id]
	if !ok || st.state == nil {
		return StationStatus{}
	}
	out := StationStatus{Built: true, Variant: st.state.Variant, Level: st.state.Level, Speed: st.theta, Cycles: st.cycles}
	if left := st.state.RetoolUntil - s.state.Clock; left > 0 {
		out.Retooling = left
	}
	if st.theta < 1 && out.Retooling == 0 {
		for _, in := range st.inputs {
			if s.stock[in.res] <= stockEpsilon {
				out.Limiting = s.Era.Resources[in.res].ID
				break
			}
		}
	}
	return out
}

// StationAvailable tells whether a station's prerequisite is met.
func (s *Sim) StationAvailable(id string) bool {
	st, ok := s.byID[id]
	return ok && (st.def.RequiresNode == "" || s.HasNode(st.def.RequiresNode))
}

// Build unlocks a station with the chosen variant.
func (s *Sim) Build(id, variant string) error {
	st, ok := s.byID[id]
	if !ok || !hasVariant(st.def, variant) {
		return fmt.Errorf("%w: %s/%s", ErrUnknown, id, variant)
	}
	if st.state != nil {
		return ErrBuilt
	}
	if !s.StationAvailable(id) {
		return ErrLocked
	}
	if err := s.pay(st.def.UnlockCost); err != nil {
		return err
	}
	st.state = &save.StationState{Variant: variant, Level: 1}
	s.state.Stations[id] = st.state
	s.recompute()
	s.updateQuests()
	return nil
}

func hasVariant(def *content.Station, id string) bool {
	return slices.ContainsFunc(def.Variants, func(v content.Variant) bool { return v.ID == id })
}

// UpgradeCost is the cost of buying n levels from the current one.
func (s *Sim) UpgradeCost(id string, n int) content.Amounts {
	st, ok := s.byID[id]
	if !ok || st.state == nil {
		return nil
	}
	first := math.Pow(levelCostGrowth, float64(st.state.Level-1))
	factor := first * (math.Pow(levelCostGrowth, float64(n)) - 1) / (levelCostGrowth - 1)
	cost := content.Amounts{}
	for res, q := range st.def.LevelCost {
		cost[res] = q * factor
	}
	return cost
}

// MaxUpgrades is how many levels the stock can pay right now.
func (s *Sim) MaxUpgrades(id string) int {
	n := 0
	for n < 1000 && s.CanAfford(s.UpgradeCost(id, n+1)) {
		n++
	}
	return n
}

// Upgrade buys n levels of a built station.
func (s *Sim) Upgrade(id string, n int) error {
	st, ok := s.byID[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknown, id)
	}
	if st.state == nil {
		return ErrLocked
	}
	if err := s.pay(s.UpgradeCost(id, n)); err != nil {
		return err
	}
	st.state.Level += n
	s.recompute()
	s.updateQuests()
	return nil
}

// RetoolCost is the price of changing a station's variant: half its unlock
// cost, less the tree's discount.
func (s *Sim) RetoolCost(id string) content.Amounts {
	st, ok := s.byID[id]
	if !ok {
		return nil
	}
	f := retoolCostShare * (1 - math.Min(0.75, s.bonus.retoolDiscount))
	cost := content.Amounts{}
	for res, q := range st.def.UnlockCost {
		cost[res] = q * f
	}
	return cost
}

// Retool changes a station's variant and stops it during the works.
func (s *Sim) Retool(id, variant string) error {
	st, ok := s.byID[id]
	if !ok || !hasVariant(st.def, variant) {
		return fmt.Errorf("%w: %s/%s", ErrUnknown, id, variant)
	}
	if st.state == nil {
		return ErrLocked
	}
	if st.state.Variant == variant {
		return ErrBuilt
	}
	if err := s.pay(s.RetoolCost(id)); err != nil {
		return err
	}
	st.state.Variant = variant
	st.state.RetoolUntil = s.state.Clock + st.def.RetoolSeconds*(1-math.Min(0.75, s.bonus.retoolDiscount))
	s.recompute()
	return nil
}

// HarvestValue is what one tap on a station gives: the era's share of the
// production per second (raised by the tree), never less than the station's
// own harvest.
func (s *Sim) HarvestValue(id string) content.Amounts {
	st, ok := s.byID[id]
	if !ok || len(st.def.Harvest) == 0 {
		return nil
	}
	rate := 0.0
	for i, r := range s.Era.Resources {
		if r.Kind != content.KindCurrency && s.net[i] > 0 {
			rate += s.net[i]
		}
	}
	share := s.Era.HarvestShare + s.bonus.harvestShare
	out := content.Amounts{}
	for res, q := range st.def.Harvest {
		out[res] = math.Max(q, math.Round(share*rate*10)/10)
	}
	return out
}

// HarvestShare is the share of the production per second that a tap gives.
func (s *Sim) HarvestShare() float64 { return s.Era.HarvestShare + s.bonus.harvestShare }

// HarvestStation is the station the harvest button gathers from, if any.
func (s *Sim) HarvestStation() string {
	for _, st := range s.stations {
		if len(st.def.Harvest) > 0 {
			return st.def.ID
		}
	}
	return ""
}

// Harvest is the tap on a station that can be gathered by hand. It returns
// what the tap gave.
func (s *Sim) Harvest(id string) (content.Amounts, error) {
	st, ok := s.byID[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknown, id)
	}
	if len(st.def.Harvest) == 0 {
		return nil, ErrLocked
	}
	got := s.HarvestValue(id)
	s.gain(got, 1)
	s.updateQuests()
	return got, nil
}

// --- tree -----------------------------------------------------------------

// Rank is how many ranks of a node are bought.
func (s *Sim) Rank(id string) int { return s.state.NodeRanks[id] }

// HasNode tells whether a tree node has at least one rank.
func (s *Sim) HasNode(id string) bool { return s.Rank(id) > 0 }

// NodeMaxed tells whether every rank of a node is bought.
func (s *Sim) NodeMaxed(id string) bool {
	n, ok := s.nodes[id]
	return ok && s.Rank(id) >= n.Ranks()
}

// RowOpen tells whether a row of the branch boards is open: enough stations
// are built. The trunk is always open.
func (s *Sim) RowOpen(row int) bool {
	if row < 0 || row >= len(s.Era.RowStations) {
		return true
	}
	return s.StationsBuilt() >= s.Era.RowStations[row]
}

// StationsBuilt counts the built stations.
func (s *Sim) StationsBuilt() int {
	n := 0
	for _, st := range s.stations {
		if st.state != nil {
			n++
		}
	}
	return n
}

// RowsOpen counts the open rows of the branch boards.
func (s *Sim) RowsOpen() int {
	n := 0
	for r := 1; r <= content.BoardRows; r++ {
		if s.RowOpen(r) {
			n++
		}
	}
	return n
}

// NodeUnlocked tells whether a node's prerequisites are met and its row is open.
func (s *Sim) NodeUnlocked(id string) bool {
	n, ok := s.nodes[id]
	if !ok {
		return false
	}
	if n.Branch != content.BranchTrunk && !s.RowOpen(n.Row) {
		return false
	}
	return s.NodeReqsMet(id)
}

// NodeReqsMet tells whether a node's prerequisites are met, whatever its row.
func (s *Sim) NodeReqsMet(id string) bool {
	n, ok := s.nodes[id]
	if !ok {
		return false
	}
	for _, r := range n.RequiresAll {
		if !s.HasNode(r) {
			return false
		}
	}
	if len(n.RequiresAny) > 0 && !slices.ContainsFunc(n.RequiresAny, s.HasNode) {
		return false
	}
	return true
}

// NodeAvailable tells whether the next rank of a node can be bought, given
// enough resources.
func (s *Sim) NodeAvailable(id string) bool { return s.NodeUnlocked(id) && !s.NodeMaxed(id) }

// NextRankCost is the price of a node's next rank, nil when it is maxed.
func (s *Sim) NextRankCost(id string) content.Amounts {
	n, ok := s.nodes[id]
	if !ok || s.NodeMaxed(id) {
		return nil
	}
	return n.RankCost(s.Rank(id) + 1)
}

// CanInvest tells whether the next rank of a node can be bought right now.
func (s *Sim) CanInvest(id string) bool {
	return s.NodeAvailable(id) && s.CanAfford(s.NextRankCost(id))
}

// BuyNode buys the next rank of a tree node.
func (s *Sim) BuyNode(id string) error {
	if _, ok := s.nodes[id]; !ok {
		return fmt.Errorf("%w: %s", ErrUnknown, id)
	}
	if s.NodeMaxed(id) {
		return ErrBuilt
	}
	if !s.NodeUnlocked(id) {
		return ErrLocked
	}
	if err := s.pay(s.NextRankCost(id)); err != nil {
		return err
	}
	if !s.HasNode(id) {
		s.state.Nodes = append(s.state.Nodes, id)
	}
	s.state.NodeRanks[id]++
	s.recompute()
	s.updateQuests()
	return nil
}

// BuyNodeFully buys every remaining rank of a node, stopping at the first
// rank the stock cannot pay.
func (s *Sim) BuyNodeFully(id string) error {
	var err error
	for !s.NodeMaxed(id) {
		if err = s.BuyNode(id); err != nil {
			break
		}
	}
	if s.NodeMaxed(id) {
		return nil
	}
	return err
}

// BranchProgress is the share of a branch's ranks bought, bridges counting
// for both of their branches. It is what will unlock the hybrid endings.
func (s *Sim) BranchProgress(axis content.Axis) float64 {
	total, got := 0, 0
	for i := range s.Era.Tree {
		n := &s.Era.Tree[i]
		if n.Branch == string(axis) || n.BridgeTo == string(axis) {
			total += n.Ranks()
			got += min(s.Rank(n.ID), n.Ranks())
		}
	}
	if total == 0 {
		return 0
	}
	return float64(got) / float64(total)
}

// --- completion -----------------------------------------------------------

// Completion is the state of the three conditions that open the portal.
type Completion struct {
	Keys          []content.Axis // Era Keys bought
	StationsBuilt int
	StationsTotal int
	QuestsDone    int
	QuestsTotal   int
}

// Complete tells whether the three conditions hold.
func (c Completion) Complete() bool {
	return len(c.Keys) > 0 && c.StationsBuilt == c.StationsTotal && c.QuestsDone == c.QuestsTotal
}

// Completion returns the current progress towards the portal.
func (s *Sim) Completion() Completion {
	c := Completion{StationsTotal: len(s.stations)}
	for _, st := range s.stations {
		if st.state != nil {
			c.StationsBuilt++
		}
	}
	for i := range s.Era.Tree {
		if n := &s.Era.Tree[i]; n.EraKey && s.HasNode(n.ID) {
			c.Keys = append(c.Keys, content.Axis(n.Branch))
		}
	}
	for i := range s.Era.Quests {
		if q := &s.Era.Quests[i]; q.Main {
			c.QuestsTotal++
			if s.QuestDone(q.ID) {
				c.QuestsDone++
			}
		}
	}
	return c
}

// Finish takes the portal with the ending of a bought Era Key.
func (s *Sim) Finish(axis content.Axis) error {
	c := s.Completion()
	if !c.Complete() || !slices.Contains(c.Keys, axis) {
		return ErrIncomplete
	}
	s.state.Ending = string(axis)
	return nil
}

// Ending returns the axis of the ending taken, if any.
func (s *Sim) Ending() string { return s.state.Ending }
