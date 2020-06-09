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
	Count      int     `json:"count"`
	ChangeRate float64 `json:"change_rate"`
}

func (s *Server) getReportItems(c *gin.Context) {
	var params reportItemQueryParams
	if err := c.Bind(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	c.Set("reportType", params.Type)
	c.Set("currentPeriodStart", params.Start)
	c.Set("currentPeriodEnd", params.End)
	c.Set("previousPeriodStart", 2*params.Start-params.End)
	c.Set("previousPeriodEnd", params.Start)
	c.Set("language", params.Language)

	switch params.Scope {
	case reportItemScopeIndividual:
		// TODO: handle individual
	case reportItemScopeNeighborhood:
		a := c.MustGet("account")
		account, ok := a.(*schema.Account)
		if !ok {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer)
			return
		}
		loc := account.Profile.State.LastLocation
		if loc == nil {
			abortWithEncoding(c, http.StatusBadRequest, errorUnknownAccountLocation)
			return
		}
		if loc.Country == "" {
			log.Info("fetch poi geo info from external service")
			resolvedLococation, err := geo.PoliticalGeoInfo(*loc)
			if err != nil {
				log.WithError(err).WithField("location", loc).Error("failed to fetch geo info")
			}
			c.Set("location", resolvedLococation)
		} else {
			c.Set("location", *loc)
		}
		s.getLocationBasedReportItems(c)
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
		location := schema.Location{
			Latitude:  poi.Location.Coordinates[1],
			Longitude: poi.Location.Coordinates[0],
			AddressComponent: schema.AddressComponent{
				Country: poi.Country,
				State:   poi.State,
				County:  poi.County,
			},
		}
		c.Set("location", location)
		s.getLocationBasedReportItems(c)
	}
}

func (s *Server) getLocationBasedReportItems(c *gin.Context) {
	loc := c.Keys["location"].(schema.Location)
	reportType := c.GetString("reportType")
	currStart := c.GetInt64("currentPeriodStart")
	currEnd := c.GetInt64("currentPeriodEnd")
	prevStart := c.GetInt64("previousPeriodStart")
	prevEnd := c.GetInt64("previousPeriodEnd")
	lang := c.GetString("language")

	localizer := utils.NewLocalizer(lang)

	switch reportType {
	case reportItemTypeScore:
		// TODO: handle score
	case reportItemTypeSymptom:
		currentDistribution, err := s.mongoStore.FindNearbySymptomDistribution(consts.NEARBY_DISTANCE_RANGE, loc, currStart, currEnd)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		previousDistribution, err := s.mongoStore.FindNearbySymptomDistribution(consts.NEARBY_DISTANCE_RANGE, loc, prevStart, prevEnd)
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
		currentDistribution, err := s.mongoStore.FindNearbyBehaviorDistribution(consts.NEARBY_DISTANCE_RANGE, loc, currStart, currEnd)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		previousDistribution, err := s.mongoStore.FindNearbyBehaviorDistribution(consts.NEARBY_DISTANCE_RANGE, loc, prevStart, prevEnd)
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
		currActiveCount, _, _, err := s.mongoStore.GetCDSActive(loc, currEnd)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		prevActiveCount, _, _, err := s.mongoStore.GetCDSActive(loc, prevEnd)
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
	for itemID, count := range currentDistribution {
		// For each reported item shown in this period, assume it's not reported in the previous period,
		// So the change rate is 100 by default.
		// If it's also reported in the previous period, the rate will be adjusted accordingly.
		items[itemID] = &reportItem{Count: count, ChangeRate: 100}
	}
	for itemID, count := range previousDistribution {
		if _, ok := items[itemID]; ok { // reported both in the current and previous periods
			items[itemID].ChangeRate = score.ChangeRate(float64(items[itemID].Count), float64(count))
		} else { // only reported in the previous period
			items[itemID] = &reportItem{Count: 0, ChangeRate: -100}
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
