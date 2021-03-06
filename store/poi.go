package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/geo"
	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/utils"
)

var (
	ErrPOINotFound          = fmt.Errorf("poi not found")
	ErrPOIResourcesNotFound = fmt.Errorf("poi resources not found")
	ErrResolvePOIResource   = fmt.Errorf("poi resources can not resolved")
	ErrPOIListNotFound      = fmt.Errorf("poi list not found")
	ErrPOIListMismatch      = fmt.Errorf("poi list mismatch")
	ErrProfileNotUpdate     = fmt.Errorf("poi not update")
	ErrEmptyPOIResourceName = fmt.Errorf("empty poi resource name")
)

// DefaultResourceCount is the total number of list in the translation list
const DefaultResourceCount = 126

// importantResourceID marks the resources that are important so they could by highlight by API
var importantResourceID = map[string]struct{}{
	"resource_1": {}, "resource_3": {}, "resource_4": {}, "resource_5": {}, "resource_6": {},
	"resource_7": {}, "resource_8": {}, "resource_9": {}, "resource_10": {}, "resource_25": {},
	"resource_28": {}, "resource_36": {}, "resource_37": {}, "resource_45": {}, "resource_57": {},
	"resource_61": {}, "resource_63": {}, "resource_68": {}, "resource_71": {}, "resource_76": {},
	"resource_81": {}, "resource_86": {}, "resource_92": {}, "resource_95": {}, "resource_97": {},
	"resource_99": {}, "resource_101": {}, "resource_105": {}, "resource_107": {}, "resource_125": {},
}

var defaultResourceList = map[string][]schema.Resource{}
var defaultImportantResourceList = map[string][]schema.Resource{}
var defaultResourceIDMap = map[string]map[string]string{}

// LoadDefaultPOIResources loads resources from the tranlation list and cache it for later usage.
func LoadDefaultPOIResources(lang string) error {
	if lang == "" {
		lang = "en"
	}

	lang = strings.ReplaceAll(strings.ToLower(lang), "-", "_")

	if _, ok := defaultResourceList[lang]; ok {
		return nil
	}

	localizer := utils.NewLocalizer(lang)
	resourceIDMap := map[string]string{}
	resources := make([]schema.Resource, DefaultResourceCount)
	importantResources := make([]schema.Resource, 0, len(importantResourceID))
	for i := 0; i < DefaultResourceCount; i++ {
		id := fmt.Sprintf("resource_%d", i+1)
		name, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: fmt.Sprintf("resources.%s.name", id)})
		if err != nil {
			log.WithError(err).Error("fail to load resource in proper language")
			return err
		}
		resources[i] = schema.Resource{
			ID:   id,
			Name: name,
		}

		// check if a resource is important
		if _, ok := importantResourceID[id]; ok {
			resources[i].Important = true
			importantResources = append(importantResources, resources[i])
		}

		resourceIDMap[id] = name
	}
	defaultResourceList[lang] = resources
	defaultImportantResourceList[lang] = importantResources
	defaultResourceIDMap[lang] = resourceIDMap
	return nil
}

// ResolveResourceNameByID returns the name of a given resource id by languages
func ResolveResourceNameByID(id, lang string) (string, error) {
	lang = strings.ReplaceAll(strings.ToLower(lang), "-", "_")

	if lang == "zh" {
		lang = "zh_tw"
	}

	m, ok := defaultResourceIDMap[lang]
	if !ok {
		m = defaultResourceIDMap["en"]
		if len(m) == 0 {
			return "", ErrResolvePOIResource
		}
	}

	return m[id], nil
}

// getResourceList returns a list of resource list by language.
func getResourceList(lang string, important bool) ([]schema.Resource, error) {
	lang = strings.ReplaceAll(strings.ToLower(lang), "-", "_")

	if lang == "zh" {
		lang = "zh_tw"
	}

	resourceList := defaultResourceList
	if important {
		resourceList = defaultImportantResourceList
	}

	list, ok := resourceList[lang]
	if !ok {
		list = resourceList["en"]
		if len(list) == 0 {
			return nil, ErrPOIResourcesNotFound
		}
	}

	return list, nil
}

