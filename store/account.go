package store

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/utils"
)

// MongoAccount - account related operations
// mongo version create account is different from postgres
type MongoAccount interface {
	CreateAccount(*schema.Account) error
	CreateAccountWithGeoPosition(*schema.Account, float64, float64) error
	UpdateAccountGeoPosition(string, float64, float64) error

	DeleteAccount(string) error
	UpdateAccountScore(string, float64) error
	UpdateAccountNudge(string, schema.NudgeType) error
	IsAccountExist(string) (bool, error)
	GetProfile(accountNumber string) (*schema.Profile, error)
	GetProfilesByPOI(id string) ([]schema.Profile, error)
	AppendPOIToAccountProfile(accountNumber string, desc schema.ProfilePOI) error

	UpdateProfileIndividualMetric(profileID string, metric schema.IndividualMetric) error
	UpdateProfileMetric(accountNumber string, metric schema.Metric) error
	UpdateProfileTimezone(accountNumber string, timezone string) error
	UpdateProfilePOIMetric(accountNumber string, poiID primitive.ObjectID, metric schema.Metric) error
	GetProfileCoefficient(accountNumber string) (*schema.ScoreCoefficient, error)
	UpdateProfileCoefficient(accountNumber string, coefficient schema.ScoreCoefficient) error
	ResetProfileCoefficient(accountNumber string) error
	ProfileMetric(accountNumber string) (*schema.Metric, error)
	UpdateAreaProfileBehavior(behaviors []schema.Behavior, location schema.Location) error
	UpdateAreaProfileSymptom(symptoms []schema.Symptom, location schema.Location) error

	UpdateProfilePOIRatingMetric(accountNumber string, poiID primitive.ObjectID, metric schema.ProfileRatingsMetric) error
	GetProfilePOIRatingMetric(accountNumber string, id string) (schema.ProfileRatingsMetric, error)
}

var (
	errAccountNotFound = fmt.Errorf("account not found")
)

// CreateAccount is to register an account into autonomy system
func (s *AutonomyStore) CreateAccount(accountNumber, encPubKey string, metadata map[string]interface{}) (*schema.Account, error) {
	a := schema.Account{
		AccountNumber: accountNumber,
		EncPubKey:     encPubKey,
		Profile: schema.AccountProfile{
			AccountNumber: accountNumber,
			State: schema.ActivityState{
				LastActiveTime: time.Now(),
			},
			Metadata: schema.AccountMetadata(metadata),
		},
	}

	if err := s.ormDB.Create(&a).Error; err != nil {
		return nil, err
	}

	return &a, nil
}

