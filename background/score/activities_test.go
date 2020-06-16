package score

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/worker"
	"go.uber.org/zap"

	"github.com/bitmark-inc/autonomy-api/external/cadence"
	"github.com/bitmark-inc/autonomy-api/mocks"
	"github.com/bitmark-inc/autonomy-api/schema"
)

var (
	fakeProfileAccount1 = "fcqu8Deozrzv6pQ5EqSsdvAHG1SbTafHqviUjVvP1mDmbPyiBU"
	fakeProfileAccount2 = "eEfqMcw7ExsoUhULQ7H41r5avLJxpzPWf4vVm6pGWB1o2wvyjR"
)

var (
	regularPOI = &schema.POI{
		Location: &schema.GeoJSON{
			Coordinates: []float64{120.125, 25.125},
		},
		Metric: schema.Metric{
			LastUpdate: time.Now().Add(-10 * time.Second).Unix(),
		},
	}

	frequentUpdatePOI = &schema.POI{
		Location: &schema.GeoJSON{
			Coordinates: []float64{120.125, 25.125},
		},
		Metric: schema.Metric{
			LastUpdate: time.Now().Unix(),
		},
	}

	poiNoLocation = &schema.POI{
		Location: nil,
		Metric: schema.Metric{
			LastUpdate: time.Now().Add(-10 * time.Second).Unix(),
		},
	}
)

type ScoreActivityTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env               *testsuite.TestActivityEnvironment
	worker            *ScoreUpdateWorker
	mockCtrl          *gomock.Controller
	mongoMock         *mocks.MockMongoStore
	notificationMock  *mocks.MockNotificationCenter
	testAccountNumber string
	testPOIID         string
}

func (ts *ScoreActivityTestSuite) SetupSuite() {
	ts.SetLogger(zap.NewNop())
	ts.testAccountNumber = "e5KNBJCzwBqAyQzKx1pv8CR4MacrUBBTQpWwAbmcLbYNsEg5WS"
	ts.testPOIID = "5e9806ae554b311b328e2f91"
}

func (ts *ScoreActivityTestSuite) SetupTest() {
	ts.env = ts.NewTestActivityEnvironment()
	ts.env.SetWorkerOptions(worker.Options{
		BackgroundActivityContext: context.Background(),
		DataConverter:             cadence.NewMsgPackDataConverter(),
	})

	ts.mockCtrl = gomock.NewController(ts.T())

	mongoMock = mocks.NewMockMongoStore(ts.mockCtrl)
	nc := mocks.NewMockNotificationCenter(ts.mockCtrl)

	testWorker.mongo = mongoMock
	testWorker.notificationCenter = nc
	ts.mongoMock = mongoMock
	ts.notificationMock = nc
	ts.worker = testWorker
}

func (ts *ScoreActivityTestSuite) TearDownTest() {
	ts.mockCtrl.Finish()
}

// TestCalculatePOIStateActivity tests the `CalculatePOIStateActivity` in normal way
func (ts *ScoreActivityTestSuite) TestCalculatePOIStateActivity() {
	poiID, err := primitive.ObjectIDFromHex(ts.testPOIID)
	ts.NoError(err)

	ts.mongoMock.
		EXPECT().
		GetPOI(gomock.Eq(poiID)).
		Return(regularPOI, nil)
	ts.mongoMock.
		EXPECT().
		CollectRawMetrics(gomock.Eq(schema.Location{
			Latitude:  regularPOI.Location.Coordinates[1],
			Longitude: regularPOI.Location.Coordinates[0],
		})).
		Return(&schema.Metric{}, nil)

	values, err := ts.env.ExecuteActivity(ts.worker.CalculatePOIStateActivity, ts.testPOIID)
	ts.NoError(err)

	var metric schema.Metric
	err = values.Get(&metric)
	ts.NoError(err)
}

// TestCalculatePOIStateActivityPOINoLocation tests the `CalculatePOIStateActivity` without given a location
func (ts *ScoreActivityTestSuite) TestCalculatePOIStateActivityPOINoLocation() {
	poiID, err := primitive.ObjectIDFromHex(ts.testPOIID)
	ts.NoError(err)

	ts.mongoMock.
		EXPECT().
		GetPOI(gomock.Eq(poiID)).
		Return(poiNoLocation, nil)

	values, err := ts.env.ExecuteActivity(ts.worker.CalculatePOIStateActivity, ts.testPOIID)
	ts.EqualError(err, "invalid location")
	ts.Nil(values)
}