type POI interface {
	AddPOI(alias, address, placeType string, lon, lat float64) (*schema.POI, error)
	ListPOI(accountNumber string) ([]schema.POIDetail, error)
	ListPOIByResource(resourceID string, coordinates schema.Location) ([]schema.POI, error)

	GetPOI(poiID primitive.ObjectID) (*schema.POI, error)
	GetPOIByCoordinates(schema.Location) (*schema.POI, error)
	GetPOIMetrics(poiID primitive.ObjectID) (*schema.Metric, error)
	UpdatePOIGeoInfo(poiID primitive.ObjectID, location schema.Location) error
	UpdatePOIMetric(poiID primitive.ObjectID, metric schema.Metric, autonomyScore, autonomyScoreDelta float64) error

	UpdatePOIAlias(accountNumber, alias string, poiID primitive.ObjectID) error
	UpdatePOIOrder(accountNumber string, poiOrder []string) error
	DeletePOI(accountNumber string, poiID primitive.ObjectID) error
	NearestPOI(distance int, cords schema.Location) ([]primitive.ObjectID, error)

	AddPOIResources(poiID primitive.ObjectID, resources []schema.Resource, lang string) ([]schema.Resource, error)
	GetPOIResources(poiID primitive.ObjectID, importantOnly, includeAdded bool, lang string) ([]schema.Resource, error)
	GetPOIResourceMetric(poiID primitive.ObjectID) (schema.POIRatingsMetric, error)
	UpdatePOIRatingMetric(accountNumber string, poiID primitive.ObjectID, ratings []schema.RatingResource) error
}

// AddPOI inserts a new POI record if it doesn't exist and append it to user's profile
func (m *mongoDB) AddPOI(alias, address, placeType string, lon, lat float64) (*schema.POI, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.POICollection)

	var poi schema.POI
	query := bson.M{
		"location.coordinates.0": lon,
		"location.coordinates.1": lat,
	}

	if err := c.FindOne(ctx, query).Decode(&poi); err != nil {
		if err == mongo.ErrNoDocuments {
			location, err := geo.PoliticalGeoInfo(schema.Location{
				Latitude:  lat,
				Longitude: lon,
			})
			if err != nil {
				return nil, err
			}

			poi = schema.POI{
				Location: &schema.GeoJSON{
					Type:        "Point",
					Coordinates: []float64{lon, lat},
				},
			}

			result, err := c.InsertOne(ctx, bson.M{
				"location":   poi.Location,
				"country":    location.Country,
				"state":      location.State,
				"county":     location.County,
				"address":    address,
				"alias":      alias,
				"place_type": placeType,
			})

			if err != nil {
				return nil, err
			}
			poi.ID = result.InsertedID.(primitive.ObjectID)
			poi.Address = address
			poi.Alias = alias
			poi.Country = location.Country
			poi.State = location.State
			poi.County = location.County
			poi.PlaceType = placeType
		} else {
			return nil, err
		}
	}

	r, err := c.UpdateOne(ctx, bson.M{
		"_id": poi.ID,
		"$or": bson.A{bson.M{"alias": ""}, bson.M{"address": ""}}},
		bson.M{"$set": bson.M{"alias": alias, "address": address}},
	)
	if err != nil {
		return nil, err
	}

	if r.ModifiedCount == 1 {
		poi.Alias = alias
		poi.Address = address
	}

	if time.Since(time.Unix(poi.Metric.LastUpdate, 0)) > metricUpdateInterval {
		newMetric, err := m.SyncPOIMetrics(poi.ID, poi.ResourceRatings.Resources, schema.Location{
			Latitude:  lat,
			Longitude: lon,
			AddressComponent: schema.AddressComponent{
				Country: poi.Country,
				State:   poi.State,
				County:  poi.County,
			},
		})
		if err == nil {
			poi.Metric = *newMetric
		} else {
			log.WithError(err).Error("fail to sync poi metrics")

		}
	}

	return &poi, nil
}

