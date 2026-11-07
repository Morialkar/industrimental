package content

import (
	"errors"
	"fmt"
	"slices"
)

// Validate checks the internal consistency of the content: unique IDs,
// existing references, positive quantities, a well-formed tree with exactly
// one Era Key per axis, and every text key translated in every locale.
func Validate(b *Bundle) error {
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	for _, loc := range SupportedLocales {
		if _, ok := b.Locales[loc]; !ok {
			add("missing locale table %q", loc)
		}
	}
	checkKey := func(where, key string) {
		if key == "" {
			add("%s: empty text key", where)
			return
		}
		for _, loc := range SupportedLocales {
			if table, ok := b.Locales[loc]; ok {
				if _, ok := table[key]; !ok {
					add("%s: text key %q missing in locale %q", where, key, loc)
				}
			}
		}
	}

	eraIDs := map[string]bool{}
	for _, e := range b.Eras {
		if eraIDs[e.ID] {
			add("duplicate era id %q", e.ID)
		}
		eraIDs[e.ID] = true
		validateEra(e, add, checkKey)
	}
	return errors.Join(errs...)
}

func validateEra(e Era, add func(string, ...any), checkKey func(string, string)) {
	where := "era " + e.ID
	checkKey(where, e.NameKey)

	res := map[string]Resource{}
	for _, r := range e.Resources {
		if _, dup := res[r.ID]; dup {
			add("%s: duplicate resource %q", where, r.ID)
		}
		res[r.ID] = r
		checkKey(where+" resource "+r.ID, r.NameKey)
		switch r.Kind {
		case KindBase, KindIntermediate, KindCritical, KindCurrency:
		default:
			add("%s: resource %q has unknown kind %q", where, r.ID, r.Kind)
		}
		if r.Kind != KindCurrency && (r.BasePrice <= 0 || r.Elasticity <= 0 || r.Demand <= 0) {
			add("%s: resource %q needs a positive basePrice, elasticity and demand", where, r.ID)
		}
		if r.DescKey != "" {
			checkKey(where+" resource "+r.ID, r.DescKey)
		}
	}
	if c, ok := res[e.Currency]; !ok || c.Kind != KindCurrency {
		add("%s: currency %q is not a currency resource of the era", where, e.Currency)
	}

	checkAmounts := func(ctx string, a Amounts, allowEmpty bool) {
		if len(a) == 0 && !allowEmpty {
			add("%s: must not be empty", ctx)
		}
		for id, q := range a {
			if _, ok := res[id]; !ok {
				add("%s: unknown resource %q", ctx, id)
			}
			if q <= 0 {
				add("%s: quantity of %q must be positive", ctx, id)
			}
		}
	}

	nodeIDs := map[string]bool{}
	for _, n := range e.Tree {
		nodeIDs[n.ID] = true
	}
	stationSet := map[string]bool{}
	for _, s := range e.Stations {
		stationSet[s.ID] = true
	}
	checkEffects := func(ctx string, effects []Effect) {
		for _, fx := range effects {
			switch fx.Kind {
			case EffectStationRate:
				if fx.Station != "*" && !stationSet[fx.Station] {
					add("%s: effect on unknown station %q", ctx, fx.Station)
				}
			case EffectResourceRate, EffectInputSaving:
				if _, ok := res[fx.Resource]; !ok {
					add("%s: effect on unknown resource %q", ctx, fx.Resource)
				}
			case EffectSellPrice, EffectDemand:
				if _, ok := res[fx.Resource]; !ok && fx.Resource != "*" {
					add("%s: effect on unknown resource %q", ctx, fx.Resource)
				}
			case EffectContractReward, EffectQuestReward, EffectRetoolDiscount:
			default:
				add("%s: unknown effect kind %q", ctx, fx.Kind)
			}
			if fx.Value <= 0 {
				add("%s: effect %q needs a positive value", ctx, fx.Kind)
			}
		}
	}
	marketStations := 0

	stationIDs := map[string]bool{}
	for _, s := range e.Stations {
		sw := where + " station " + s.ID
		if s.Market {
			marketStations++
		}
		if s.RequiresNode != "" && !nodeIDs[s.RequiresNode] {
			add("%s: requires unknown node %q", sw, s.RequiresNode)
		}
		checkAmounts(sw+" levelCost", s.LevelCost, true)
		checkAmounts(sw+" harvest", s.Harvest, true)
		if s.DescKey != "" {
			checkKey(sw, s.DescKey)
		}
		if stationIDs[s.ID] {
			add("%s: duplicate station", sw)
		}
		stationIDs[s.ID] = true
		checkKey(sw, s.NameKey)
		checkAmounts(sw+" unlockCost", s.UnlockCost, true)
		if n := len(s.Variants); n < 2 || n > 3 {
			add("%s: needs 2 or 3 variants, has %d", sw, n)
		}
		varIDs := map[string]bool{}
		for _, v := range s.Variants {
			vw := sw + " variant " + v.ID
			if varIDs[v.ID] {
				add("%s: duplicate variant", vw)
			}
			varIDs[v.ID] = true
			checkKey(vw, v.NameKey)
			checkAmounts(vw+" inputs", v.Recipe.Inputs, true)
			checkAmounts(vw+" outputs", v.Recipe.Outputs, false)
			if v.Recipe.Seconds <= 0 {
				add("%s: recipe seconds must be positive", vw)
			}
			checkEffects(vw, v.Effects)
		}
	}

	if marketStations > 1 {
		add("%s: at most one station may open the market, %d do", where, marketStations)
	}

	validateTree(e, where, add, checkKey, checkAmounts, checkEffects)

	npcs := map[string]bool{}
	for _, n := range e.NPCs {
		nw := where + " npc " + n.ID
		if npcs[n.ID] {
			add("%s: duplicate npc", nw)
		}
		npcs[n.ID] = true
		checkKey(nw, n.NameKey)
		if n.DescKey != "" {
			checkKey(nw, n.DescKey)
		}
		if n.RequiresStation != "" && !stationSet[n.RequiresStation] {
			add("%s: requires unknown station %q", nw, n.RequiresStation)
		}
		for id := range n.Buys {
			if _, ok := res[id]; !ok {
				add("%s buys: unknown resource %q", nw, id)
			}
		}
		for _, d := range n.Deals {
			checkKey(nw+" deal "+d.ID, d.NameKey)
			checkAmounts(nw+" deal "+d.ID+" give", d.Give, false)
			checkAmounts(nw+" deal "+d.ID+" get", d.Get, false)
		}
	}

	for _, c := range e.Contracts {
		cw := where + " contract " + c.ID
		checkKey(cw, c.NameKey)
		if !npcs[c.NPC] {
			add("%s: unknown npc %q", cw, c.NPC)
		}
		if _, ok := res[c.Resource]; !ok {
			add("%s: unknown resource %q", cw, c.Resource)
		}
		if c.Amount <= 0 || c.Premium <= 0 || c.Seconds <= 0 {
			add("%s: amount, premium and seconds must be positive", cw)
		}
	}

	endings := map[Axis]bool{}
	for _, end := range e.Endings {
		ew := where + " ending " + string(end.Axis)
		if !slices.Contains(Axes, end.Axis) || endings[end.Axis] {
			add("%s: unknown or duplicate axis", ew)
		}
		endings[end.Axis] = true
		checkKey(ew, end.NameKey)
		checkKey(ew, end.PatentKey)
		checkKey(ew, end.PatentDescKey)
		if end.DescKey != "" {
			checkKey(ew, end.DescKey)
		}
	}
	for _, axis := range Axes {
		if !endings[axis] {
			add("%s: missing ending for axis %q", where, axis)
		}
	}

	var checkGoal func(ctx string, g Goal)
	checkGoal = func(ctx string, g Goal) {
		checkAmounts(ctx+" have", g.Have, true)
		checkAmounts(ctx+" produced", g.Produced, true)
		checkAmounts(ctx+" deliver", g.Deliver, true)
		for _, id := range g.Stations {
			if !stationSet[id] {
				add("%s: unknown station %q", ctx, id)
			}
		}
		for id := range g.Levels {
			if !stationSet[id] {
				add("%s: unknown station %q", ctx, id)
			}
		}
		for _, id := range g.Nodes {
			if !nodeIDs[id] {
				add("%s: unknown node %q", ctx, id)
			}
		}
		empty := len(g.Have)+len(g.Produced)+len(g.Deliver)+len(g.Stations)+len(g.Levels)+len(g.Nodes)+len(g.Any) == 0 && g.Earned == 0
		if empty {
			add("%s: goal must not be empty", ctx)
		}
		for _, alt := range g.Any {
			checkGoal(ctx+" alternative", alt)
		}
	}

	questIDs := map[string]bool{}
	for _, q := range e.Quests {
		qw := where + " quest " + q.ID
		if questIDs[q.ID] {
			add("%s: duplicate quest", qw)
		}
		questIDs[q.ID] = true
		checkKey(qw, q.NameKey)
		switch q.Kind {
		case QuestProduction, QuestContract, QuestNarrative:
		default:
			add("%s: unknown kind %q", qw, q.Kind)
		}
		if q.NPC != "" && !npcs[q.NPC] {
			add("%s: unknown npc %q", qw, q.NPC)
		}
		if q.Branch != "" && !slices.Contains(Axes, Axis(q.Branch)) {
			add("%s: unknown branch %q", qw, q.Branch)
		}
		if q.Kind == QuestContract && q.Seconds <= 0 {
			add("%s: a contract needs a delay in seconds", qw)
		}
		if len(q.Steps) == 0 {
			add("%s: needs at least one step", qw)
		}
		for i, step := range q.Steps {
			sw := fmt.Sprintf("%s step %d", qw, i)
			checkKey(sw, step.TextKey)
			checkGoal(sw, step.Goal)
		}
		checkAmounts(qw+" reward", q.Reward, true)
	}
	for _, q := range e.Quests {
		for _, r := range q.Requires {
			if !questIDs[r] {
				add("%s quest %s: requires unknown quest %q", where, q.ID, r)
			}
		}
	}
}

