package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/score"
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
	Days        int                               `form:"days"`
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
	var params reportItemQueryParams
	if err := c.Bind(&params); err != nil {
		abortWithEncoding(c, http.StatusBadRequest, errorInvalidParameters, err)
		return
	}

	macaroonToken := c.GetHeader("X-FORWARD-MACAROON-CDS")
	if macaroonToken == "" {
		abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, fmt.Errorf("invalid maracoon token"))
		return
	}

	switch params.Type {
	case reportItemTypeSymptom:
		items, err := s.dataStore.GetCommunitySymptomReportItems(macaroonToken, params.Days)
		if err != nil {
			abortWithEncoding(c, http.StatusInternalServerError, errorInternalServer, err)
			return
		}

		c.JSON(http.StatusOK, items)
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
		if *results[i].Value > *results[j].Value {
			return true
		}
		if *results[i].Value < *results[j].Value {
			return false
		}
		return results[i].Name < results[j].Name
	})
	return results
}