// ListPOI finds the POI list of an account along with customied alias and address
func (m *mongoDB) ListPOI(accountNumber string) ([]schema.POIDetail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.ProfileCollection)

	// find user's POI list
	var result struct {
		Points []schema.POIDetail `bson:"points_of_interest"`
	}

	if err := c.FindOne(ctx,
		bson.M{
			"account_number":               accountNumber,
			"points_of_interest.monitored": true,
		},
		options.FindOne().SetProjection(
			bson.M{
				"points_of_interest": 1,
			}),
	).Decode(&result); err != nil {
		if err == mongo.ErrNoDocuments {
			return []schema.POIDetail{}, nil
		}
		return nil, err
	}

	monitoredPoints := make([]schema.POIDetail, 0, len(result.Points))
	for _, p := range result.Points {
		if p.Monitored {
			monitoredPoints = append(monitoredPoints, p)
		}
	}

	// find scores
	poiIDs := make([]primitive.ObjectID, 0)
	for _, p := range monitoredPoints {
		poiIDs = append(poiIDs, p.ID)
	}

	// $in query doesn't guarantee order
	// use aggregation to sort the nested docs according to the query order
	pipeline := []bson.M{
		{"$match": bson.M{"_id": bson.M{"$in": poiIDs}}},
		{"$addFields": bson.M{"__order": bson.M{"$indexOfArray": bson.A{poiIDs, "$_id"}}}},
		{"$sort": bson.M{"__order": 1}},
	}
	c = m.client.Database(m.database).Collection(schema.POICollection)
	cursor, err := c.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var pois []schema.POI
	if err = cursor.All(ctx, &pois); err != nil {
		return nil, err
	}

	if len(pois) != len(monitoredPoints) {
		log.WithFields(log.Fields{
			"pois":     pois,
			"poi_desc": monitoredPoints,
		}).Error("poi data wrongly retrieved or removed")
		return nil, fmt.Errorf("poi data wrongly retrieved or removed")
	}
	for i := range monitoredPoints {
		monitoredPoints[i].Score = pois[i].Score
		if l := pois[i].Location; l != nil {
			monitoredPoints[i].Location = &schema.Location{
				Longitude: l.Coordinates[0],
				Latitude:  l.Coordinates[1],
			}
		}
	}

	return monitoredPoints, nil
}

// ListPOIByResource returns all POI that satisfied following conditions:
// - the place has rated for a given resource.
// - the distance between the place and a given coordinates is within 50000m
func (m *mongoDB) ListPOIByResource(resourceID string, coordinates schema.Location) ([]schema.POI, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.POICollection)

	log.WithField("location", coordinates).WithField("resource", resourceID).Info("list POI by resource and location ")
	cursor, err := c.Aggregate(ctx, mongo.Pipeline{
		AggregationGeoNear(coordinates, 50000, GeoNearOption{
			DistanceKey:        "distance",
			DistanceMultiplier: 0.001,
		}),
		AggregationUnwind("$resource_ratings.resources"),
		AggregationMatch(bson.M{
			"resource_ratings.resources.resource.id": resourceID,
			"resource_ratings.resources.ratings":     bson.M{"$gt": 0},
		}),
		AggregationAddFields(bson.M{
			"resource_score": "$resource_ratings.resources.score",
		}),
		AggregationProject(bson.M{
			"resource_ratings": 0,
			"metric":           0,
		}),
	})
	if err != nil {
		return nil, err
	}

	var pois []schema.POI
	if err := cursor.All(ctx, &pois); err != nil {
		return nil, err
	}

	return pois, nil
}

