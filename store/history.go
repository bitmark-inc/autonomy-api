package store

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bitmark-inc/autonomy-api/schema"
)

type History interface {
	GetReportedSymptoms(accountNumber string, earierThan, limit int64, lang string) ([]*schema.SymptomReportData, error)
	GetReportedBehaviors(accountNumber string, earierThan, limit int64, lang string) ([]*schema.BehaviorReportData, error)
	GetReportedLocations(accountNumber string, earierThan, limit int64) ([]schema.Geographic, error)
}

func (m *mongoDB) GetReportedSymptoms(accountNumber string, earierThan, limit int64, lang string) ([]*schema.SymptomReportData, error) {
	c := m.client.Database(m.database).Collection(schema.SymptomReportCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	symptoms, err := m.ListOfficialSymptoms(lang)
	if err != nil {
		return nil, err
	}
	// TODO: put the mapping in memory
	mapping := make(map[schema.SymptomType]schema.Symptom)
	for _, s := range symptoms {
		mapping[s.ID] = s
	}

	query, options := historyQuery(accountNumber, earierThan, limit)
	cur, err := c.Find(ctx, query, options)
	if err != nil {
		return nil, err
	}

	reports := make([]*schema.SymptomReportData, 0)
	for cur.Next(ctx) {
		var r schema.SymptomReportData
		if err := cur.Decode(&r); err != nil {
			return nil, err
		}
		// TODO: make OfficialSymptoms as []*schema.Symptom
		translatedSymptoms := make([]schema.Symptom, 0)
		for _, s := range r.OfficialSymptoms {
			s.Name = mapping[s.ID].Name
			s.Desc = mapping[s.ID].Desc
			translatedSymptoms = append(translatedSymptoms, s)
		}
		r.OfficialSymptoms = translatedSymptoms
		reports = append(reports, &r)
	}

	return reports, nil
}

func (m *mongoDB) GetReportedBehaviors(accountNumber string, earierThan, limit int64, lang string) ([]*schema.BehaviorReportData, error) {
	c := m.client.Database(m.database).Collection(schema.BehaviorReportCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	behaviors, err := m.ListOfficialBehavior(lang)
	if err != nil {
		return nil, err
	}
	// TODO: put the mapping in memory
	mapping := make(map[schema.GoodBehaviorType]schema.Behavior)
	for _, b := range behaviors {
		mapping[b.ID] = b
	}

	query, options := historyQuery(accountNumber, earierThan, limit)
	cur, err := c.Find(ctx, query, options)
	if err != nil {
		return nil, err
	}

	reports := make([]*schema.BehaviorReportData, 0)
	for cur.Next(ctx) {
		var r schema.BehaviorReportData
		if err := cur.Decode(&r); err != nil {
			return nil, err
		}
		// TODO: make OfficialBehaviors as []*schema.Behavior
		translatedBehaviors := make([]schema.Behavior, 0)
		for _, b := range r.OfficialBehaviors {
			b.Name = mapping[b.ID].Name
			b.Desc = mapping[b.ID].Desc
			translatedBehaviors = append(translatedBehaviors, b)
		}
		r.OfficialBehaviors = translatedBehaviors
		reports = append(reports, &r)
	}

	return reports, nil
}

func (m *mongoDB) GetReportedLocations(accountNumber string, earierThan, limit int64) ([]schema.Geographic, error) {
	c := m.client.Database(m.database).Collection(schema.GeographicCollection)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	query, options := historyQuery(accountNumber, earierThan, limit)
	cur, err := c.Find(ctx, query, options)
	if err != nil {
		return nil, err
	}

	result := make([]schema.Geographic, 0)
	for cur.Next(ctx) {
		var g schema.Geographic
		if err = cur.Decode(&g); err != nil {
			return nil, err
		}
		result = append(result, g)
	}

	return result, nil
}

func historyQuery(accountNumber string, earierThan, limit int64) (bson.M, *options.FindOptions) {
	query := bson.M{
		"account_number": accountNumber,
		"ts":             bson.M{"$lt": earierThan},
	}
	options := options.Find()
	options = options.SetSort(bson.M{"ts": -1}).SetLimit(limit)
	return query, options
}
