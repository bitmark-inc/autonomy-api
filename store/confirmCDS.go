package store

import (
	"context"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/schema"
	"github.com/bitmark-inc/autonomy-api/score"
)

var (
	ErrNoConfirmDataset       = fmt.Errorf("no data-set")
	ErrInvalidConfirmDataset  = fmt.Errorf("invalid confirm data-set")
	ErrPoliticalTypeGeoInfo   = fmt.Errorf("no political type geo info")
	ErrConfirmDataFetch       = fmt.Errorf("fetch cds confirm data fail")
	ErrConfirmDecode          = fmt.Errorf("decode confirm data fail")
	ErrConfirmDuplicateRecord = fmt.Errorf("confirm data duplicate")
)

type ConfirmCDS interface {
	ReplaceCDS(result []schema.CDSData, country string) error
	CreateCDS(result []schema.CDSData, country string) error
	GetCDSActive(loc schema.Location, referenceTime int64) (float64, float64, float64, error)
	DeleteCDSUnused(country string, timeBefore int64) error
	ContinuousDataCDSConfirm(loc schema.Location, num int64, timeBefore int64) ([]schema.CDSScoreDataSet, error)
}

func (m *mongoDB) ReplaceCDS(result []schema.CDSData, country string) error {
	collection, ok := schema.CDSCountyCollectionMatrix[schema.CDSCountryType(country)]
	if !ok {
		return errors.New("no cds country availible")
	}
	if len(result) <= 0 {
		log.WithFields(log.Fields{"prefix": mongoLogPrefix}).Debug("no record to update")
		return nil
	}

	for _, v := range result {
		filter := bson.M{"name": v.Name, "report_ts": v.ReportTime}
		replacement := bson.M{
			"name":        v.Name,
			"city":        v.City,
			"county":      v.County,
			"state":       v.State,
			"country":     v.Country,
			"level":       v.Level,
			"cases":       v.Cases,
			"deaths":      v.Deaths,
			"recovered":   v.Recovered,
			"active":      v.Active,
			"report_ts":   v.ReportTime,
			"update_ts":   v.UpdateTime,
			"report_date": v.ReportTimeDate,
			"countryId":   v.CountryID,
			"stateId":     v.StateID,
			"countyId":    v.CountyID,
			"location":    v.Location,
			"tz":          v.Timezone,
		}
		opts := options.Replace().SetUpsert(true)
		_, err := m.client.Database(m.database).Collection(collection).ReplaceOne(context.Background(), filter, replacement, opts)
		if err != nil {
			if errs, hasErr := err.(mongo.BulkWriteException); hasErr {
				if 1 == len(errs.WriteErrors) && DuplicateKeyCode == errs.WriteErrors[0].Code {
					log.WithField("prefix", mongoLogPrefix).Warnf("cds update with error: %s", err)
				}
			}
		}
	}
	return nil
}

func (m *mongoDB) CreateCDS(result []schema.CDSData, country string) error {
	collection, ok := schema.CDSCountyCollectionMatrix[schema.CDSCountryType(country)]
	if !ok {
		return errors.New("no cds country availible")
	}
	data := make([]interface{}, len(result))
	for i, v := range result {
		data[i] = v
	}
	opts := options.InsertMany().SetOrdered(false)
	res, err := m.client.Database(m.database).Collection(collection).InsertMany(context.Background(), data, opts)
	if err != nil {
		if errs, hasErr := err.(mongo.BulkWriteException); hasErr {
			if 1 == len(errs.WriteErrors) && DuplicateKeyCode == errs.WriteErrors[0].Code {
				log.WithFields(log.Fields{"prefix": mongoLogPrefix, "err": errs}).Warn("createCDSData Insert data")
				return nil
			}
		}
	}
	if res != nil {
		log.WithFields(log.Fields{"prefix": mongoLogPrefix, "records": len(res.InsertedIDs)}).Debug("CreateCDSData Insert data")
	}
	return nil
}

func (m *mongoDB) DeleteCDSUnused(country string, timeBefore int64) error {
	collection, ok := schema.CDSCountyCollectionMatrix[schema.CDSCountryType(country)]
	if !ok {
		return errors.New("no cds country availible")
	}
	filter := bson.M{"report_ts": bson.D{{"$lte", timeBefore}}}
	res, err := m.client.Database(m.database).Collection(collection).DeleteMany(context.Background(), filter)
	if err != nil {
		log.WithField("prefix", mongoLogPrefix).Warnf("cds delete unused record wih error: %s", err)
		return err
	}
	log.WithFields(log.Fields{"prefix": mongoLogPrefix, "records": res.DeletedCount}).Debug("DeleteCDSUnused delete data")
	return nil
}