// GetPOI finds POI by poi ID
func (m *mongoDB) GetPOI(poiID primitive.ObjectID) (*schema.POI, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.POICollection)

	// find user's POI
	var poi schema.POI
	query := bson.M{"_id": poiID}
	if err := c.FindOne(ctx, query).Decode(&poi); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrPOINotFound
		}
		return nil, err
	}

	// fetch poi geo info if it is not existent
	if poi.Country == "" {
		log.Info("fetch poi geo info from external service")
		location := schema.Location{
			Latitude:  poi.Location.Coordinates[1],
			Longitude: poi.Location.Coordinates[0],
		}
		location, err := geo.PoliticalGeoInfo(location)
		if err != nil {
			log.WithError(err).Error("can not fetch geo info")
			return nil, err
		}

		if err := m.UpdatePOIGeoInfo(poiID, location); err != nil {
			log.WithError(err).Error("can not update poi geo info")
			return nil, err
		}

		poi.Country = location.Country
		poi.County = location.County
		poi.State = location.State
	}

	return &poi, nil
}

// GetPOIByCoordinates searches POI by coordinates. There will be only one POI matches since
// we have add an index to make sure the coordinates of each POI is unique.
func (m *mongoDB) GetPOIByCoordinates(coordinates schema.Location) (*schema.POI, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.POICollection)

	// find user's POI
	var poi schema.POI
	query := bson.M{"location.coordinates": bson.A{coordinates.Longitude, coordinates.Latitude}}
	if err := c.FindOne(ctx, query).Decode(&poi); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrPOINotFound
		}
		return nil, err
	}

	// fetch poi geo info if it is not existent
	if poi.Country == "" {
		log.Info("fetch poi geo info from external service")
		location := schema.Location{
			Latitude:  poi.Location.Coordinates[1],
			Longitude: poi.Location.Coordinates[0],
		}
		location, err := geo.PoliticalGeoInfo(location)
		if err != nil {
			log.WithError(err).Error("can not fetch geo info")
			return nil, err
		}

		if err := m.UpdatePOIGeoInfo(poi.ID, location); err != nil {
			log.WithError(err).Error("can not update poi geo info")
			return nil, err
		}

		poi.Country = location.Country
		poi.County = location.County
		poi.State = location.State
	}

	return &poi, nil
}

// GetPOIMetrics finds POI by poi ID
func (m *mongoDB) GetPOIMetrics(poiID primitive.ObjectID) (*schema.Metric, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.POICollection)

	// find user's POI
	var poi schema.POI
	query := bson.M{"_id": poiID}
	if err := c.FindOne(ctx, query, options.FindOne().SetProjection(bson.M{
		"metric": 1,
	})).Decode(&poi); err != nil {
		return nil, err
	}

	return &poi.Metric, nil
}

// UpdatePOIAlias updates the alias of a POI for the specified account
func (m *mongoDB) UpdatePOIAlias(accountNumber, alias string, poiID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	query := bson.M{
		"account_number":        accountNumber,
		"points_of_interest.id": poiID,
	}
	update := bson.M{"$set": bson.M{"points_of_interest.$.alias": alias}}
	result, err := c.UpdateOne(ctx, query, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrPOINotFound
	}

	return nil
}