// TestCalculatePOIStateActivityFrequentUpdate tests the `CalculatePOIStateActivity` is invoked frequently
func (ts *ScoreActivityTestSuite) TestCalculatePOIStateActivityFrequentUpdate() {
	poiID, err := primitive.ObjectIDFromHex(ts.testPOIID)
	ts.NoError(err)

	ts.mongoMock.
		EXPECT().
		GetPOI(gomock.Eq(poiID)).
		Return(frequentUpdatePOI, nil)

	values, err := ts.env.ExecuteActivity(ts.worker.CalculatePOIStateActivity, ts.testPOIID)
	ts.EqualError(err, "too frequent update")
	ts.Nil(values)
}

// TestCheckLocationSpikeActivity tests `CheckLocationSpikeActivity` in a normal way
func (ts *ScoreActivityTestSuite) TestCheckLocationSpikeActivity() {
	ts.mongoMock.
		EXPECT().
		FindSymptomsByIDs(gomock.Eq([]string{
			schema.COVID19Symptoms[0].ID,
			schema.COVID19Symptoms[1].ID,
		})).
		Return([]schema.Symptom{
			schema.COVID19Symptoms[0],
			schema.COVID19Symptoms[1],
		}, nil)

	values, err := ts.env.ExecuteActivity(ts.worker.CheckLocationSpikeActivity, []string{
		schema.COVID19Symptoms[0].ID,
		schema.COVID19Symptoms[1].ID,
	})

	ts.NoError(err)
	symptoms := []schema.Symptom{}
	err = values.Get(&symptoms)
	ts.NoError(err)
	ts.Len(symptoms, 2)
}

// TestCheckLocationSpikeActivityWithError tests `CheckLocationSpikeActivity` with error return
func (ts *ScoreActivityTestSuite) TestCheckLocationSpikeActivityWithError() {
	ts.mongoMock.
		EXPECT().
		FindSymptomsByIDs(gomock.Eq([]string{
			schema.COVID19Symptoms[0].ID,
			schema.COVID19Symptoms[1].ID,
		})).
		Return(nil, fmt.Errorf("can not find symptoms"))

	values, err := ts.env.ExecuteActivity(ts.worker.CheckLocationSpikeActivity, []string{
		schema.COVID19Symptoms[0].ID,
		schema.COVID19Symptoms[1].ID,
	})

	ts.EqualError(err, "can not find symptoms")
	ts.Nil(values)
}

// TestRefreshLocationStateActivityForAccountNormalRun tests RefreshLocationStateActivity
// without any notification
func (ts *ScoreActivityTestSuite) TestRefreshLocationStateActivityForAccountNormalRun() {
	metricToUpdate := schema.Metric{Score: 98}

	ts.mongoMock.
		EXPECT().
		GetProfile(gomock.Eq(ts.testAccountNumber)).
		Return(&schema.Profile{
			AccountNumber: ts.testAccountNumber,
			Timezone:      "GMT+8",
		}, nil)

	ts.mongoMock.
		EXPECT().
		UpdateProfileMetric(gomock.Eq(ts.testAccountNumber), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil)

	values, err := ts.env.ExecuteActivity(ts.worker.RefreshLocationStateActivity, ts.testAccountNumber, "", metricToUpdate)
	ts.NoError(err)

	var np NotificationProfile
	err = values.Get(&np)
	ts.NoError(err)
	ts.False(np.ReportRiskArea)
	ts.False(np.RemindGoodBehavior)
	ts.Len(np.SymptomsSpikeAccounts, 0)
	ts.Len(np.StateChangedAccounts, 0)
}

// TestRefreshLocationStateActivityForAccountEnterSymptomSpikeArea tests an account **SHOULD** be
// marked `RemindGoodBehavior` when he enters a symptom spike area.
func (ts *ScoreActivityTestSuite) TestRefreshLocationStateActivityForAccountEnterSymptomSpikeArea() {
	metricToUpdate := schema.Metric{
		Score:        55,
		SymptomDelta: 10,
	}

	ts.mongoMock.
		EXPECT().
		GetProfile(gomock.Eq(ts.testAccountNumber)).
		Return(&schema.Profile{
			AccountNumber: ts.testAccountNumber,
			Timezone:      "GMT+8",
			LastNudge: schema.NudgeTime{
				schema.NudgeBehaviorOnSymptomSpikeArea: time.Now().Add(-100 * time.Minute),
			},
			Metric: schema.Metric{
				SymptomDelta: 5,
			},
		}, nil)

	ts.mongoMock.
		EXPECT().
		UpdateProfileMetric(gomock.Eq(ts.testAccountNumber), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil)

	values, err := ts.env.ExecuteActivity(ts.worker.RefreshLocationStateActivity, ts.testAccountNumber, "", metricToUpdate)
	ts.NoError(err)

	var np NotificationProfile
	err = values.Get(&np)
	ts.NoError(err)
	ts.False(np.ReportRiskArea)
	ts.True(np.RemindGoodBehavior)
	ts.Len(np.SymptomsSpikeAccounts, 0)
	ts.Len(np.StateChangedAccounts, 0)
}

