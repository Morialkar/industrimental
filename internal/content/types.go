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
	NPCs      []NPC      `json:"npcs"`
	Contracts []Contract `json:"contracts"`
	Endings   []Ending   `json:"endings"`
}

// Resource is anything the player can hold, produce or sell.
type Resource struct {
	ID         string       `json:"id"`
	NameKey    string       `json:"nameKey"`
	DescKey    string       `json:"descKey,omitempty"`
	Kind       ResourceKind `json:"kind"`
	BasePrice  float64      `json:"basePrice"`  // P0 of the market formula
	Elasticity float64      `json:"elasticity"` // epsilon of the market formula
	Demand     float64      `json:"demand"`     // D of the market formula, before growth
}

// Station is a building that occupies a slot of the panorama.
type Station struct {
	ID      string `json:"id"`
	NameKey string `json:"nameKey"`
	DescKey string `json:"descKey,omitempty"`
	// RequiresNode, if set, is the tree node that makes the station buildable.
	RequiresNode string  `json:"requiresNode,omitempty"`
	UnlockCost   Amounts `json:"unlockCost"`
	// LevelCost is the cost of going from level 1 to 2; each further level
	// multiplies it by 1.15.
	LevelCost Amounts `json:"levelCost"`
	// RetoolSeconds is how long a variant change keeps the station idle.
	RetoolSeconds float64 `json:"retoolSeconds"`
	// Market marks the station that opens the era's market once built.
	Market bool `json:"market,omitempty"`
	// Harvest, if set, is what one tap on the station gives.
	Harvest  Amounts   `json:"harvest,omitempty"`
	Variants []Variant `json:"variants"`
}

// Variant is one of the 2-3 specializations chosen when a station unlocks.
type Variant struct {
	ID      string   `json:"id"`
	NameKey string   `json:"nameKey"`
	Recipe  Recipe   `json:"recipe"`
	Effects []Effect `json:"effects,omitempty"` // active while the variant runs
}

// Recipe turns inputs into outputs every cycle. A recipe with no inputs is
// plain production.
type Recipe struct {
	Inputs  Amounts `json:"inputs,omitempty"`
	Outputs Amounts `json:"outputs"`
	Seconds float64 `json:"seconds"`
}

// EffectKind names what an Effect changes.
type EffectKind string

const (
	EffectStationRate    EffectKind = "stationRate"    // Station ("*" for all): cycle speed +Value
	EffectResourceRate   EffectKind = "resourceRate"   // Resource: output +Value
	EffectInputSaving    EffectKind = "inputSaving"    // Resource: consumed -Value (at most 75 %)
	EffectSellPrice      EffectKind = "sellPrice"      // Resource ("*" for all): sale price +Value
	EffectDemand         EffectKind = "demand"         // Resource ("*" for all): market demand +Value
	EffectContractReward EffectKind = "contractReward" // contract rewards +Value
	EffectQuestReward    EffectKind = "questReward"    // quest rewards +Value
	EffectRetoolDiscount EffectKind = "retoolDiscount" // retooling cost and time -Value (at most 75 %)
)

// Effect is a passive bonus granted by a tree node or a running variant.
// Value is a fraction: 0.25 means +25 %.
type Effect struct {
	Kind     EffectKind `json:"kind"`
	Station  string     `json:"station,omitempty"`
	Resource string     `json:"resource,omitempty"`
	Value    float64    `json:"value"`
}

// Node is one upgrade of the era tree.
//
// RequiresAll must all be bought; if RequiresAny is non-empty, at least one of
// them must be bought too. RequiresAny is how parallel paths converge.
type Node struct {
	ID          string   `json:"id"`
	NameKey     string   `json:"nameKey"`
	DescKey     string   `json:"descKey,omitempty"`
	Branch      string   `json:"branch"` // BranchTrunk or an Axis
	Path        string   `json:"path,omitempty"`
	BridgeTo    string   `json:"bridgeTo,omitempty"` // a bridge also counts for this neighbor branch
	Cost        Amounts  `json:"cost"`
	RequiresAll []string `json:"requiresAll,omitempty"`
	RequiresAny []string `json:"requiresAny,omitempty"`
	Effects     []Effect `json:"effects,omitempty"`
	EraKey      bool     `json:"eraKey,omitempty"` // final node of a branch
}

