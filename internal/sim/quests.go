package sim

import (
	"fmt"
	"slices"

	"github.com/morialkar/industrimental/internal/content"
)

const contractRetryDelay = 3600.0 // a missed main contract comes back an hour later

// QuestDone tells whether a quest is finished.
func (s *Sim) QuestDone(id string) bool { return slices.Contains(s.state.Quests, id) }

// QuestStep returns the index of a quest's current step.
func (s *Sim) QuestStep(id string) int { return s.state.QuestSteps[id] }

// QuestDeadline returns the seconds left on a contract quest, and whether one
// is running.
func (s *Sim) QuestDeadline(id string) (float64, bool) {
	d, ok := s.state.QuestDeadlines[id]
	return d - s.state.Clock, ok
}

func (s *Sim) questOpen(q *content.Quest) bool {
	if s.QuestDone(q.ID) {
		return false
	}
	for _, r := range q.Requires {
		if !s.QuestDone(r) {
			return false
		}
	}
	return s.state.Clock >= s.state.QuestRetry[q.ID]
}

// ActiveQuests lists the quests the player can work on: at most three main
// quests, in file order, to keep a clear heading, plus the optional ones.
func (s *Sim) ActiveQuests() []*content.Quest {
	var out []*content.Quest
	main := 0
	for i := range s.Era.Quests {
		q := &s.Era.Quests[i]
		if !s.questOpen(q) {
			continue
		}
		if q.Main {
			if main >= maxActiveMain {
				continue
			}
			main++
		}
		out = append(out, q)
	}
	return out
}

func (s *Sim) goalMet(g *content.Goal) bool {
	if len(g.Any) > 0 {
		return slices.ContainsFunc(g.Any, func(alt content.Goal) bool { return s.goalMet(&alt) })
	}
	for id, q := range g.Have {
		if s.Stock(id) < q {
			return false
		}
	}
	for id, q := range g.Deliver {
		if s.Stock(id) < q {
			return false
		}
	}
	for id, q := range g.Produced {
		if s.Produced(id) < q {
			return false
		}
	}
	if s.state.Earned < g.Earned {
		return false
	}
	for _, id := range g.Stations {
		if !s.Station(id).Built {
			return false
		}
	}
	for id, lvl := range g.Levels {
		if s.Station(id).Level < lvl {
			return false
		}
	}
	return !slices.ContainsFunc(g.Nodes, func(id string) bool { return !s.HasNode(id) })
}

// needsDelivery tells whether a goal closes only when the player hands goods
// over, rather than by itself.
func needsDelivery(g *content.Goal) bool {
	if len(g.Any) > 0 {
		return slices.ContainsFunc(g.Any, func(alt content.Goal) bool { return needsDelivery(&alt) })
	}
	return len(g.Deliver) > 0
}

// DeliverableAmount returns what handing over a goal would take now.
func (s *Sim) deliverable(g *content.Goal) (content.Amounts, bool) {
	if len(g.Any) > 0 {
		for i := range g.Any {
			if a, ok := s.deliverable(&g.Any[i]); ok {
				return a, true
			}
		}
		return nil, false
	}
	return g.Deliver, s.goalMet(g)
}

// CurrentStep returns the step a quest is at.
func (s *Sim) CurrentStep(q *content.Quest) *content.Step {
	i := s.state.QuestSteps[q.ID]
	if i >= len(q.Steps) {
		i = len(q.Steps) - 1
	}
	return &q.Steps[i]
}

// CanDeliver tells whether a quest's current step awaits goods the player holds.
func (s *Sim) CanDeliver(q *content.Quest) bool {
	g := &s.CurrentStep(q).Goal
	if !needsDelivery(g) {
		return false
	}
	_, ok := s.deliverable(g)
	return ok
}

// Deliver hands over the goods a quest's current step asks for.
func (s *Sim) Deliver(id string) error {
	q, ok := s.quests[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknown, id)
	}
	if !slices.Contains(s.ActiveQuests(), q) {
		return ErrLocked
	}
	if !s.CanDeliver(q) {
		return ErrAfford
	}
	a, _ := s.deliverable(&s.CurrentStep(q).Goal)
	if err := s.pay(a); err != nil {
		return err
	}
	s.advanceQuest(q)
	s.updateQuests()
	return nil
}

func (s *Sim) advanceQuest(q *content.Quest) {
	s.state.QuestSteps[q.ID]++
	if s.state.QuestSteps[q.ID] < len(q.Steps) {
		return
	}
	s.state.Quests = append(s.state.Quests, q.ID)
	delete(s.state.QuestDeadlines, q.ID)
	s.gain(q.Reward, 1+s.bonus.questReward)
	if q.NPC != "" {
		s.state.Affinity[q.NPC] += affinityStep
	}
}

// updateQuests closes the steps whose goal is met and runs contract delays.
func (s *Sim) updateQuests() {
	for changed := true; changed; {
		changed = false
		for _, q := range s.ActiveQuests() {
			if q.Kind == content.QuestContract {
				if _, running := s.state.QuestDeadlines[q.ID]; !running {
					s.state.QuestDeadlines[q.ID] = s.state.Clock + q.Seconds
				}
				if s.state.QuestDeadlines[q.ID] <= s.state.Clock {
					s.failContract(q)
					changed = true
					continue
				}
			}
			g := &s.CurrentStep(q).Goal
			if !needsDelivery(g) && s.goalMet(g) {
				s.advanceQuest(q)
				changed = true
			}
		}
	}
}

func (s *Sim) failContract(q *content.Quest) {
	delete(s.state.QuestDeadlines, q.ID)
	s.state.QuestRetry[q.ID] = s.state.Clock + contractRetryDelay
	if q.NPC != "" && s.state.Affinity[q.NPC] >= affinityFailStep {
		s.state.Affinity[q.NPC] -= affinityFailStep
	}
}