// TestRefreshLocationStateActivityForAccountStayInSymptomSpikeArea tests an account **SHOULD NOT** be
// marked `RemindGoodBehavior` when he stays in a symptom spike area.
func (ts *ScoreActivityTestSuite) TestRefreshLocationStateActivityForAccountStayInSymptomSpikeArea() {
	metricToUpdate := schema.Metric{
		Score:        55,
		SymptomDelta: 10,
	}

	ts.mongoMock.
		EXPECT().
		GetProfile(gomock.Eq(ts.testAccountNumber)).
		Return(&schema.Profile{
			AccountNumber: ts.testAccountNumber,
			Timezone:      "GMT+8",
			LastNudge: schema.NudgeTime{
				schema.NudgeBehaviorOnSymptomSpikeArea: time.Now().Add(-100 * time.Minute),
			},
			Metric: schema.Metric{
				SymptomDelta: 10,
			},
		}, nil)

	ts.mongoMock.
		EXPECT().
		UpdateProfileMetric(gomock.Eq(ts.testAccountNumber), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil)

	values, err := ts.env.ExecuteActivity(ts.worker.RefreshLocationStateActivity, ts.testAccountNumber, "", metricToUpdate)
	ts.NoError(err)

	var np NotificationProfile
	err = values.Get(&np)
	ts.NoError(err)
	ts.False(np.ReportRiskArea)
	ts.False(np.RemindGoodBehavior)
	ts.Len(np.SymptomsSpikeAccounts, 0)
	ts.Len(np.StateChangedAccounts, 0)
}

// TestRefreshLocationStateActivityForAccountEnterSymptomSpikeAreaAgainWithin90Minutes tests an account **SHOULD NOT** be
// marked `RemindGoodBehavior` when he enters in a symptom spike area again within 90 minutes.
func (ts *ScoreActivityTestSuite) TestRefreshLocationStateActivityForAccountEnterSymptomSpikeAreaAgainWithin90Minutes() {
	metricToUpdate := schema.Metric{
		Score:        55,
		SymptomDelta: 10,
	}

	ts.mongoMock.
		EXPECT().
		GetProfile(gomock.Eq(ts.testAccountNumber)).
		Return(&schema.Profile{
			AccountNumber: ts.testAccountNumber,
			Timezone:      "GMT+8",
			LastNudge: schema.NudgeTime{
				schema.NudgeBehaviorOnSymptomSpikeArea: time.Now().Add(-10 * time.Minute),
			},
			Metric: schema.Metric{
				SymptomDelta: 5,
			},
		}, nil)

	ts.mongoMock.
		EXPECT().
		UpdateProfileMetric(gomock.Eq(ts.testAccountNumber), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil)

	values, err := ts.env.ExecuteActivity(ts.worker.RefreshLocationStateActivity, ts.testAccountNumber, "", metricToUpdate)
	ts.NoError(err)

	var np NotificationProfile
	err = values.Get(&np)
	ts.NoError(err)
	ts.False(np.ReportRiskArea)
	ts.False(np.RemindGoodBehavior)
	ts.Len(np.SymptomsSpikeAccounts, 0)
	ts.Len(np.StateChangedAccounts, 0)
}

// TestRefreshLocationStateActivityForAccountLastSpikeTodayButListIncrease checks if the last spike day today,
// it will send a notifiction only when the counts of the spike list is greater than previous.
func (ts *ScoreActivityTestSuite) TestRefreshLocationStateActivityForAccountLastSpikeTodayButListIncrease() {
	metricToUpdate := schema.Metric{
		Score: 98,
		Details: schema.Details{
			Symptoms: schema.SymptomDetail{
				LastSpikeList: []string{
					schema.COVID19Symptoms[0].ID,
					schema.COVID19Symptoms[1].ID,
					schema.COVID19Symptoms[2].ID,
				},
			},
		},
	}

	ts.mongoMock.
		EXPECT().
		GetProfile(gomock.Eq(ts.testAccountNumber)).
		Return(&schema.Profile{
			AccountNumber: ts.testAccountNumber,
			Timezone:      "GMT+8",
			Metric: schema.Metric{
				Details: schema.Details{
					Symptoms: schema.SymptomDetail{
						LastSpikeUpdate: time.Now(),
						LastSpikeList: []string{
							schema.COVID19Symptoms[0].ID,
							schema.COVID19Symptoms[1].ID,
						},
					},
				},
			},
		}, nil)

	ts.mongoMock.
		EXPECT().
		UpdateProfileMetric(gomock.Eq(ts.testAccountNumber), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil)

	values, err := ts.env.ExecuteActivity(ts.worker.RefreshLocationStateActivity, ts.testAccountNumber, "", metricToUpdate)
	ts.NoError(err)

	var np NotificationProfile
	err = values.Get(&np)
	ts.NoError(err)
	ts.False(np.ReportRiskArea)
	ts.False(np.RemindGoodBehavior)
	ts.Len(np.SymptomsSpikeAccounts, 1)
	ts.Len(np.StateChangedAccounts, 0)
}