// GetCDSActive returns latest metrics available for the given `referenceTime`,
// so only history data before `referenceTime` is taken into consideration.
// The returned values are active count, the delta of active count compared to previous day and the change rate.
func (m mongoDB) GetCDSActive(loc schema.Location, referenceTime int64) (float64, float64, float64, error) {
	log.WithFields(log.Fields{"prefix": mongoLogPrefix, "country": loc.Country, "state": loc.State, "county": loc.County}).Debug("GetCDSConfirm geo info")

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	collectionName, ok := schema.CDSCountyCollectionMatrix[schema.CDSCountryType(loc.Country)]
	if !ok {
		return 0, 0, 0, ErrNoConfirmDataset
	}

	filter := bson.M{"report_ts": bson.M{"$lte": referenceTime}}
	opts := options.Find().SetSort(bson.M{"report_ts": -1}).SetLimit(2)
	cur, err := m.client.Database(m.database).Collection(collectionName).Find(context.Background(), filter, opts)
	if nil != err {
		log.WithField("prefix", mongoLogPrefix).WithError(err).Errorf("failed to get CDS data")
		return 0, 0, 0, ErrConfirmDataFetch
	}

	var results []schema.CDSData
	for cur.Next(ctx) {
		var result schema.CDSData
		if errDecode := cur.Decode(&result); errDecode != nil {
			return 0, 0, 0, ErrConfirmDecode
		}
		log.WithField("prefix", mongoLogPrefix).Debugf("cds query name: %s date:%s", result.Name, result.ReportTimeDate)
		results = append(results, result)
	}

	var delta float64
	var today, yesterday schema.CDSData
	if len(results) >= 2 {
		if results[0].ReportTime > results[1].ReportTime {
			today = results[0]
			yesterday = results[1]
		} else if results[0].ReportTime < results[1].ReportTime {
			today = results[1]
			yesterday = results[0]
			log.WithField("prefix", mongoLogPrefix).Errorf("cds query result error: name :%v  today:%v date:%v", results[0].Name, results[0].ReportTime, results[1].ReportTime)
		} else {
			return 0, 0, 0, ErrConfirmDuplicateRecord
		}
		delta = today.Active - yesterday.Active
		return today.Active, delta, score.ChangeRate(today.Active, yesterday.Active), nil

	} else if 1 == len(results) {
		today = results[0]
		return today.Active, today.Active, score.ChangeRate(today.Active, 0), nil
	}
	return 0, 0, 0, nil
}

func (m mongoDB) ContinuousDataCDSConfirm(loc schema.Location, windowSize int64, timeBefore int64) ([]schema.CDSScoreDataSet, error) {
	log.WithFields(log.Fields{"prefix": mongoLogPrefix, "country": loc.Country, "lv1": loc.State, "lv2": loc.County}).Debug("ContinuousDataCDSConfirm geo info")
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var col *mongo.Collection
	var filter bson.M
	var opts *options.FindOptions
	switch loc.Country { //  Currently this function support only USA data
	case schema.CdsTaiwan:
		col = m.client.Database(m.database).Collection(schema.CDSCountyCollectionMatrix[schema.CDSCountryType(schema.CdsTaiwan)])
		opts = options.Find().SetSort(bson.M{"report_ts": -1}).SetLimit(windowSize + 1)
		filter = bson.M{}
		if timeBefore > 0 {
			filter = bson.M{"report_ts": bson.D{{"$lte", timeBefore}}}
		}
	case schema.CdsIceland:
		col = m.client.Database(m.database).Collection(schema.CDSCountyCollectionMatrix[schema.CDSCountryType(schema.CdsIceland)])
		opts = options.Find().SetSort(bson.M{"report_ts": -1}).SetLimit(windowSize + 1)
		filter = bson.M{}
		if timeBefore > 0 {
			filter = bson.M{"report_ts": bson.D{{"$lte", timeBefore}}}
		}
	case schema.CdsUSA:
		col = m.client.Database(m.database).Collection(schema.CDSCountyCollectionMatrix[schema.CDSCountryType(schema.CdsUSA)])
		opts = options.Find().SetSort(bson.M{"report_ts": -1}).SetLimit(windowSize + 1)
		filter = bson.M{"county": loc.County, "state": loc.State}
		if timeBefore > 0 {
			filter = bson.M{"county": loc.County, "state": loc.State, "report_ts": bson.D{{"$lte", timeBefore}}}
		}
	default:
		return nil, ErrNoConfirmDataset
	}

	var results []schema.CDSScoreDataSet
	cur, err := col.Find(context.Background(), filter, opts)
	if nil != err {
		log.WithField("prefix", mongoLogPrefix).Errorf("%v: %s", ErrConfirmDataFetch, err)
		return nil, ErrConfirmDataFetch
	}

	now := schema.CDSScoreDataSet{}

	for cur.Next(ctx) {
		var result schema.CDSScoreDataSet
		if errDecode := cur.Decode(&result); errDecode != nil {
			log.WithField("prefix", mongoLogPrefix).Errorf("cds Decode with error: %s", errDecode)
			return nil, errDecode
		}
		if len(now.Name) > 0 { // now data is valid
			head := make([]schema.CDSScoreDataSet, 1)
			head[0] = schema.CDSScoreDataSet{Name: now.Name, Cases: now.Cases - result.Cases}
			results = append(head, results...)
		}
		now = result
	}
	if len(results) == 0 && now.Name != "" { // only one record
		results = append(results, now)
	}
	cur.Close(ctx)
	return results, nil
}
