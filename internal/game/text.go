package game

import (
	"errors"
	"fmt"
	"image"
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/morialkar/industrimental/internal/content"
	"github.com/morialkar/industrimental/internal/sim"
	"github.com/morialkar/industrimental/internal/ui"
)

func (g *Game) text(key string) string { return g.bundle.Text(g.locale, key) }
func (g *Game) tr(key string) string   { return g.bundle.Text(g.locale, key) }

func (g *Game) resName(id string) string {
	for i := range g.era.Resources {
		if g.era.Resources[i].ID == id {
			return g.text(g.era.Resources[i].NameKey)
		}
	}
	return id
}

func (g *Game) stationDef(id string) *content.Station {
	for i := range g.era.Stations {
		if g.era.Stations[i].ID == id {
			return &g.era.Stations[i]
		}
	}
	return nil
}

func (g *Game) stationName(id string) string {
	if def := g.stationDef(id); def != nil {
		return g.text(def.NameKey)
	}
	return id
}

func (g *Game) nodeDef(id string) *content.Node {
	for i := range g.era.Tree {
		if g.era.Tree[i].ID == id {
			return &g.era.Tree[i]
		}
	}
	return nil
}

func (g *Game) nodeName(id string) string {
	if n := g.nodeDef(id); n != nil {
		return g.text(n.NameKey)
	}
	return id
}

func (g *Game) npcName(id string) string {
	for i := range g.era.NPCs {
		if g.era.NPCs[i].ID == id {
			return g.text(g.era.NPCs[i].NameKey)
		}
	}
	return id
}

func (g *Game) amountsText(a content.Amounts) string {
	if len(a) == 0 {
		return g.tr("ui.free")
	}
	parts := make([]string, 0, len(a))
	for _, id := range sortedAmounts(a) {
		parts = append(parts, sim.Format(a[id])+" "+strings.ToLower(g.resName(id)))
	}
	return strings.Join(parts, " · ")
}