// TestRefreshLocationStateActivityForAccountLastSpikePriviousDay checks if the last spike day is before today,
// it will always send a notifiction
func (ts *ScoreActivityTestSuite) TestRefreshLocationStateActivityForAccountLastSpikePriviousDay() {
	metricToUpdate := schema.Metric{
		Score: 98,
		Details: schema.Details{
			Symptoms: schema.SymptomDetail{
				LastSpikeList: []string{
					schema.COVID19Symptoms[0].ID,
				},
			},
		},
	}

	ts.mongoMock.
		EXPECT().
		GetProfile(gomock.Eq(ts.testAccountNumber)).
		Return(&schema.Profile{
			AccountNumber: ts.testAccountNumber,
			Timezone:      "GMT+8",
			Metric: schema.Metric{
				Details: schema.Details{
					Symptoms: schema.SymptomDetail{
						LastSpikeUpdate: time.Now().Add(-24 * time.Hour),
						LastSpikeList: []string{
							schema.COVID19Symptoms[0].ID,
							schema.COVID19Symptoms[1].ID,
						},
					},
				},
			},
		}, nil)

	ts.mongoMock.
		EXPECT().
		UpdateProfileMetric(gomock.Eq(ts.testAccountNumber), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil)

	values, err := ts.env.ExecuteActivity(ts.worker.RefreshLocationStateActivity, ts.testAccountNumber, "", metricToUpdate)
	ts.NoError(err)

	var np NotificationProfile
	err = values.Get(&np)
	ts.NoError(err)
	ts.False(np.ReportRiskArea)
	ts.False(np.RemindGoodBehavior)
	ts.Len(np.SymptomsSpikeAccounts, 1)
	ts.Len(np.StateChangedAccounts, 0)
}

// TestRefreshLocationStateActivityForAccountWhereScoreChangesSignificant tests an account is added
// into StateChangedAccounts when the score is changed significantly
func (ts *ScoreActivityTestSuite) TestRefreshLocationStateActivityForAccountWhereScoreChangesSignificant() {
	metricToUpdate := schema.Metric{
		Score: 77,
	}

	ts.mongoMock.
		EXPECT().
		GetProfile(gomock.Eq(ts.testAccountNumber)).
		Return(&schema.Profile{
			AccountNumber: ts.testAccountNumber,
			Timezone:      "GMT+8",
			Metric: schema.Metric{
				LastUpdate: time.Now().Unix(),
				Score:      55,
			},
		}, nil)

	ts.mongoMock.
		EXPECT().
		UpdateProfileMetric(gomock.Eq(ts.testAccountNumber), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil)

	values, err := ts.env.ExecuteActivity(ts.worker.RefreshLocationStateActivity, ts.testAccountNumber, "", metricToUpdate)
	ts.NoError(err)

	var np NotificationProfile
	err = values.Get(&np)
	ts.NoError(err)
	ts.False(np.ReportRiskArea)
	ts.False(np.RemindGoodBehavior)
	ts.Len(np.SymptomsSpikeAccounts, 0)
	ts.Len(np.StateChangedAccounts, 1)
}

// TestRefreshLocationStateActivityForAccountOnEnteringHighRiskArea tests an account **SHOULD** be
// marked `ReportRiskArea` when he enters a high risk area.
func (ts *ScoreActivityTestSuite) TestRefreshLocationStateActivityForAccountOnEnteringHighRiskArea() {
	metricToUpdate := schema.Metric{
		Score: 55,
	}

	ts.mongoMock.
		EXPECT().
		GetProfile(gomock.Eq(ts.testAccountNumber)).
		Return(&schema.Profile{
			AccountNumber: ts.testAccountNumber,
			Timezone:      "GMT+8",
			Metric: schema.Metric{
				LastUpdate: time.Now().Unix(),
				Score:      77,
			},
		}, nil)

	ts.mongoMock.
		EXPECT().
		UpdateProfileMetric(gomock.Eq(ts.testAccountNumber), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil)

	values, err := ts.env.ExecuteActivity(ts.worker.RefreshLocationStateActivity, ts.testAccountNumber, "", metricToUpdate)
	ts.NoError(err)

	var np NotificationProfile
	err = values.Get(&np)
	ts.NoError(err)
	ts.True(np.ReportRiskArea)
	ts.False(np.RemindGoodBehavior)
	ts.Len(np.SymptomsSpikeAccounts, 0)
	ts.Len(np.StateChangedAccounts, 1)
}