func validateTree(e Era, where string, add func(string, ...any), checkKey func(string, string), checkAmounts func(string, Amounts, bool), checkEffects func(string, []Effect)) {
	nodes := map[string]Node{}
	for _, n := range e.Tree {
		if _, dup := nodes[n.ID]; dup {
			add("%s: duplicate tree node %q", where, n.ID)
		}
		nodes[n.ID] = n
	}

	keys := map[string]int{}
	for _, n := range e.Tree {
		nw := where + " node " + n.ID
		checkKey(nw, n.NameKey)
		checkAmounts(nw+" cost", n.Cost, false)
		checkEffects(nw, n.Effects)
		if n.DescKey != "" {
			checkKey(nw, n.DescKey)
		}
		if n.BridgeTo != "" && (n.Branch == BranchTrunk || !slices.Contains(Axes, Axis(n.BridgeTo)) || n.BridgeTo == n.Branch) {
			add("%s: invalid bridge to %q", nw, n.BridgeTo)
		}
		if n.Branch != BranchTrunk && !slices.Contains(Axes, Axis(n.Branch)) {
			add("%s: unknown branch %q", nw, n.Branch)
		}
		if n.EraKey {
			if n.Branch == BranchTrunk {
				add("%s: the trunk cannot hold an Era Key", nw)
			}
			keys[n.Branch]++
		}
		for _, r := range append(slices.Clone(n.RequiresAll), n.RequiresAny...) {
			if _, ok := nodes[r]; !ok {
				add("%s: requires unknown node %q", nw, r)
			}
		}
		if len(n.RequiresAll)+len(n.RequiresAny) == 0 && n.Branch != BranchTrunk {
			add("%s: only trunk nodes may be roots", nw)
		}
	}
	for _, axis := range Axes {
		if keys[string(axis)] != 1 {
			add("%s: branch %q needs exactly one Era Key, has %d", where, axis, keys[string(axis)])
		}
	}

	// Cycle detection (depth-first, three colors).
	const (
		white = iota
		grey
		black
	)
	color := map[string]int{}
	var visit func(id string) bool
	visit = func(id string) bool {
		switch color[id] {
		case grey:
			return false
		case black:
			return true
		}
		color[id] = grey
		n := nodes[id]
		for _, r := range append(slices.Clone(n.RequiresAll), n.RequiresAny...) {
			if _, ok := nodes[r]; ok && !visit(r) {
				return false
			}
		}
		color[id] = black
		return true
	}
	for _, n := range e.Tree {
		if !visit(n.ID) {
			add("%s: tree has a cycle through node %q", where, n.ID)
			return
		}
	}
}