// UpdatePOIOrder updates the order of the POIs for the specified account
func (m *mongoDB) UpdatePOIOrder(accountNumber string, poiOrder []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.ProfileCollection)

	// construct mongodb aggregation $switch branches
	poiCondition := bson.A{}
	for i, id := range poiOrder {
		poiID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return err
		}

		poiCondition = append(poiCondition,
			bson.M{"case": bson.M{"$eq": bson.A{"$points_of_interest.id", poiID}}, "then": i})
	}

	cur, err := c.Aggregate(ctx, mongo.Pipeline{
		bson.D{
			{"$match", bson.D{
				{"account_number", accountNumber},
			}},
		},

		bson.D{
			{"$unwind", "$points_of_interest"},
		},

		bson.D{
			{"$addFields", bson.D{
				{"points_of_interest.order", bson.D{
					{"$switch", bson.D{
						{"branches", poiCondition},
						{"default", 1000},
					}},
				}},
			}},
		},

		bson.D{
			{"$sort", bson.M{
				"points_of_interest.order": 1}},
		},

		bson.D{
			{"$group", bson.D{
				{"_id", "$_id"},
				{"points_of_interest", bson.D{{"$push", "$points_of_interest"}}},
			}},
		},
	})

	if err != nil {
		switch e := err.(type) {
		case mongo.CommandError:
			if e.Code == 40066 { // $switch has no default and an input matched no case
				return ErrPOIListMismatch
			}
		default:
			return err
		}

	}

	var profiles []bson.M

	if err := cur.All(ctx, &profiles); nil != err {
		return err
	}

	if len(profiles) < 1 {
		return ErrPOIListNotFound
	}

	poi, ok := profiles[0]["points_of_interest"]
	if !ok {
		return ErrPOIListNotFound
	}

	query := bson.M{
		"account_number": accountNumber,
	}
	update := bson.M{"$set": bson.M{"points_of_interest": poi}}
	result, err := c.UpdateOne(ctx, query, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return ErrProfileNotUpdate
	}

	return nil
}

func (m *mongoDB) DeletePOI(accountNumber string, poiID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	query := bson.M{
		"account_number":        accountNumber,
		"points_of_interest.id": poiID,
	}
	update := bson.M{"$set": bson.M{
		"points_of_interest.$.monitored": false,
	}}
	if _, err := c.UpdateOne(ctx, query, update); err != nil {
		return err
	}

	return nil
}

func (m *mongoDB) UpdatePOIGeoInfo(poiID primitive.ObjectID, location schema.Location) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.POICollection)
	query := bson.M{
		"_id": poiID,
	}

	update := bson.M{
		"$set": bson.M{
			"country": location.Country,
			"state":   location.State,
			"county":  location.County,
		},
	}

	result, err := c.UpdateOne(ctx, query, update)
	pid := poiID.String()
	if err != nil {
		log.WithFields(log.Fields{
			"prefix": mongoLogPrefix,
			"poi ID": pid,
			"error":  err,
		}).Error("update poi location")
		return err
	}

	if result.MatchedCount == 0 {
		log.WithFields(log.Fields{
			"prefix": mongoLogPrefix,
			"poi ID": pid,
			"error":  ErrPOINotFound.Error(),
		}).Error("update poi metric")
		return ErrPOINotFound
	}

	return nil
}

func (m *mongoDB) UpdatePOIMetric(poiID primitive.ObjectID, metric schema.Metric, autonomyScore, autonomyScoreDelta float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.POICollection)
	query := bson.M{
		"_id": poiID,
	}

	metric.LastUpdate = time.Now().Unix()
	update := bson.M{
		"$set": bson.M{
			"autonomy_score":       autonomyScore,
			"autonomy_score_delta": autonomyScoreDelta,
			"metric":               metric,
		},
	}

	result, err := c.UpdateOne(ctx, query, update)
	pid := poiID.String()
	if err != nil {
		log.WithFields(log.Fields{
			"prefix": mongoLogPrefix,
			"poi ID": pid,
			"error":  err,
		}).Error("update poi metric")
		return err
	}

	if result.MatchedCount == 0 {
		log.WithFields(log.Fields{
			"prefix": mongoLogPrefix,
			"poi ID": pid,
			"error":  ErrPOINotFound.Error(),
		}).Error("update poi metric")
		return ErrPOINotFound
	}

	return nil
}