// TestRefreshLocationStateActivityForAccountStayInHighRiskArea tests an account **SHOULD NOT** be
// marked `ReportRiskArea` when he stays in a high risk area.
func (ts *ScoreActivityTestSuite) TestRefreshLocationStateActivityForAccountStayInHighRiskArea() {
	metricToUpdate := schema.Metric{
		Score: 55,
	}

	ts.mongoMock.
		EXPECT().
		GetProfile(gomock.Eq(ts.testAccountNumber)).
		Return(&schema.Profile{
			AccountNumber: ts.testAccountNumber,
			Timezone:      "GMT+8",
			Metric: schema.Metric{
				LastUpdate: time.Now().Unix(),
				Score:      55,
			},
		}, nil)

	ts.mongoMock.
		EXPECT().
		UpdateProfileMetric(gomock.Eq(ts.testAccountNumber), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil)

	values, err := ts.env.ExecuteActivity(ts.worker.RefreshLocationStateActivity, ts.testAccountNumber, "", metricToUpdate)
	ts.NoError(err)

	var np NotificationProfile
	err = values.Get(&np)
	ts.NoError(err)
	ts.False(np.ReportRiskArea)
	ts.False(np.RemindGoodBehavior)
	ts.Len(np.SymptomsSpikeAccounts, 0)
	ts.Len(np.StateChangedAccounts, 0)
}

// TestRefreshLocationStateActivityForPOINormalRun tests RefreshLocationStateActivity
// without any notification
func (ts *ScoreActivityTestSuite) TestRefreshLocationStateActivityForPOINormalRun() {
	poiID, err := primitive.ObjectIDFromHex(ts.testPOIID)
	ts.NoError(err)

	ts.mongoMock.
		EXPECT().
		UpdatePOIMetric(gomock.Eq(poiID), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil)

	ts.mongoMock.
		EXPECT().
		GetProfilesByPOI(gomock.Eq(ts.testPOIID)).
		Return([]schema.Profile{
			{
				AccountNumber: ts.testAccountNumber,
				Timezone:      "GMT+8",
				PointsOfInterest: []schema.ProfilePOI{
					{
						ID: poiID,
					},
				},
			},
		}, nil)
	ts.mongoMock.
		EXPECT().
		GetPOIResourceMetric(
			gomock.Eq(poiID),
		).
		Return(schema.POIRatingsMetric{}, nil)
	ts.mongoMock.
		EXPECT().
		UpdateProfilePOIMetric(gomock.Eq(ts.testAccountNumber), gomock.Eq(poiID), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil)
	ts.mongoMock.
		EXPECT().
		AddScoreRecord(
			gomock.Eq(ts.testPOIID),
			schema.ScoreRecordTypePOI,
			gomock.AssignableToTypeOf(float64(1)),
			gomock.AssignableToTypeOf(int64(1)),
		).
		Return(nil)

	values, err := ts.env.ExecuteActivity(ts.worker.RefreshLocationStateActivity, "", ts.testPOIID, schema.Metric{})
	ts.NoError(err)

	var np NotificationProfile
	err = values.Get(&np)
	ts.NoError(err)
	ts.False(np.ReportRiskArea)
	ts.False(np.RemindGoodBehavior)
	ts.Len(np.SymptomsSpikeAccounts, 0)
	ts.Len(np.StateChangedAccounts, 0)
}

