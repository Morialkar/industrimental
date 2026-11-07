package sim

import (
	"fmt"
	"hash/fnv"
	"math"
	"math/rand/v2"
	"slices"

	"github.com/morialkar/industrimental/internal/content"
	"github.com/morialkar/industrimental/internal/save"
)

const (
	soldHalfLife     = 3600.0 // S loses half of its value every hour
	demandGrowth     = 0.1    // each unit ever sold adds this much to D
	buySpread        = 1.5    // buying costs this much more than selling brings
	trendAmplitude   = 0.15
	trendMinPeriod   = 4 * 3600.0
	trendMaxPeriod   = 12 * 3600.0
	sellSlices       = 16
	affinityStep     = 1.0
	affinityFailStep = 0.5
	affinityBonus    = 0.02 // per point of affinity
	affinityBonusCap = 0.20

	maxOpenContracts = 3
	contractEvery    = 600.0
)

// MarketOpen tells whether the station that opens the market is built. An era
// without such a station always has its market open.
func (s *Sim) MarketOpen() bool {
	found := false
	for _, st := range s.stations {
		if st.def.Market {
			found = true
			if st.state != nil {
				return true
			}
		}
	}
	return !found
}

func (s *Sim) resource(id string) (*content.Resource, bool) {
	i, ok := s.resIdx[id]
	if !ok {
		return nil, false
	}
	return &s.Era.Resources[i], true
}

func (s *Sim) sellable(id string) (*content.Resource, error) {
	r, ok := s.resource(id)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknown, id)
	}
	if r.Kind == content.KindCurrency {
		return nil, ErrLocked
	}
	if !s.MarketOpen() {
		return nil, ErrMarketClose
	}
	return r, nil
}

func hashFloat(id string, salt byte) float64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	_, _ = h.Write([]byte{salt})
	return float64(h.Sum32()) / (1 << 32)
}

// Trend is the slow cycle of a resource's price, between -15 % and +15 %.
func (s *Sim) Trend(id string) float64 {
	period := trendMinPeriod + hashFloat(id, 1)*(trendMaxPeriod-trendMinPeriod)
	phase := hashFloat(id, 2)
	return trendAmplitude * math.Sin(2*math.Pi*(s.state.MarketClock/period+phase))
}

func (s *Sim) demand(r *content.Resource) float64 {
	return r.Demand*(1+s.bonus.demand[r.ID]+s.bonus.demand["*"]) + demandGrowth*s.state.TotalSold[r.ID]
}

// priceAt is the market formula, for S units sold recently:
// P = P0 x (D / (D + S))^e x (1 + trend), plus the bonuses of the tree.
func (s *Sim) priceAt(r *content.Resource, sold float64) float64 {
	d := s.demand(r)
	return r.BasePrice * math.Pow(d/(d+sold), r.Elasticity) *
		(1 + s.Trend(r.ID)) * (1 + s.bonus.sellPrice[r.ID] + s.bonus.sellPrice["*"])
}

// Price returns the current selling price of one unit.
func (s *Sim) Price(id string) float64 {
	r, ok := s.resource(id)
	if !ok || r.Kind == content.KindCurrency {
		return 0
	}
	return s.priceAt(r, s.state.Sold[id])
}

// integrate sums the price between two values of S. Selling a whole lot at
// once thus pays what selling it unit by unit would have.
func (s *Sim) integrate(r *content.Resource, from, to float64) float64 {
	step := (to - from) / sellSlices
	total := 0.0
	for i := 0; i < sellSlices; i++ {
		total += s.priceAt(r, from+(float64(i)+0.5)*step) * step
	}
	return total
}

// SellValue is what selling qty units would bring now, in currency.
func (s *Sim) SellValue(id string, qty float64) float64 {
	r, ok := s.resource(id)
	if !ok || r.Kind == content.KindCurrency {
		return 0
	}
	sold := s.state.Sold[id]
	return s.integrate(r, sold, sold+qty)
}

// Sell sells on the market and returns the currency earned.
func (s *Sim) Sell(id string, qty float64) (float64, error) {
	return s.sellWith(id, qty, 1)
}

