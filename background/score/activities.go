package score

import (
	"context"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"github.com/vmihailenco/msgpack/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/score"
	"github.com/bitmark-inc/autonomy-api/utils"
)

var ErrInvalidLocation = fmt.Errorf("invalid location")
var ErrTooFrequentUpdate = fmt.Errorf("too frequent update")

// NotificationProfile is a struct that summarizes how notifications are going to deliver.
type NotificationProfile struct {
	StateChangedAccounts  []string
	SymptomsSpikeAccounts []string
	ReportRiskArea        bool
	RemindGoodBehavior    bool
}

// CalculatePOIStateActivity calculates metrics by the location of a POI
func (s *ScoreUpdateWorker) CalculatePOIStateActivity(ctx context.Context, id string) (*schema.Metric, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Query poi for calculating state.", zap.String("poiID", id))

	poiID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	poi, err := s.mongo.GetPOI(poiID)
	if err != nil {
		return nil, err
	}

	if poi == nil || poi.Location == nil {
		return nil, ErrInvalidLocation
	}

	if time.Since(time.Unix(poi.Metric.LastUpdate, 0)) < 5*time.Second {
		return nil, ErrTooFrequentUpdate
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

	logger.Info("Calculate POI metric by location.", zap.Any("location", location))
	rawMetrics, err := s.mongo.CollectRawMetrics(location)
	if err != nil {
		return nil, err
	}
	metric := score.CalculateMetric(*rawMetrics, nil)

	return &metric, nil
}

func (s *ScoreUpdateWorker) CheckLocationSpikeActivity(ctx context.Context, spikeSymptomTypes []string) ([]schema.Symptom, error) {
	if len(spikeSymptomTypes) > 0 {
		symptoms, err := s.mongo.FindSymptomsByIDs(spikeSymptomTypes)
		if err != nil {
			return nil, err
		}
		return symptoms, nil
	}

	return nil, nil
}

// RefreshLocationStateActivity updates the metrics as well as the score if the POI id
// is not provided. Otherwise, it updates the score of POIs in the profile.
// It will return accounts whose score's color is changed.
func (s *ScoreUpdateWorker) RefreshLocationStateActivity(ctx context.Context, accountNumber, poiID string, metric schema.Metric) (*NotificationProfile, error) {
	logger := activity.GetLogger(ctx)

	var reportRiskArea, remindGoodBehavior bool
	stateChangedAccounts := make([]string, 0)
	symptomsSpikeAccounts := make([]string, 0)

	if poiID != "" {
		id, err := primitive.ObjectIDFromHex(poiID)
		if err != nil {
			return nil, err
		}

		resourceMetric, err := s.mongo.GetPOIResourceMetric(id)
		if err != nil {
			return nil, err
		}

		autonomyScore, _, autonomyScoreDelta := score.CalculatePOIAutonomyScore(resourceMetric.Resources, metric)

		if err := s.mongo.UpdatePOIMetric(id, metric, autonomyScore, autonomyScoreDelta); err != nil {
			return nil, err
		}

		if err := s.mongo.AddScoreRecord(poiID, schema.ScoreRecordTypePOI, autonomyScore, time.Now().UTC().Unix()); err != nil {
			sentry.CaptureException(err)
		}

		profiles, err := s.mongo.GetProfilesByPOI(poiID)
		if err != nil {
			return nil, err
		}

		// Since metric is used by all profile, we make a deep copy of metric to
		// prevent it from mutating by calculation
		b, err := msgpack.Marshal(metric)
		if err != nil {
			return nil, err
		}

		for _, profile := range profiles {
			var metric schema.Metric
			if err := msgpack.Unmarshal(b, &metric); err != nil {
				return nil, err
			}

			accountLocation := utils.GetLocation(profile.Timezone)
			if accountLocation == nil {
				accountLocation = utils.GetLocation("GMT+8")
			}

			accountNow := time.Now().In(accountLocation)
			accountToday := time.Date(accountNow.Year(), accountNow.Month(), accountNow.Day(), 0, 0, 0, 0, accountLocation)

			if err := s.mongo.UpdateProfilePOIMetric(profile.AccountNumber, id, metric); err != nil {
				return nil, err
			}

			poi := profile.PointsOfInterest[0]
			lastSpikeUpdate := poi.Metric.Details.Symptoms.LastSpikeUpdate.In(accountLocation)
			lastSpikeDay := time.Date(lastSpikeUpdate.Year(), lastSpikeUpdate.Month(), lastSpikeUpdate.Day(), 0, 0, 0, 0, accountLocation)

			if currentSpikeLength := len(metric.Details.Symptoms.LastSpikeList); currentSpikeLength > 0 {
				if accountToday.Sub(lastSpikeDay) == 0 { // spike in the same day
					if currentSpikeLength > len(poi.Metric.Details.Symptoms.LastSpikeList) {
						symptomsSpikeAccounts = append(symptomsSpikeAccounts, profile.AccountNumber)
					}
				} else {
					symptomsSpikeAccounts = append(symptomsSpikeAccounts, profile.AccountNumber)
				}
			}

			var changed bool
			if len(profile.PointsOfInterest) != 0 {
				changed = score.CheckScoreColorChange(poi.Score, metric.Score)
			}

			if changed {
				logger.Debug("State color changed", zap.Any("old", profile.Metric.Score), zap.Any("new", metric.Score))
				stateChangedAccounts = append(stateChangedAccounts, profile.AccountNumber)
			}
		}
	} else { // poiID == ''
		profile, err := s.mongo.GetProfile(accountNumber)
		if err != nil {
			return nil, err
		}

		accountLocation := utils.GetLocation(profile.Timezone)
		if accountLocation == nil {
			accountLocation = utils.GetLocation("GMT+8")
		}

		accountNow := time.Now().In(accountLocation)
		accountToday := time.Date(accountNow.Year(), accountNow.Month(), accountNow.Day(), 0, 0, 0, 0, accountLocation)

		if err := s.mongo.UpdateProfileMetric(accountNumber, metric); err != nil {
			return nil, err
		}

		if time.Since(profile.LastNudge[schema.NudgeBehaviorOnSymptomSpikeArea]) > 90*time.Minute { // 90 minutes of delay between nudges
			if profile.Metric.SymptomDelta < 10 && metric.SymptomDelta >= 10 { // from a non-spike area to a spike area
				remindGoodBehavior = true
			}
		}

		lastSpikeUpdate := profile.Metric.Details.Symptoms.LastSpikeUpdate.In(accountLocation)
		lastSpikeDay := time.Date(lastSpikeUpdate.Year(), lastSpikeUpdate.Month(), lastSpikeUpdate.Day(), 0, 0, 0, 0, accountLocation)

		if currentSpikeLength := len(metric.Details.Symptoms.LastSpikeList); currentSpikeLength > 0 {
			if spikeDayDelta := accountToday.Sub(lastSpikeDay); spikeDayDelta == 0 { // spike in the same day
				if currentSpikeLength > len(profile.Metric.Details.Symptoms.LastSpikeList) {
					symptomsSpikeAccounts = append(symptomsSpikeAccounts, profile.AccountNumber)
				}
			} else if spikeDayDelta > 0 {
				symptomsSpikeAccounts = append(symptomsSpikeAccounts, profile.AccountNumber)
			} else {
				logger.Warn("last spike day is greater than today", zap.String("accountNumber", accountNumber),
					zap.Any("accountToday", accountToday), zap.Any("lastSpikeDay", lastSpikeDay))
			}
		}

		var changed bool
		if profile.Metric.LastUpdate != 0 {
			changed = score.CheckScoreColorChange(profile.Metric.Score, metric.Score)
		}

		if changed {
			logger.Debug("State color changed", zap.Any("old", profile.Metric.Score), zap.Any("new", metric.Score))
			stateChangedAccounts = append(stateChangedAccounts, profile.AccountNumber)
		}

		// only report the risk area when a location state change is detected and
		// the score it lower than 67 (with color yellow and red)
		if changed && metric.Score < 67 {
			reportRiskArea = true
		}
	}

	logger.Debug("finish state refreshing",
		zap.Any("stateChangedAccounts", stateChangedAccounts),
		zap.Any("symptomsSpikeAccounts", symptomsSpikeAccounts))

	return &NotificationProfile{
		StateChangedAccounts:  stateChangedAccounts,
		SymptomsSpikeAccounts: symptomsSpikeAccounts,
		ReportRiskArea:        reportRiskArea,
		RemindGoodBehavior:    remindGoodBehavior,
	}, nil
}

// NotifyLocationStateActivity is to send notification to end users for notifing the
// significant changes of location states.
func (s *ScoreUpdateWorker) NotifyLocationStateActivity(ctx context.Context, id string, accounts []string) error {
	logger := activity.GetLogger(ctx)
	if len(accounts) == 0 {
		logger.Warn("Send notification without accounts")
		return nil
	}

	if id == "" {
		return s.notificationCenter.NotifyAccountsByTemplate(accounts, viper.GetString("onesignal.template.new_location_status_change"),
			map[string]interface{}{
				"notification_type": "RISK_LEVEL_CHANGED",
			},
		)
	}

	return s.notificationCenter.NotifyAccountsByTemplate(accounts, viper.GetString("onesignal.template.saved_location_status_change"),
		map[string]interface{}{
			"notification_type": "RISK_LEVEL_CHANGED",
			"poi_id":            id,
		},
	)
}

// CalculateAccountStateActivity calculates metrics by a given account's location
func (s *ScoreUpdateWorker) CalculateAccountStateActivity(ctx context.Context, accountNumber string) (*schema.Metric, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Query account profile for calculating state.", zap.String("accountNumber", accountNumber))

	profile, err := s.mongo.GetProfile(accountNumber)
	if err != nil {
		return nil, err
	}
	logger.Info("Account profile.", zap.Any("profile", profile))

	if profile.Location == nil {
		return nil, ErrInvalidLocation
	}

	location := schema.Location{
		Latitude:  profile.Location.Coordinates[1],
		Longitude: profile.Location.Coordinates[0],
	}

	logger.Info("Calculate metric by location.", zap.Any("location", location))
	rawMetrics, err := s.mongo.CollectRawMetrics(location)
	if err != nil {
		return nil, err
	}

	metric := score.CalculateMetric(*rawMetrics, profile.ScoreCoefficient)

	// FIXME: `profile.IndividualMetric` could be outdated
	// for users who don't use the app for a long time
	score, _ := score.CalculateIndividualAutonomyScore(profile.IndividualMetric, metric)
	if err := s.mongo.AddScoreRecord(accountNumber, schema.ScoreRecordTypeIndividual, score, time.Now().UTC().Unix()); err != nil {
		sentry.CaptureException(err)
	}

	return &metric, nil
}