// TestRefreshLocationStateActivityForPOILastSymptomSpikeInYesterday tests accounts should be
// added into `SymptomsSpikeAccounts` if the last spike is yesterday
func (ts *ScoreActivityTestSuite) TestRefreshLocationStateActivityForPOILastSymptomSpikeInYesterday() {
	poiID, err := primitive.ObjectIDFromHex(ts.testPOIID)
	ts.NoError(err)

	metricToUpdate := schema.Metric{
		Score: 98,
		Details: schema.Details{
			Symptoms: schema.SymptomDetail{
				LastSpikeList: []string{
					schema.COVID19Symptoms[0].ID,
					schema.COVID19Symptoms[1].ID,
				},
			},
		},
	}

	ts.mongoMock.
		EXPECT().
		UpdatePOIMetric(gomock.Eq(poiID), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil)

	ts.mongoMock.
		EXPECT().
		GetProfilesByPOI(gomock.Eq(ts.testPOIID)).
		Return([]schema.Profile{
			{
				AccountNumber: fakeProfileAccount1,
				Timezone:      "GMT+8",
				PointsOfInterest: []schema.ProfilePOI{
					{
						ID:    poiID,
						Score: 98,
						Metric: schema.Metric{
							Score: 98,
							Details: schema.Details{
								Symptoms: schema.SymptomDetail{
									LastSpikeUpdate: time.Now().Add(-24 * time.Hour),
									LastSpikeList: []string{
										schema.COVID19Symptoms[0].ID,
										schema.COVID19Symptoms[1].ID,
									},
								},
							},
						},
					},
				},
			},
			{
				AccountNumber: fakeProfileAccount2,
				Timezone:      "GMT+8",
				PointsOfInterest: []schema.ProfilePOI{
					{
						ID:    poiID,
						Score: 98,
						Metric: schema.Metric{
							Score: 98,
							Details: schema.Details{
								Symptoms: schema.SymptomDetail{
									LastSpikeUpdate: time.Now().Add(-24 * time.Hour),
									LastSpikeList: []string{
										schema.COVID19Symptoms[0].ID,
									},
								},
							},
						},
					},
				},
			},
		}, nil)
	ts.mongoMock.
		EXPECT().
		GetPOIResourceMetric(
			gomock.Eq(poiID),
		).
		Return(schema.POIRatingsMetric{Resources: []schema.POIResourceRating{{SumOfScore: 5, Score: 5, Ratings: 1}}}, nil)
	ts.mongoMock.
		EXPECT().
		UpdateProfilePOIMetric(gomock.AssignableToTypeOf(""), gomock.Eq(poiID), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil).Times(2)
	ts.mongoMock.
		EXPECT().
		AddScoreRecord(
			gomock.Eq(ts.testPOIID),
			schema.ScoreRecordTypePOI,
			gomock.AssignableToTypeOf(float64(1)),
			gomock.AssignableToTypeOf(int64(1)),
		).
		Return(nil)

	values, err := ts.env.ExecuteActivity(ts.worker.RefreshLocationStateActivity, "", ts.testPOIID, metricToUpdate)
	ts.NoError(err)

	var np NotificationProfile
	err = values.Get(&np)
	ts.NoError(err)
	ts.False(np.ReportRiskArea)
	ts.False(np.RemindGoodBehavior)
	ts.Len(np.SymptomsSpikeAccounts, 2)
	ts.Len(np.StateChangedAccounts, 0)
}

// TestRefreshLocationStateActivityForPOILastSymptomSpikeInToday tests accounts would be
// added into `SymptomsSpikeAccounts` only if spike counts is greater in a single day
func (ts *ScoreActivityTestSuite) TestRefreshLocationStateActivityForPOILastSymptomSpikeInToday() {
	poiID, err := primitive.ObjectIDFromHex(ts.testPOIID)
	ts.NoError(err)

	metricToUpdate := schema.Metric{
		Score: 98,
		Details: schema.Details{
			Symptoms: schema.SymptomDetail{
				LastSpikeList: []string{
					schema.COVID19Symptoms[0].ID,
					schema.COVID19Symptoms[1].ID,
				},
			},
		},
	}

	ts.mongoMock.
		EXPECT().
		UpdatePOIMetric(gomock.Eq(poiID), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil)

	ts.mongoMock.
		EXPECT().
		GetProfilesByPOI(gomock.Eq(ts.testPOIID)).
		Return([]schema.Profile{
			{
				AccountNumber: fakeProfileAccount1,
				Timezone:      "GMT+8",
				PointsOfInterest: []schema.ProfilePOI{
					{
						ID:    poiID,
						Score: 98,
						Metric: schema.Metric{
							Score: 98,
							Details: schema.Details{
								Symptoms: schema.SymptomDetail{
									LastSpikeUpdate: time.Now(),
									LastSpikeList: []string{
										schema.COVID19Symptoms[0].ID,
									},
								},
							},
						},
					},
				},
			},
			{
				AccountNumber: fakeProfileAccount2,
				Timezone:      "GMT+8",
				PointsOfInterest: []schema.ProfilePOI{
					{
						ID:    poiID,
						Score: 98,
						Metric: schema.Metric{
							Score: 98,
							Details: schema.Details{
								Symptoms: schema.SymptomDetail{
									LastSpikeUpdate: time.Now(),
									LastSpikeList: []string{
										schema.COVID19Symptoms[0].ID,
										schema.COVID19Symptoms[1].ID,
									},
								},
							},
						},
					},
				},
			},
		}, nil)

	ts.mongoMock.
		EXPECT().
		UpdateProfilePOIMetric(gomock.AssignableToTypeOf(""), gomock.Eq(poiID), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil).Times(2)
	ts.mongoMock.
		EXPECT().
		GetPOIResourceMetric(
			gomock.Eq(poiID),
		).
		Return(schema.POIRatingsMetric{Resources: []schema.POIResourceRating{{SumOfScore: 5, Score: 5, Ratings: 1}}}, nil)
	ts.mongoMock.
		EXPECT().
		AddScoreRecord(
			gomock.Eq(ts.testPOIID),
			schema.ScoreRecordTypePOI,
			gomock.AssignableToTypeOf(float64(1)),
			gomock.AssignableToTypeOf(int64(1)),
		).
		Return(nil)

	values, err := ts.env.ExecuteActivity(ts.worker.RefreshLocationStateActivity, "", ts.testPOIID, metricToUpdate)
	ts.NoError(err)

	var np NotificationProfile
	err = values.Get(&np)
	ts.NoError(err)
	ts.False(np.ReportRiskArea)
	ts.False(np.RemindGoodBehavior)
	ts.Len(np.SymptomsSpikeAccounts, 1)
	ts.Len(np.StateChangedAccounts, 0)
	ts.Equal(fakeProfileAccount1, np.SymptomsSpikeAccounts[0])
}