// SellTo sells to an NPC, who pays more for what they like and more still as
// their affinity rises.
func (s *Sim) SellTo(npc, id string, qty float64) (float64, error) {
	n, ok := s.npcs[npc]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrUnknown, npc)
	}
	mult, ok := n.Buys[id]
	if !ok {
		return 0, ErrLocked
	}
	earned, err := s.sellWith(id, qty, mult*(1+s.AffinityBonus(npc)))
	if err == nil {
		s.state.Affinity[npc] += affinityStep
	}
	return earned, err
}

func (s *Sim) sellWith(id string, qty, mult float64) (float64, error) {
	r, err := s.sellable(id)
	if err != nil {
		return 0, err
	}
	if qty <= 0 || s.Stock(id) < qty {
		return 0, ErrAfford
	}
	sold := s.state.Sold[id]
	earned := s.integrate(r, sold, sold+qty) * mult
	s.stock[s.resIdx[id]] -= qty
	s.state.Sold[id] = sold + qty
	s.state.TotalSold[id] += qty
	s.state.Earned += earned
	s.stock[s.resIdx[s.Era.Currency]] += earned
	return earned, nil
}

// BuyCost is what buying qty units would cost now, in currency. It is always
// more than selling them back brings, so that no round trip can pay.
func (s *Sim) BuyCost(id string, qty float64) float64 {
	r, ok := s.resource(id)
	if !ok || r.Kind == content.KindCurrency {
		return 0
	}
	sold := s.state.Sold[id]
	onCurve := math.Min(sold, qty)
	cost := s.integrate(r, sold-onCurve, sold)
	if qty > onCurve {
		cost += (qty - onCurve) * s.priceAt(r, 0)
	}
	return cost * buySpread
}

// Buy buys on the market. Buying eases the pressure left by earlier sales.
func (s *Sim) Buy(id string, qty float64) error {
	if _, err := s.sellable(id); err != nil {
		return err
	}
	if qty <= 0 {
		return ErrAfford
	}
	cost := s.BuyCost(id, qty)
	cur := s.resIdx[s.Era.Currency]
	if s.stock[cur] < cost {
		return ErrAfford
	}
	s.stock[cur] -= cost
	s.stock[s.resIdx[id]] += qty
	s.state.Sold[id] = math.Max(0, s.state.Sold[id]-qty)
	return nil
}

func (s *Sim) decayMarket(dt float64) {
	f := math.Pow(0.5, dt/soldHalfLife)
	for id, v := range s.state.Sold {
		s.state.Sold[id] = v * f
	}
}

// --- NPCs -----------------------------------------------------------------

// NPCPresent tells whether an NPC can trade right now.
func (s *Sim) NPCPresent(id string) bool {
	n, ok := s.npcs[id]
	if !ok {
		return false
	}
	if n.RequiresStation == "" {
		return true
	}
	st, ok := s.byID[n.RequiresStation]
	return ok && st.state != nil
}

// Affinity returns an NPC's affinity.
func (s *Sim) Affinity(id string) float64 { return s.state.Affinity[id] }

// AffinityBonus is the price advantage an NPC's affinity gives.
func (s *Sim) AffinityBonus(id string) float64 {
	return math.Min(affinityBonusCap, s.state.Affinity[id]*affinityBonus)
}

// DealGive is what a deal costs now, the NPC's affinity included.
func (s *Sim) DealGive(npc string, d *content.Deal) content.Amounts {
	f := 1 - s.AffinityBonus(npc)
	give := content.Amounts{}
	for id, q := range d.Give {
		give[id] = q * f
	}
	return give
}

// Trade makes a deal with an NPC.
func (s *Sim) Trade(npc, deal string) error {
	n, ok := s.npcs[npc]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknown, npc)
	}
	if !s.NPCPresent(npc) {
		return ErrLocked
	}
	i := slices.IndexFunc(n.Deals, func(d content.Deal) bool { return d.ID == deal })
	if i < 0 {
		return fmt.Errorf("%w: %s", ErrUnknown, deal)
	}
	if err := s.pay(s.DealGive(npc, &n.Deals[i])); err != nil {
		return err
	}
	s.gain(n.Deals[i].Get, 1)
	s.state.Affinity[npc] += affinityStep
	s.updateQuests()
	return nil
}