// GetAccount returns an account instance of a given account number
func (s *AutonomyStore) GetAccount(accountNumber string) (*schema.Account, error) {
	var a schema.Account
	if err := s.ormDB.Preload("Profile").Where("account_number = ?", accountNumber).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// UpdateAccountMetadata is to update metadata for a specific account
func (s *AutonomyStore) UpdateAccountMetadata(accountNumber string, metadata map[string]interface{}) error {
	var a schema.Account
	if err := s.ormDB.Preload("Profile").Where("account_number = ?", accountNumber).First(&a).Error; err != nil {
		return err
	}

	for k, v := range metadata {
		a.Profile.Metadata[k] = v
		if k == "timezone" {
			if timezone, _ := v.(string); utils.GetLocation(timezone) != nil {
				if err := s.mongo.UpdateProfileTimezone(accountNumber, timezone); err != nil {
					return err
				}
			}
		}
	}

	return s.ormDB.Save(&a.Profile).Error
}

// UpdateAccountMetadata is to update metadata for a specific account
func (s *AutonomyStore) UpdateAccountGeoPosition(accountNumber string, latitude, longitude float64) error {
	var a schema.Account
	if err := s.ormDB.Preload("Profile").Where("account_number = ?", accountNumber).First(&a).Error; err != nil {
		return err
	}

	a.Profile.State.LastLocation = &schema.Location{
		Latitude:  latitude,
		Longitude: longitude,
	}

	err := s.ormDB.Save(&a.Profile).Error
	if nil != err {
		return err
	}

	// if mongo db has no record, create new account with geolocation data
	exist, err := s.mongo.IsAccountExist(a.AccountNumber)
	if nil != err {
		return err
	}

	if exist {
		return s.mongo.UpdateAccountGeoPosition(a.AccountNumber, latitude, longitude)
	}

	return s.mongo.CreateAccountWithGeoPosition(&a, latitude, longitude)
}

// DeleteAccount removes an account from our system permanently
func (s *AutonomyStore) DeleteAccount(accountNumber string) error {
	if err := s.ormDB.Delete(schema.Account{}, "account_number = ?", accountNumber).Error; err != nil {
		return err
	}

	if err := s.ormDB.Delete(schema.AccountProfile{}, "account_number = ?", accountNumber).Error; err != nil {
		return err
	}

	return nil
}

func (m *mongoDB) CreateAccount(a *schema.Account) error {
	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	p := schema.Profile{
		ID:            a.ProfileID.String(),
		AccountNumber: a.AccountNumber,
		HealthScore:   100,
	}

	log.WithField("prefix", mongoLogPrefix).Debug("account profile")

	result, err := c.InsertOne(ctx, p)
	if nil != err {
		log.WithField("prefix", mongoLogPrefix).Errorf("create mongo account error: %s", err)
		return err
	}

	log.WithField("prefix", mongoLogPrefix).Infof("create mongo account result: %v", result)
	return nil
}

func (m *mongoDB) CreateAccountWithGeoPosition(a *schema.Account, latitude, longitude float64) error {
	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	p := schema.Profile{
		ID:            a.ProfileID.String(),
		AccountNumber: a.AccountNumber,
		HealthScore:   100,
		Location: &schema.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{longitude, latitude},
		},
	}

	log.WithField("prefix", mongoLogPrefix).Debugf("create account profile: %v", p)

	result, err := c.InsertOne(ctx, p)
	if nil != err {
		log.WithField("prefix", mongoLogPrefix).Errorf("create mongo account error: %s", err)
		return err
	}

	log.WithField("prefix", mongoLogPrefix).Infof("create mongo account result: %v", result)
	return nil
}

func (m *mongoDB) UpdateAccountGeoPosition(accountNumber string, latitude, longitude float64) error {
	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	log.WithField("prefix", mongoLogPrefix).Debugf("update account %s to new geolocation: long %f, lat %f", accountNumber, longitude, latitude)

	query := bson.M{
		"account_number": bson.M{
			"$eq": accountNumber,
		},
	}

	update := bson.M{
		"$set": bson.M{
			"location": bson.M{
				"type": "Point",
				"coordinates": bson.A{
					longitude, latitude,
				},
			},
		},
	}

	result, err := c.UpdateMany(ctx, query, update)
	if nil != err {
		log.WithField("prefix", mongoLogPrefix).Errorf("update account %s geolocation with error: %s", accountNumber, err)
		return err
	}

	log.WithField("prefix", mongoLogPrefix).Debugf("update mongo account geolocation result: %v", result)

	return nil
}

func (m *mongoDB) DeleteAccount(accountNumber string) error {
	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	result, err := c.DeleteOne(ctx, bson.M{"account_number": accountNumber})
	if nil != err {
		log.WithField("prefix", mongoLogPrefix).Errorf("delete mongo account %s with error: %s", accountNumber, err)
		return err
	}

	log.WithField("prefix", mongoLogPrefix).Infof("delete account number %s success with result: %v", accountNumber, result)

	return nil
}

// IsAccountExist check if account number exist in mongo db
func (m *mongoDB) IsAccountExist(accountNumber string) (bool, error) {
	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	count, err := c.CountDocuments(ctx, bson.M{"account_number": accountNumber})
	if nil != err {
		log.WithField("prefix", mongoLogPrefix).Errorf("query account number %s with error: %v", accountNumber, err)

		return false, err
	}
	return count > 0, nil
}