// TestRefreshLocationStateActivityForPOILastStateChanges tests accounts with significant
// changes in its saved poi will be added into `StateChangedAccounts`
func (ts *ScoreActivityTestSuite) TestRefreshLocationStateActivityForPOILastStateChanges() {
	poiID, err := primitive.ObjectIDFromHex(ts.testPOIID)
	ts.NoError(err)

	metricToUpdate := schema.Metric{
		Score: 98,
	}

	ts.mongoMock.
		EXPECT().
		UpdatePOIMetric(gomock.Eq(poiID), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil)

	ts.mongoMock.
		EXPECT().
		GetProfilesByPOI(gomock.Eq(ts.testPOIID)).
		Return([]schema.Profile{
			{
				AccountNumber: fakeProfileAccount1,
				Timezone:      "GMT+8",
				PointsOfInterest: []schema.ProfilePOI{
					{
						ID:    poiID,
						Score: 98,
					},
				},
			},
			{
				AccountNumber: fakeProfileAccount2,
				Timezone:      "GMT+8",
				PointsOfInterest: []schema.ProfilePOI{
					{
						ID:    poiID,
						Score: 55,
					},
				},
			},
		}, nil)

	ts.mongoMock.
		EXPECT().
		UpdateProfilePOIMetric(gomock.AssignableToTypeOf(""), gomock.Eq(poiID), gomock.AssignableToTypeOf(schema.Metric{})).
		Return(nil).Times(2)
	ts.mongoMock.
		EXPECT().
		GetPOIResourceMetric(
			gomock.Eq(poiID),
		).
		Return(schema.POIRatingsMetric{Resources: []schema.POIResourceRating{{SumOfScore: 5, Score: 5, Ratings: 1}}}, nil)
	ts.mongoMock.
		EXPECT().
		AddScoreRecord(
			gomock.Eq(ts.testPOIID),
			schema.ScoreRecordTypePOI,
			gomock.AssignableToTypeOf(float64(1)),
			gomock.AssignableToTypeOf(int64(1)),
		).
		Return(nil)

	values, err := ts.env.ExecuteActivity(ts.worker.RefreshLocationStateActivity, "", ts.testPOIID, metricToUpdate)
	ts.NoError(err)

	var np NotificationProfile
	err = values.Get(&np)
	ts.NoError(err)
	ts.False(np.ReportRiskArea)
	ts.False(np.RemindGoodBehavior)
	ts.Len(np.SymptomsSpikeAccounts, 0)
	ts.Len(np.StateChangedAccounts, 1)
	ts.Equal(fakeProfileAccount2, np.StateChangedAccounts[0])
}

