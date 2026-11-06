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
		if r.Kind != KindCurrency && (r.BasePrice <= 0 || r.Elasticity <= 0) {
			add("%s: resource %q needs a positive basePrice and elasticity", where, r.ID)
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

	stationIDs := map[string]bool{}
	for _, s := range e.Stations {
		sw := where + " station " + s.ID
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
		}
	}

	validateTree(e, where, add, checkKey, checkAmounts)

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
		checkAmounts(qw+" goal", q.Goal, false)
	}
}

func validateTree(e Era, where string, add func(string, ...any), checkKey func(string, string), checkAmounts func(string, Amounts, bool)) {
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