func sortedAmounts(a content.Amounts) []string {
	keys := make([]string, 0, len(a))
	for k := range a {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (g *Game) act(err error) {
	switch {
	case err == nil:
	case errors.Is(err, sim.ErrAfford):
		g.say(g.tr("ui.toast.afford"))
	default:
		g.say(g.tr("ui.toast.locked"))
	}
}

func (g *Game) say(s string) { g.toast, g.toastT = s, 150 }

func (g *Game) variantOf(def *content.Station, id string) *content.Variant {
	for i := range def.Variants {
		if def.Variants[i].ID == id {
			return &def.Variants[i]
		}
	}
	return nil
}

func (g *Game) statusText(st sim.StationStatus) string {
	switch {
	case st.Retooling > 0:
		return g.tr("ui.retooling") + " : " + sim.FormatDuration(st.Retooling)
	case st.Limiting != "":
		return fmt.Sprintf("%s : %s (%.0f %%)", g.tr("ui.limiting"), g.resName(st.Limiting), st.Speed*100)
	default:
		return fmt.Sprintf("%s : ×%s", g.tr("ui.speed"), sim.Format(st.Cycles))
	}
}

func easeOut(t float64) float64 { return 1 - (1-t)*(1-t)*(1-t) }

// effectsText describes effects, each value multiplied by f.
func (g *Game) effectsText(effects []content.Effect, f float64) string {
	parts := make([]string, 0, len(effects))
	for _, fx := range effects {
		target := ""
		switch {
		case fx.Station == "*" || fx.Resource == "*":
			target = g.tr("ui.fx.all")
		case fx.Station != "":
			target = g.stationName(fx.Station)
		case fx.Resource != "":
			target = g.resName(fx.Resource)
		}
		sign := "+"
		if fx.Kind == content.EffectRetoolDiscount || fx.Kind == content.EffectInputSaving {
			sign = "-"
		}
		v := fx.Value * f * 100
		num := fmt.Sprintf("%.0f", v)
		if v < 10 && v != math.Trunc(v) {
			num = fmt.Sprintf("%.1f", v)
		}
		s := fmt.Sprintf("%s%s %% %s", sign, num, g.tr("ui.fx."+string(fx.Kind)))
		if target != "" {
			s += " " + target
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " · ")
}

func signed(v float64) string {
	if v >= 0 {
		return "+" + sim.Format(v)
	}
	return sim.Format(v)
}

func sortedBuys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func goalDelivers(goal content.Goal) bool {
	if len(goal.Any) > 0 {
		return slices.ContainsFunc(goal.Any, goalDelivers)
	}
	return len(goal.Deliver) > 0
}

// goalText says what a goal still asks for, with the current progress.
func (g *Game) goalText(goal content.Goal) string {
	if len(goal.Any) > 0 {
		parts := make([]string, 0, len(goal.Any))
		for _, alt := range goal.Any {
			parts = append(parts, g.goalText(alt))
		}
		return strings.Join(parts, " "+g.tr("ui.goal.or")+" ")
	}
	var parts []string
	progress := func(verb string, a content.Amounts, cur func(string) float64) {
		for _, id := range sortedAmounts(a) {
			parts = append(parts, fmt.Sprintf("%s %s / %s %s", verb, sim.Format(math.Min(cur(id), a[id])), sim.Format(a[id]), strings.ToLower(g.resName(id))))
		}
	}
	progress(g.tr("ui.goal.have"), goal.Have, g.sim.Stock)
	progress(g.tr("ui.goal.deliver"), goal.Deliver, g.sim.Stock)
	progress(g.tr("ui.goal.produced"), goal.Produced, g.sim.Produced)
	if goal.Earned > 0 {
		parts = append(parts, fmt.Sprintf("%s %s / %s %s", g.tr("ui.goal.earned"), sim.Format(math.Min(g.sim.State().Earned, goal.Earned)), sim.Format(goal.Earned), strings.ToLower(g.resName(g.era.Currency))))
	}
	for _, id := range goal.Stations {
		parts = append(parts, g.tr("ui.goal.station")+" "+g.stationName(id))
	}
	levels := make([]string, 0, len(goal.Levels))
	for id := range goal.Levels {
		levels = append(levels, id)
	}
	sort.Strings(levels)
	for _, id := range levels {
		parts = append(parts, fmt.Sprintf("%s %s : %d / %d", g.tr("ui.goal.level"), g.stationName(id), g.sim.Station(id).Level, goal.Levels[id]))
	}
	for _, id := range goal.Nodes {
		parts = append(parts, g.tr("ui.goal.node")+" "+g.nodeName(id))
	}
	return strings.Join(parts, " · ")
}

// goalProgress measures a goal from 0 to 1, with a short "x / y" count when
// the goal is a single number. An alternative counts by its best option.
func (g *Game) goalProgress(goal content.Goal) (float64, string) {
	if len(goal.Any) > 0 {
		best, label := 0.0, ""
		for _, alt := range goal.Any {
			if v, l := g.goalProgress(alt); v >= best {
				best, label = v, l
			}
		}
		return best, label
	}
	var parts []float64
	count := ""
	add := func(cur, want float64) {
		parts = append(parts, math.Min(1, cur/math.Max(want, 1e-9)))
		count = sim.Format(math.Min(cur, want)) + " / " + sim.Format(want)
	}
	for id, q := range goal.Have {
		add(g.sim.Stock(id), q)
	}
	for id, q := range goal.Deliver {
		add(g.sim.Stock(id), q)
	}
	for id, q := range goal.Produced {
		add(g.sim.Produced(id), q)
	}
	if goal.Earned > 0 {
		add(g.sim.State().Earned, goal.Earned)
	}
	for _, id := range goal.Stations {
		done := 0.0
		if g.sim.Station(id).Built {
			done = 1
		}
		add(done, 1)
	}
	for id, lvl := range goal.Levels {
		add(float64(g.sim.Station(id).Level), float64(lvl))
	}
	for _, id := range goal.Nodes {
		done := 0.0
		if g.sim.HasNode(id) {
			done = 1
		}
		add(done, 1)
	}
	if len(parts) == 0 {
		return 0, ""
	}
	sum := 0.0
	for _, p := range parts {
		sum += p
	}
	if len(parts) > 1 {
		count = fmt.Sprintf("%.0f %%", 100*sum/float64(len(parts)))
	}
	return sum / float64(len(parts)), count
}

// questProgress is the progress of a quest: finished steps plus the current one.
func (g *Game) questProgress(q *content.Quest) (float64, string) {
	step := g.sim.QuestStep(q.ID)
	v, count := g.goalProgress(g.sim.CurrentStep(q).Goal)
	if len(q.Steps) > 1 {
		return (float64(step) + v) / float64(len(q.Steps)), fmt.Sprintf("%d/%d · %s", step+1, len(q.Steps), count)
	}
	return v, count
}

func rectI(r ui.Rect) image.Rectangle {
	return image.Rect(int(r.X), int(r.Y), int(math.Ceil(r.X+r.W)), int(math.Ceil(r.Y+r.H)))
}