// UpdateAccountScore update a score of an account
func (m *mongoDB) UpdateAccountScore(accountNumber string, score float64) error {
	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	log.WithField("prefix", mongoLogPrefix).Debugf("update account %s with new score %f", accountNumber, score)

	query := bson.M{
		"account_number": bson.M{
			"$eq": accountNumber,
		},
	}
	update := bson.M{"$set": bson.M{"health_score": score}}
	result, err := c.UpdateMany(ctx, query, update)
	if nil != err {
		log.WithField("prefix", mongoLogPrefix).Errorf("update score of account:%s  error: %s", accountNumber, err)
		return err
	}

	log.WithField("prefix", mongoLogPrefix).Debugf("update score of an account:%s result: %v", accountNumber, result)

	return nil
}

func (m *mongoDB) UpdateAccountNudge(accountNumber string, nudgeType schema.NudgeType) error {
	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	log.WithField("prefix", mongoLogPrefix).Debugf("update last symptom nudge time for account: %s ", accountNumber)

	query := bson.M{
		"account_number": bson.M{
			"$eq": accountNumber,
		},
	}

	update := bson.M{"$set": bson.M{fmt.Sprintf("last_nudge.%s", nudgeType): time.Now().UTC()}}
	_, err := c.UpdateOne(ctx, query, update)
	if nil != err {
		log.WithField("prefix", mongoLogPrefix).WithError(err).WithField("account_number", accountNumber).
			Errorf("update last symptom update time of")
		return err
	}

	return nil
}

// AppendPOIToAccountProfile appends a POI to end of the POI list of an account
func (m *mongoDB) AppendPOIToAccountProfile(accountNumber string, poiDesc schema.ProfilePOI) error {
	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	prefixedLog := log.WithField("prefix", mongoLogPrefix).WithField("account", accountNumber).WithField("poi_id", poiDesc.ID)

	// don't do anything if the POI was added before
	query := bson.M{
		"account_number":        accountNumber,
		"points_of_interest.id": poiDesc.ID,
	}
	count, err := c.CountDocuments(ctx, query)
	if err != nil {
		prefixedLog.WithError(err).Error("unable to query POI for this account")
		return err
	}
	if count > 0 {
		return nil
	}

	// add the POI if not added before
	query = bson.M{"account_number": bson.M{"$eq": accountNumber}}
	update := bson.M{"$push": bson.M{"points_of_interest": poiDesc}}
	if r, err := c.UpdateOne(ctx, query, update); nil != err {
		prefixedLog.WithError(err).Error("unable to add POI for this account")
		return err
	} else {
		if r.ModifiedCount != 1 {
			return fmt.Errorf("fail to update poi into profile")
		}
	}

	return nil
}

// GetProfileCoefficient returns coefficient saved in mongodb
func (m *mongoDB) GetProfileCoefficient(accountNumber string) (*schema.ScoreCoefficient, error) {
	var profile schema.Profile

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	query := bson.M{
		"account_number": accountNumber,
	}
	if err := c.FindOne(ctx, query).Decode(&profile); err != nil {
		return nil, err
	}

	return profile.ScoreCoefficient, nil
}

// UpdateProfileCoefficient updates coefficient for an account into mongodb
func (m *mongoDB) UpdateProfileCoefficient(accountNumber string, coefficient schema.ScoreCoefficient) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	query := bson.M{
		"account_number": accountNumber,
	}
	update := bson.M{
		"$set": bson.M{
			"score_coefficient": coefficient,
		},
	}

	result, err := c.UpdateOne(ctx, query, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errAccountNotFound
	}

	return nil
}

// UpdateProfileTimezone updates timezone for an account into mongodb
func (m *mongoDB) UpdateProfileTimezone(accountNumber string, timezone string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	query := bson.M{
		"account_number": accountNumber,
	}
	update := bson.M{
		"$set": bson.M{
			"timezone": timezone,
		},
	}

	result, err := c.UpdateOne(ctx, query, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errAccountNotFound
	}

	return nil
}

// UpdateProfileCoefficient clean up coefficient for an account from mongodb
func (m *mongoDB) ResetProfileCoefficient(accountNumber string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	query := bson.M{
		"account_number": accountNumber,
	}
	update := bson.M{
		"$unset": bson.M{
			"score_coefficient": "",
		},
		"$set": bson.M{
			"metric.last_update": 0,
		},
	}

	_, err := c.UpdateOne(ctx, query, update)
	return err
}