// --- repeating contracts --------------------------------------------------

// OpenContract is a delivery order the player can fulfil.
type OpenContract struct {
	Index    int
	Template *content.Contract
	Amount   float64
	Reward   float64
	Left     float64 // seconds before it expires
}

// Contracts lists the open delivery orders.
func (s *Sim) Contracts() []OpenContract {
	out := make([]OpenContract, 0, len(s.state.Contracts))
	for i, c := range s.state.Contracts {
		if t := s.contractTemplate(c.Template); t != nil {
			out = append(out, OpenContract{i, t, c.Amount, c.Reward, c.Deadline - s.state.Clock})
		}
	}
	return out
}

func (s *Sim) contractTemplate(id string) *content.Contract {
	for i := range s.Era.Contracts {
		if s.Era.Contracts[i].ID == id {
			return &s.Era.Contracts[i]
		}
	}
	return nil
}

func (s *Sim) updateContracts() {
	if !s.MarketOpen() || len(s.Era.Contracts) == 0 {
		return
	}
	kept := s.state.Contracts[:0]
	for _, c := range s.state.Contracts {
		if c.Deadline > s.state.Clock {
			kept = append(kept, c)
		} else if t := s.contractTemplate(c.Template); t != nil {
			// A missed contract costs nothing but a little reputation.
			s.state.Affinity[t.NPC] = math.Max(0, s.state.Affinity[t.NPC]-affinityFailStep)
		}
	}
	s.state.Contracts = kept
	for len(s.state.Contracts) < maxOpenContracts && s.state.Clock >= s.state.NextContractAt {
		s.spawnContract()
		s.state.NextContractAt = s.state.Clock + contractEvery
	}
}

func (s *Sim) spawnContract() {
	var free []*content.Contract
	for i := range s.Era.Contracts {
		t := &s.Era.Contracts[i]
		if !slices.ContainsFunc(s.state.Contracts, func(c save.ContractState) bool { return c.Template == t.ID }) {
			free = append(free, t)
		}
	}
	if len(free) == 0 {
		return
	}
	s.state.ContractSeq++
	rng := rand.New(rand.NewPCG(s.state.Seed, s.state.ContractSeq))
	t := free[rng.IntN(len(free))]
	r, ok := s.resource(t.Resource)
	if !ok {
		return
	}
	// The order grows with the player: half of what they make during its delay.
	amount := math.Ceil(math.Max(t.Amount, s.producedRate(t.Resource)*t.Seconds/2))
	reward := amount * r.BasePrice * t.Premium * (1 + s.bonus.contractReward)
	s.state.Contracts = append(s.state.Contracts, save.ContractState{
		Template: t.ID, Amount: amount, Reward: reward, Deadline: s.state.Clock + t.Seconds,
	})
}

// producedRate is the gross output of a resource per second.
func (s *Sim) producedRate(id string) float64 {
	i, ok := s.resIdx[id]
	if !ok {
		return 0
	}
	total := 0.0
	for _, st := range s.stations {
		for _, o := range st.outputs {
			if o.res == i {
				total += o.amount * st.cycles * st.theta
			}
		}
	}
	return total
}

// DeliverContract fulfils an open delivery order.
func (s *Sim) DeliverContract(index int) error {
	if index < 0 || index >= len(s.state.Contracts) {
		return fmt.Errorf("%w: contract %d", ErrUnknown, index)
	}
	c := s.state.Contracts[index]
	t := s.contractTemplate(c.Template)
	if t == nil {
		return fmt.Errorf("%w: %s", ErrUnknown, c.Template)
	}
	if err := s.pay(content.Amounts{t.Resource: c.Amount}); err != nil {
		return err
	}
	s.stock[s.resIdx[s.Era.Currency]] += c.Reward
	s.state.Affinity[t.NPC] += affinityStep
	s.state.Contracts = slices.Delete(s.state.Contracts, index, index+1)
	return nil
}