func (m *mongoDB) NearestPOI(distance int, cords schema.Location) ([]primitive.ObjectID, error) {
	query := distanceQuery(distance, cords)
	c := m.client.Database(m.database).Collection(schema.POICollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	cur, err := c.Find(ctx, query)
	if nil != err {
		log.WithField("prefix", mongoLogPrefix).Errorf("query poi nearest distance with error: %s", err.Error())
		return nil, fmt.Errorf("poi nearest distance query with error: %s", err.Error())
	}

	var POIs []primitive.ObjectID

	// iterate
	for cur.Next(ctx) {
		var poi schema.POI
		if err := cur.Decode(&poi); nil != err {
			log.WithField("prefix", mongoLogPrefix).Infof("query nearest distance with error: %s", err.Error())
			return nil, fmt.Errorf("nearest distance query decode record with error: %s", err.Error())
		}
		POIs = append(POIs, poi.ID)
	}

	log.WithField("prefix", mongoLogPrefix).Debugf("poi nearest distance query gets %d records near long:%v lat:%v", len(POIs),
		cords.Longitude, cords.Latitude)

	return POIs, nil
}

// AddPOIResources add resources into a POI. If a resource ID is given, it will resolve it name by language. On the
// other hand, if a name is given it will generate an ID by hashing. The `ratings` of `POIResourceRating` is default to 0.
func (m *mongoDB) AddPOIResources(poiID primitive.ObjectID, resources []schema.Resource, lang string) ([]schema.Resource, error) {
	if 0 == len(resources) {
		return nil, errors.New("empty resource")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db := m.client.Database(m.database)

	poi := schema.POI{}
	if err := db.Collection(schema.POICollection).FindOne(ctx, bson.M{"_id": poiID}).Decode(&poi); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrPOINotFound
		}
		return nil, err
	}

	for i := range resources {
		resource := &resources[i]
		if resource.ID != "" {
			var err error
			resource.Name, err = ResolveResourceNameByID(resource.ID, lang)
			if err != nil {
				return nil, err
			}
		} else if resource.Name != "" {
			h := sha256.New()
			h.Write([]byte(fmt.Sprintf("%s=:=", strings.ToLower(resource.Name))))
			resource.ID = hex.EncodeToString(h.Sum(nil))
		} else {
			return nil, fmt.Errorf("the id and name of resource should not be both empty (what is this?)")
		}

		r := schema.POIResourceRating{
			Resource: *resource,
		}

		query := bson.M{"_id": poiID, "resource_ratings.resources": bson.M{"$not": bson.M{"$elemMatch": bson.M{"resource.id": resource.ID}}}}
		update := bson.M{"$push": bson.M{"resource_ratings.resources": r}}

		_, err := db.Collection(schema.POICollection).UpdateOne(ctx, query, update)
		if err != nil {
			return nil, err
		}
	}

	return resources, nil
}

// GetPOIResources get resources suggestion for a POI. The list now comes from the translation file.
// If there is any changes over the list, we need to update `DefaultResourceCount` and `importantResourceID` accordingly.
// The importantOnly variable determines whether only shows resources where it is importants.
// Normally, added resources will be excluded from the return. If `includeAdded` is set to true,
// the return will contain those added resources.
func (m *mongoDB) GetPOIResources(poiID primitive.ObjectID, importantOnly, includeAdded bool, lang string) ([]schema.Resource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := m.client.Database(m.database)

	query := bson.M{
		"_id": poiID,
	}

	var poi schema.POI
	if err := c.Collection(schema.POICollection).
		FindOne(ctx, query, options.FindOne().SetProjection(bson.M{"resource_ratings.resources": 1})).
		Decode(&poi); err != nil {
		return nil, err
	}

	ratedResource := map[string]struct{}{}
	for _, r := range poi.ResourceRatings.Resources {
		ratedResource[r.ID] = struct{}{}
	}

	resources, err := getResourceList(lang, false)
	if err != nil {
		return nil, err
	}

	suggestedResource := make([]schema.Resource, 0)
	for _, r := range resources {
		_, added := ratedResource[r.ID]
		if includeAdded && added {
			suggestedResource = append(suggestedResource, r)
		}

		if importantOnly && !r.Important {
			continue
		}

		if added {
			continue
		}

		suggestedResource = append(suggestedResource, r)
	}

	return suggestedResource, nil
}