// UpdateProfileIndividualMetric update individual metric into profile
func (m *mongoDB) UpdateProfileIndividualMetric(profileID string, metric schema.IndividualMetric) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	query := bson.M{
		"id": profileID,
	}
	update := bson.M{
		"$set": bson.M{
			"individual_metric": metric,
		},
	}

	result, err := c.UpdateOne(ctx, query, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errAccountNotFound
	}

	return nil
}

func (m *mongoDB) UpdateProfileMetric(accountNumber string, metric schema.Metric) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	metric.LastUpdate = time.Now().Unix()

	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	query := bson.M{
		"account_number": accountNumber,
	}
	update := bson.M{
		"$set": bson.M{
			"metric": metric,
		},
	}

	result, err := c.UpdateOne(ctx, query, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errAccountNotFound
	}

	return nil
}

func (m *mongoDB) UpdateProfilePOIMetric(accountNumber string, poiID primitive.ObjectID, metric schema.Metric) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	query := bson.M{
		"account_number":        accountNumber,
		"points_of_interest.id": poiID,
	}

	update := bson.M{
		"$set": bson.M{
			"points_of_interest.$.metric":     metric,
			"points_of_interest.$.score":      metric.Score,
			"points_of_interest.$.updated_at": time.Now().UTC(),
		},
	}

	result, err := c.UpdateOne(ctx, query, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errAccountNotFound
	}

	return nil
}

func (m *mongoDB) GetProfilesByPOI(id string) ([]schema.Profile, error) {
	log.WithField("prefix", mongoLogPrefix).Debugf("get accounts by POI id: %s", id)

	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	poiID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.WithField("prefix", mongoLogPrefix).Errorf("incorrect POI id:%s  error: %s", id, err.Error())
		return nil, err
	}

	query := bson.M{"points_of_interest.id": poiID}

	filter := bson.M{
		"account_number":       1,
		"score_coefficient":    1,
		"metric":               1,
		"points_of_interest.$": 1,
	}

	cursor, err := c.Find(ctx, query, options.Find().SetProjection(filter))

	profiles := []schema.Profile{}

	for cursor.Next(ctx) {
		// Declare a result BSON object
		var profile schema.Profile
		if err := cursor.Decode(&profile); err != nil {
			log.WithField("prefix", mongoLogPrefix).Errorf("fail to query accounts include POI id:%s  error: %s", id, err.Error())
			return nil, err
		}

		profiles = append(profiles, profile)
	}

	return profiles, nil
}

func (m *mongoDB) UpdateProfilePOIRatingMetric(accountNumber string, poiID primitive.ObjectID, metric schema.ProfileRatingsMetric) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	query := bson.M{
		"account_number":        accountNumber,
		"points_of_interest.id": poiID,
	}

	update := bson.M{
		"$set": bson.M{
			"points_of_interest.$.resource_ratings": metric,
			"points_of_interest.$.updated_at":       time.Now().UTC(),
		},
	}

	result, err := c.UpdateOne(ctx, query, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errAccountNotFound
	}

	return nil
}

func (m *mongoDB) GetProfilePOIRatingMetric(accountNumber string, id string) (schema.ProfileRatingsMetric, error) {
	c := m.client.Database(m.database).Collection(schema.ProfileCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	log.WithField("prefix", mongoLogPrefix).Debugf("get accounts:%s by POI id:%s", accountNumber, id)

	filter := bson.D{{"account_number", accountNumber}, {}}
	var result schema.Profile
	err := c.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		log.WithFields(log.Fields{
			"prefix":         mongoLogPrefix,
			"Account number": accountNumber,
			"error":          err,
		}).Error("get profile fail")
		return schema.ProfileRatingsMetric{}, err
	}
	for _, p := range result.PointsOfInterest {
		if p.ID.Hex() == id {
			return p.ResourceRatings, nil
		}
	}

	return schema.ProfileRatingsMetric{}, err
}
