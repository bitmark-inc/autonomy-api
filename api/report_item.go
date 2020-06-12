package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/bitmark-inc/autonomy-api/consts"
	"github.com/bitmark-inc/autonomy-api/geo"
	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/score"
	"github.com/bitmark-inc/autonomy-api/utils"
)

const (
	reportItemScopeIndividual   = "individual"
	reportItemScopeNeighborhood = "neighborhood"
	reportItemScopePOI          = "poi"

	reportItemTypeScore    = "score"
	reportItemTypeSymptom  = "symptom"
	reportItemTypeBehavior = "behavior"
	reportItemTypeCase     = "case"
)

type reportItemQueryParams struct {
	Scope    string `form:"scope"`
	Type     string `form:"type"`
	Start    int64  `form:"start"`
	End      int64  `form:"end"`
	Language string `form:"lang"`
	PoiID    string `form:"poi_id"`
}

type reportItem struct {
	Name       string  `json:"name"`
	Value      int     `json:"value"`
	ChangeRate float64 `json:"change_rate"`
}

func (s *Server) getReportItems(c *gin.Context) {
	account, ok := c.MustGet("account").(*schema.Account)
	if !ok {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
		return
	}

	var params reportItemQueryParams
	if err := c.Bind(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}
	currentPeriodStart := params.Start
	currentPeriodEnd := params.End
	previousPeriodStart := 2*params.Start - params.End
	previousPeriodEnd := params.Start
	localizer := utils.NewLocalizer(params.Language)

	var profileID string
	var loc schema.Location
	var scoreOwner string
	switch params.Scope {
	case reportItemScopeIndividual:
		profileID = account.ProfileID.String()
		scoreOwner = account.AccountNumber
	case reportItemScopeNeighborhood:
		if lastLocation := account.Profile.State.LastLocation; lastLocation == nil {
			abortWithEncoding(c, http.StatusBadRequest, errorUnknownAccountLocation)
			return
		} else {
			loc = *lastLocation
		}
		if loc.Country == "" {
			resolvedLococation, err := geo.PoliticalGeoInfo(loc)
			if err != nil {
				log.WithError(err).WithField("location", loc).Error("failed to fetch geo info")
				abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
				return
			}
			loc = resolvedLococation
		}
	case reportItemScopePOI:
		poiID, err := primitive.ObjectIDFromHex(params.PoiID)
		if err != nil {
			abortWithEncoding(c, http.StatusBadRequest, errorUnknownPOI, err)
			return
		}
		poi, err := s.mongoStore.GetPOI(poiID)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		loc = schema.Location{
			Latitude:  poi.Location.Coordinates[1],
			Longitude: poi.Location.Coordinates[0],
			AddressComponent: schema.AddressComponent{
				Country: poi.Country,
				State:   poi.State,
				County:  poi.County,
			},
		}
		scoreOwner = params.PoiID
	default:
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters)
		return
	}

	switch params.Type {
	case reportItemTypeScore:
		currAvgScore, err := s.mongoStore.GetScoreAverage(scoreOwner, currentPeriodStart, currentPeriodEnd)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		prevAvgScore, err := s.mongoStore.GetScoreAverage(scoreOwner, previousPeriodStart, previousPeriodEnd)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		results := gatherReportItems(
			map[string]int{"autonomy_score": int(currAvgScore)},
			map[string]int{"autonomy_score": int(prevAvgScore)})
		items := getReportItemsForDisplay(results, func(scoreID string) string {
			// TODO: translate
			return scoreID
		})
		c.JSON(http.StatusOK, gin.H{"report_items": items})
	case reportItemTypeSymptom:
		currentDistribution, err := s.mongoStore.FindSymptomDistribution(profileID, &loc, consts.NEARBY_DISTANCE_RANGE, currentPeriodStart, currentPeriodEnd, false)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		previousDistribution, err := s.mongoStore.FindSymptomDistribution(profileID, &loc, consts.NEARBY_DISTANCE_RANGE, previousPeriodStart, previousPeriodEnd, false)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		results := gatherReportItems(currentDistribution, previousDistribution)
		items := getReportItemsForDisplay(results, func(symptomID string) string {
			name, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: fmt.Sprintf("symptoms.%s.name", symptomID)})
			if err == nil {
				return name
			}

			symptoms, _ := s.mongoStore.FindSymptomsByIDs([]string{symptomID})
			if len(symptoms) == 1 {
				return symptoms[0].Name
			}
			return ""
		})
		c.JSON(http.StatusOK, gin.H{"report_items": items})
	case reportItemTypeBehavior:
		currentDistribution, err := s.mongoStore.FindBehaviorDistribution(profileID, &loc, consts.NEARBY_DISTANCE_RANGE, currentPeriodStart, currentPeriodEnd)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		previousDistribution, err := s.mongoStore.FindBehaviorDistribution(profileID, &loc, consts.NEARBY_DISTANCE_RANGE, previousPeriodStart, previousPeriodEnd)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		results := gatherReportItems(currentDistribution, previousDistribution)
		items := getReportItemsForDisplay(results, func(behaviorID string) string {
			name, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: fmt.Sprintf("behaviors.%s.name", behaviorID)})
			if err == nil {
				return name
			}

			behaviors, _ := s.mongoStore.FindBehaviorsByIDs([]string{behaviorID})
			if len(behaviors) == 1 {
				return behaviors[0].Name
			}
			return ""
		})
		c.JSON(http.StatusOK, gin.H{"report_items": items})
	case reportItemTypeCase:
		currActiveCount, _, _, err := s.mongoStore.GetCDSActive(loc, currentPeriodEnd)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		prevActiveCount, _, _, err := s.mongoStore.GetCDSActive(loc, previousPeriodEnd)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		currentDistribution := map[string]int{"covid_19": int(currActiveCount)}
		previousDistribution := map[string]int{"covid_19": int(prevActiveCount)}
		results := gatherReportItems(currentDistribution, previousDistribution)
		items := getReportItemsForDisplay(results, func(conditionID string) string {
			name, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: fmt.Sprintf("conditions.%s.name", conditionID)})
			if err == nil {
				return name
			}
			return ""
		})
		c.JSON(http.StatusOK, gin.H{"report_items": items})
	default:
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters)
		return
	}
}

// gatherReportItems returns a map or report item ID to the struct reportItem.
// It identifies report items once appearing in either currentDistribution or previousDistribution
// and determines respective count and change rate for each item.
func gatherReportItems(currentDistribution, previousDistribution map[string]int) map[string]*reportItem {
	items := make(map[string]*reportItem)
	for itemID, value := range currentDistribution {
		// For each reported item shown in this period, assume it's not reported in the previous period,
		// So the change rate is 100 by default.
		// If it's also reported in the previous period, the rate will be adjusted accordingly.
		items[itemID] = &reportItem{Value: value, ChangeRate: 100}
	}
	for itemID, value := range previousDistribution {
		if _, ok := items[itemID]; ok { // reported both in the current and previous periods
			items[itemID].ChangeRate = score.ChangeRate(float64(items[itemID].Value), float64(value))
		} else { // only reported in the previous period
			items[itemID] = &reportItem{Value: 0, ChangeRate: -100}
		}
	}
	return items
}

// getReportItemsForDisplay returns the final results to be shown.
// It also fills in the name of each report item, which is determined by getNameFunc.
func getReportItemsForDisplay(entries map[string]*reportItem, getNameFunc func(string) string) []*reportItem {
	results := make([]*reportItem, 0)
	for entryID, entry := range entries {
		entry.Name = getNameFunc(entryID)
		results = append(results, entry)
	}
	return results
}