// QuestKind is the family a quest belongs to.
type QuestKind string

const (
	QuestProduction QuestKind = "production"
	QuestContract   QuestKind = "contract"
	QuestNarrative  QuestKind = "narrative"
)

// Quest is an objective. Main quests are required to complete the era, and at
// most three are active at a time, in file order.
type Quest struct {
	ID      string    `json:"id"`
	NameKey string    `json:"nameKey"`
	Kind    QuestKind `json:"kind"`
	Main    bool      `json:"main"`
	NPC     string    `json:"npc,omitempty"`
	Branch  string    `json:"branch,omitempty"` // optional quests: the axis they serve
	// Requires lists the quests to finish before this one becomes active.
	Requires []string `json:"requires,omitempty"`
	// Seconds, on a contract, is the delivery delay once the quest is active.
	Seconds float64 `json:"seconds,omitempty"`
	Steps   []Step  `json:"steps"`
	Reward  Amounts `json:"reward,omitempty"`
}

// Step is one stage of a quest: a line of text and the goal that closes it.
type Step struct {
	TextKey string `json:"textKey"`
	Goal    Goal   `json:"goal"`
}

// Goal is a set of conditions that must all hold. With Any set, meeting any
// one alternative is enough, so that every ending can finish neutral quests.
type Goal struct {
	Have     Amounts        `json:"have,omitempty"`     // in stock
	Produced Amounts        `json:"produced,omitempty"` // produced in total
	Deliver  Amounts        `json:"deliver,omitempty"`  // handed over by the player
	Earned   float64        `json:"earned,omitempty"`   // currency earned on the market in total
	Stations []string       `json:"stations,omitempty"` // built
	Levels   map[string]int `json:"levels,omitempty"`   // station -> minimum level
	Nodes    []string       `json:"nodes,omitempty"`    // bought
	Any      []Goal         `json:"any,omitempty"`
}

// NPC is a character of the era: they give quests and may trade.
type NPC struct {
	ID      string `json:"id"`
	NameKey string `json:"nameKey"`
	DescKey string `json:"descKey,omitempty"`
	Axis    Axis   `json:"axis,omitempty"`
	// RequiresStation, if set, is the station that brings them to the market.
	RequiresStation string `json:"requiresStation,omitempty"`
	// Buys multiplies the market price when selling this resource to them.
	Buys  map[string]float64 `json:"buys,omitempty"`
	Deals []Deal             `json:"deals,omitempty"`
}

// Deal is a fixed barter offered by an NPC.
type Deal struct {
	ID      string  `json:"id"`
	NameKey string  `json:"nameKey"`
	Give    Amounts `json:"give"`
	Get     Amounts `json:"get"`
}

// Contract is a template for the repeating delivery orders of the market.
type Contract struct {
	ID       string  `json:"id"`
	NameKey  string  `json:"nameKey"`
	NPC      string  `json:"npc"`
	Resource string  `json:"resource"`
	Amount   float64 `json:"amount"`  // at the start of the era; grows with the player's output
	Premium  float64 `json:"premium"` // reward = market value x Premium
	Seconds  float64 `json:"seconds"`
}

// Ending describes one of the four ends of an era and its Patent.
type Ending struct {
	Axis          Axis   `json:"axis"`
	NameKey       string `json:"nameKey"`
	DescKey       string `json:"descKey,omitempty"`
	PatentKey     string `json:"patentKey"`
	PatentDescKey string `json:"patentDescKey"`
}
