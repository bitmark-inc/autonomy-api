package schema

import (
	"encoding/json"
)

type GoodBehaviorType string

// OfficialBehaviorMatrix is a map which key is GoodBehavior.ID and value is a object of GoodBehavior
var OfficialBehaviorMatrix = map[GoodBehaviorType]Behavior{
	GoodBehaviorType(CleanHand):        OfficialBehaviors[0],
	GoodBehaviorType(SocialDistancing): OfficialBehaviors[1],
	GoodBehaviorType(TouchFace):        OfficialBehaviors[2],
	GoodBehaviorType(WearMask):         OfficialBehaviors[3],
	GoodBehaviorType(CoveringCough):    OfficialBehaviors[4],
	GoodBehaviorType(CleanSurface):     OfficialBehaviors[5],
}

// DefaultBehaviorWeightMatrix is a map which key is GoodBehavior.ID and value is a object of GoodBehavior
var DefaultBehaviorWeightMatrix = map[GoodBehaviorType]BehaviorWeight{
	GoodBehaviorType(CleanHand):        {ID: GoodBehaviorType(CleanHand), Weight: 1},
	GoodBehaviorType(SocialDistancing): {ID: GoodBehaviorType(SocialDistancing), Weight: 1},
	GoodBehaviorType(TouchFace):        {ID: GoodBehaviorType(TouchFace), Weight: 1},
	GoodBehaviorType(WearMask):         {ID: GoodBehaviorType(WearMask), Weight: 1},
	GoodBehaviorType(CoveringCough):    {ID: GoodBehaviorType(CoveringCough), Weight: 1},
	GoodBehaviorType(CleanSurface):     {ID: GoodBehaviorType(CleanSurface), Weight: 1},
}

const (
	BehaviorCollection          = "behaviors"
	BehaviorReportCollection    = "behaviorReport"
	TotalOfficialBehaviorWeight = float64(6)
)

type BehaviorSource string

const (
	OfficialBehavior   BehaviorSource = "official"
	CustomizedBehavior BehaviorSource = "customized"
)

const (
	CleanHand        GoodBehaviorType = "clean_hand"
	SocialDistancing GoodBehaviorType = "social_distancing"
	TouchFace        GoodBehaviorType = "touch_face"
	WearMask         GoodBehaviorType = "wear_mask"
	CoveringCough    GoodBehaviorType = "covering_coughs"
	CleanSurface     GoodBehaviorType = "clean_surface"
)

// Behavior a struct to define a good behavior
type Behavior struct {
	ID     GoodBehaviorType `json:"id" bson:"_id"`
	Name   string           `json:"name"  bson:"name"`
	Desc   string           `json:"desc"  bson:"desc"`
	Source BehaviorSource   `json:"-" bson:"source"`
}

// BehaviorWeight a struct to define a good behavior weight
type BehaviorWeight struct {
	ID     GoodBehaviorType `json:"id"`
	Weight float64          `json:"weight"`
}

// OfficialBehaviors return a slice that contains all GoodBehavior
var OfficialBehaviors = []Behavior{
	{CleanHand, "Frequent hand cleaning", "Washing hands thoroughly with soap and water for at least 20 seconds or applying an alcohol-based hand sanitizer", OfficialBehavior},
	{SocialDistancing, "Social & physical distancing", "Avoiding crowds, working from home, and maintaining at least 6 feet of distance from others whenever possibl", OfficialBehavior},
	{TouchFace, "Avoiding touching face", "Restraining from touching your eyes, nose, or mouth, especially with unwashed hands.", OfficialBehavior},
	{WearMask, "Wearing a face mask or covering", "Covering your nose and mouth when in public or whenever social distancing measures are difficult to maintain.", OfficialBehavior},
	{CoveringCough, "Covering coughs and sneezes", "Covering your mouth with the inside of your elbow or a tissue whenever you cough or sneeze.", OfficialBehavior},
	{CleanSurface, "Cleaning and disinfecting surfaces", "Cleaning and disinfecting frequently touched surfaces daily, such as doorknobs, tables, light switches, and keyboards.", OfficialBehavior},
}

// BehaviorReportData the struct to store citizen data and score
type BehaviorReportData struct {
	ProfileID     string     `json:"profile_id" bson:"profile_id"`
	AccountNumber string     `json:"account_number" bson:"account_number"`
	Behaviors     []Behavior `json:"behaviors" bson:"behaviors"`
	Location      GeoJSON    `json:"location" bson:"location"`
	Timestamp     int64      `json:"ts" bson:"ts"`
}

func (b *BehaviorReportData) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Behaviors []Behavior `json:"behaviors"`
		Location  Location   `json:"location"`
		Timestamp int64      `json:"timestamp"`
	}{
		Behaviors: b.Behaviors,
		Location:  Location{Longitude: b.Location.Coordinates[0], Latitude: b.Location.Coordinates[1]},
		Timestamp: b.Timestamp,
	})
}

// SplitBehaviors separates official and non-official behaviors
func SplitBehaviors(behaviors []Behavior) ([]Behavior, []Behavior) {
	official := make([]Behavior, 0)
	nonOfficial := make([]Behavior, 0)
	for _, s := range behaviors {
		if _, ok := OfficialBehaviorMatrix[s.ID]; ok {
			official = append(official, s)
		} else {
			nonOfficial = append(nonOfficial, s)
		}
	}

	return official, nonOfficial
}