// TestCalculateAccountStateActivityWithoutLocation tests CalculateAccountStateActivity
// for an account without location
func (ts *ScoreActivityTestSuite) TestCalculateAccountStateActivityWithoutLocation() {
	ts.mongoMock.
		EXPECT().
		GetProfile(gomock.Eq(ts.testAccountNumber)).
		Return(&schema.Profile{
			AccountNumber: ts.testAccountNumber,
			Timezone:      "GMT+8",
			Metric: schema.Metric{
				LastUpdate: time.Now().Unix(),
				Score:      55,
			},
		}, nil)

	_, err := ts.env.ExecuteActivity(ts.worker.CalculateAccountStateActivity, ts.testAccountNumber)
	ts.EqualError(err, "invalid location")
}

// TestCalculateAccountStateActivityNormal tests CalculateAccountStateActivity
// running normally
func (ts *ScoreActivityTestSuite) TestCalculateAccountStateActivityNormal() {
	var testProfile = &schema.Profile{
		AccountNumber: ts.testAccountNumber,
		Timezone:      "GMT+8",
		Metric: schema.Metric{
			LastUpdate: time.Now().Unix(),
			Score:      55,
		},
		Location: &schema.GeoJSON{
			Coordinates: []float64{120.1256, 25.1256},
		},
		IndividualMetric: schema.IndividualMetric{
			Score:          100.0,
			ScoreYesterday: 0.0,
		},
	}
	ts.mongoMock.
		EXPECT().
		GetProfile(gomock.Eq(ts.testAccountNumber)).
		Return(testProfile, nil)

	ts.mongoMock.
		EXPECT().
		CollectRawMetrics(gomock.Eq(schema.Location{
			Latitude:  testProfile.Location.Coordinates[1],
			Longitude: testProfile.Location.Coordinates[0],
		})).
		Return(&schema.Metric{}, nil)
	ts.mongoMock.
		EXPECT().
		AddScoreRecord(
			gomock.Eq(ts.testAccountNumber),
			schema.ScoreRecordTypeIndividual,
			gomock.Eq(85.0), // 0.8*100 + 0.2*25
			gomock.AssignableToTypeOf(int64(1)),
		).
		Return(nil)

	_, err := ts.env.ExecuteActivity(ts.worker.CalculateAccountStateActivity, ts.testAccountNumber)
	ts.NoError(err)
}

// TestCalculateAccountStateActivityNormal tests CalculateAccountStateActivity
// where it fails to collect raw metrics
func (ts *ScoreActivityTestSuite) TestCalculateAccountStateActivityWithCollectError() {
	var testProfile = &schema.Profile{
		AccountNumber: ts.testAccountNumber,
		Timezone:      "GMT+8",
		Metric: schema.Metric{
			LastUpdate: time.Now().Unix(),
			Score:      55,
		},
		Location: &schema.GeoJSON{
			Coordinates: []float64{120.1256, 25.1256},
		},
	}
	ts.mongoMock.
		EXPECT().
		GetProfile(gomock.Eq(ts.testAccountNumber)).
		Return(testProfile, nil)

	ts.mongoMock.
		EXPECT().
		CollectRawMetrics(gomock.Eq(schema.Location{
			Latitude:  testProfile.Location.Coordinates[1],
			Longitude: testProfile.Location.Coordinates[0],
		})).
		Return(nil, fmt.Errorf("can not collect metrics"))

	_, err := ts.env.ExecuteActivity(ts.worker.CalculateAccountStateActivity, ts.testAccountNumber)
	ts.EqualError(err, "can not collect metrics")
}

func (ts *ScoreActivityTestSuite) TestNotifyLocationStateActivity() {
	ts.notificationMock.EXPECT().NotifyAccountsByTemplate(
		gomock.Eq([]string{ts.testAccountNumber}),
		gomock.AssignableToTypeOf(""),
		gomock.Eq(map[string]interface{}{
			"notification_type": "RISK_LEVEL_CHANGED",
		})).
		Return(nil).Times(1)

	_, err := ts.env.ExecuteActivity(ts.worker.NotifyLocationStateActivity, "", []string{ts.testAccountNumber})
	ts.NoError(err)
}

func (ts *ScoreActivityTestSuite) TestNotifyLocationStateActivityWithSpecificPOIID() {
	ts.notificationMock.EXPECT().NotifyAccountsByTemplate(
		gomock.Eq([]string{ts.testAccountNumber}),
		gomock.AssignableToTypeOf(""),
		gomock.Eq(map[string]interface{}{
			"notification_type": "RISK_LEVEL_CHANGED",
			"poi_id":            ts.testPOIID,
		})).
		Return(nil).Times(1)

	_, err := ts.env.ExecuteActivity(ts.worker.NotifyLocationStateActivity, ts.testPOIID, []string{ts.testAccountNumber})
	ts.NoError(err)
}

func TestScoreActivity(t *testing.T) {
	suite.Run(t, new(ScoreActivityTestSuite))
}
