// Package content defines the data schema for eras (resources, stations,
// recipes, upgrade tree, quests) and loads it from embedded JSON files.
//
// All gameplay content lives in data files so that adding an era requires no
// new code. Player-facing strings are referenced by key and resolved through
// the locale tables (see Locales).
package content

// ResourceKind classifies a resource by its role in the era economy.
type ResourceKind string

const (
	KindBase         ResourceKind = "base"         // produced without input
	KindIntermediate ResourceKind = "intermediate" // produced by a recipe, used as input
	KindCritical     ResourceKind = "critical"     // required by Era Keys
	KindCurrency     ResourceKind = "currency"     // the era's trading currency
)

// Axis is one of the four ending axes shared by every era.
type Axis string

const (
	AxisTrade      Axis = "trade"      // Négoce
	AxisThroughput Axis = "throughput" // Cadence
	AxisIngenuity  Axis = "ingenuity"  // Ingéniosité
	AxisCommunity  Axis = "community"  // Communauté
)

// Axes lists the four ending axes in their canonical (ring) order.
var Axes = []Axis{AxisTrade, AxisThroughput, AxisIngenuity, AxisCommunity}

// BranchTrunk is the shared trunk of every era tree.
const BranchTrunk = "trunk"

// Amounts maps resource IDs to quantities (costs, recipe inputs/outputs).
type Amounts map[string]float64

// Era is one chapter of the timeline.
type Era struct {
	ID        string     `json:"id"`
	Index     int        `json:"index"`
	NameKey   string     `json:"nameKey"`
	Currency  string     `json:"currency"`
	Resources []Resource `json:"resources"`
	Stations  []Station  `json:"stations"`
	Tree      []Node     `json:"tree"`
	Quests    []Quest    `json:"quests"`
}

// Resource is anything the player can hold, produce or sell.
type Resource struct {
	ID         string       `json:"id"`
	NameKey    string       `json:"nameKey"`
	Kind       ResourceKind `json:"kind"`
	BasePrice  float64      `json:"basePrice"`  // P0 of the market formula
	Elasticity float64      `json:"elasticity"` // epsilon of the market formula
}

// Station is a building that occupies a slot of the panorama.
type Station struct {
	ID         string    `json:"id"`
	NameKey    string    `json:"nameKey"`
	UnlockCost Amounts   `json:"unlockCost"`
	Variants   []Variant `json:"variants"`
}

// Variant is one of the 2-3 specializations chosen when a station unlocks.
type Variant struct {
	ID      string `json:"id"`
	NameKey string `json:"nameKey"`
	Recipe  Recipe `json:"recipe"`
}

// Recipe turns inputs into outputs every cycle. A recipe with no inputs is
// plain production.
type Recipe struct {
	Inputs  Amounts `json:"inputs,omitempty"`
	Outputs Amounts `json:"outputs"`
	Seconds float64 `json:"seconds"`
}

// Node is one upgrade of the era tree.
//
// RequiresAll must all be bought; if RequiresAny is non-empty, at least one of
// them must be bought too. RequiresAny is how parallel paths converge.
type Node struct {
	ID          string   `json:"id"`
	NameKey     string   `json:"nameKey"`
	Branch      string   `json:"branch"` // BranchTrunk or an Axis
	Cost        Amounts  `json:"cost"`
	RequiresAll []string `json:"requiresAll,omitempty"`
	RequiresAny []string `json:"requiresAny,omitempty"`
	EraKey      bool     `json:"eraKey,omitempty"` // final node of a branch
}

// QuestKind is the family a quest belongs to.
type QuestKind string

const (
	QuestProduction QuestKind = "production"
	QuestContract   QuestKind = "contract"
	QuestNarrative  QuestKind = "narrative"
)

// Quest is an objective. Main quests are required to complete the era.
type Quest struct {
	ID      string    `json:"id"`
	NameKey string    `json:"nameKey"`
	Kind    QuestKind `json:"kind"`
	Main    bool      `json:"main"`
	Goal    Amounts   `json:"goal"`
}
