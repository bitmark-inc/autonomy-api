package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

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
	Scope       string                            `form:"scope"`
	Type        string                            `form:"type"`
	Granularity schema.AggregationTimeGranularity `form:"granularity"`
	Start       string                            `form:"start"`
	End         string                            `form:"end"`
	Language    string                            `form:"lang"`
	PoiID       string                            `form:"poi_id"`
}

type reportItem struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Value        *int           `json:"value"`
	ChangeRate   *float64       `json:"change_rate"`
	Distribution map[string]int `json:"distribution"`
}

func (r *reportItem) MarshalJSON() ([]byte, error) {
	distribution := map[string]int{}
	if r.Distribution != nil {
		distribution = r.Distribution
	}
	return json.Marshal(&struct {
		ID           string         `json:"id"`
		Name         string         `json:"name"`
		Value        *int           `json:"value"`
		ChangeRate   *float64       `json:"change_rate"`
		Distribution map[string]int `json:"distribution"`
	}{
		ID:           r.ID,
		Name:         r.Name,
		Value:        r.Value,
		ChangeRate:   r.ChangeRate,
		Distribution: distribution,
	})
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
	start, err := time.Parse(time.RFC3339, params.Start)
	if err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}
	end, err := time.Parse(time.RFC3339, params.End)
	if err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}
	if params.Granularity != schema.AggregationByMonth && params.Granularity != schema.AggregationByDay {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters)
		return
	}

	currentPeriodStart := start.UTC().Unix()
	currentPeriodEnd := end.UTC().Unix()
	previousPeriodStart := 2*currentPeriodStart - currentPeriodEnd
	previousPeriodEnd := currentPeriodStart
	localizer := utils.NewLocalizer(params.Language)

	utcOffset := "+0000"
	if start.Location() != nil {
		utcOffset = time.Now().In(start.Location()).Format("-0700")
	}

	var profileID string
	var loc schema.Location
	var scoreOwner string
	switch params.Scope {
	case reportItemScopeIndividual:
		profileID = account.ProfileID.String()
		scoreOwner = account.AccountNumber

		if params.Type == reportItemTypeSymptom {
			currentBuckets, err := s.mongoStore.GetPersonalSymptomTimeSeriesData(profileID, currentPeriodStart, currentPeriodEnd, utcOffset, params.Granularity)
			if err != nil {
				abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
				return
			}
			previousDistribution, err := s.mongoStore.FindSymptomDistribution(profileID, &loc, consts.NEARBY_DISTANCE_RANGE, previousPeriodStart, previousPeriodEnd, false)
			if err != nil {
				abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
				return
			}
			results := gatherReportItemsWithDistribution(currentBuckets, previousDistribution, false)
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
			return
		} else if params.Type == reportItemTypeBehavior {
			currentBuckets, err := s.mongoStore.GetPersonalBehaviorTimeSeriesData(profileID, currentPeriodStart, currentPeriodEnd, utcOffset, params.Granularity)
			if err != nil {
				abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
				return
			}
			previousDistribution, err := s.mongoStore.FindBehaviorDistribution(profileID, &loc, consts.NEARBY_DISTANCE_RANGE, previousPeriodStart, previousPeriodEnd)
			if err != nil {
				abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
				return
			}
			results := gatherReportItemsWithDistribution(currentBuckets, previousDistribution, false)
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
			return
		}
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
		currData, err := s.mongoStore.GetScoreTimeSeriesData(scoreOwner, currentPeriodStart, currentPeriodEnd, params.Granularity)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}
		prevAvgScore, err := s.mongoStore.GetScoreAverage(scoreOwner, previousPeriodStart, previousPeriodEnd)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}

		results := gatherReportItemsWithDistribution(
			map[string][]schema.Bucket{"autonomy score": currData},
			map[string]int{"autonomy score": int(prevAvgScore)},
			true)
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
		if _, ok := schema.CDSCountyCollectionMatrix[schema.CDSCountryType(loc.Country)]; !ok {
			name, _ := localizer.Localize(&i18n.LocalizeConfig{MessageID: fmt.Sprintf("conditions.%s.name", "covid_19")})
			c.JSON(http.StatusOK, gin.H{"report_items": []*reportItem{{Name: name}}})
			return
		}
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
		v := value
		changeRate := 100.0
		items[itemID] = &reportItem{ID: itemID, Value: &v, ChangeRate: &changeRate}
	}
	for itemID, value := range previousDistribution {
		if _, ok := items[itemID]; ok { // reported both in the current and previous periods
			changeRate := score.ChangeRate(float64(*items[itemID].Value), float64(value))
			items[itemID].ChangeRate = &changeRate
		} else { // only reported in the previous period
			v := 0
			changeRate := -100.0
			items[itemID] = &reportItem{ID: itemID, Value: &v, ChangeRate: &changeRate}
		}
	}
	return items
}

// gatherReportItemsByAggregation is similar to the above function `gatherReportItems`.
// It provides distribution and the aggregated value of the current time period.
func gatherReportItemsWithDistribution(currentBuckets map[string][]schema.Bucket, previousDistribution map[string]int, avg bool) map[string]*reportItem {
	items := make(map[string]*reportItem)
	for itemID, buckets := range currentBuckets {
		if len(buckets) == 0 {
			continue
		}
		// For each reported item shown in this period, assume it's not reported in the previous period,
		// So the change rate is 100 by default.
		// If it's also reported in the previous period, the rate will be adjusted accordingly.
		sum := 0
		distribution := make(map[string]int)
		for _, b := range buckets {
			sum += b.Value
			distribution[b.Name] = b.Value
		}
		value := sum
		if avg {
			value = sum / len(distribution)
		}
		changeRate := 100.0
		items[itemID] = &reportItem{ID: itemID, Value: &value, ChangeRate: &changeRate, Distribution: distribution}
	}
	for itemID, value := range previousDistribution {
		if _, ok := items[itemID]; ok { // reported both in the current and previous periods
			changeRate := score.ChangeRate(float64(*items[itemID].Value), float64(value))
			items[itemID].ChangeRate = &changeRate
		} else { // only reported in the previous period
			v := 0
			changeRate := -100.0
			items[itemID] = &reportItem{ID: itemID, Value: &v, ChangeRate: &changeRate}
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
	sort.SliceStable(results, func(i, j int) bool {
		return *results[i].Value > *results[j].Value
	})
	return results
}
