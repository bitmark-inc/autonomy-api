package store

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/bitmark-inc/autonomy-api/schema"
)

type Survey interface {
	CreateSurvey(symptom schema.Survey) (string, error)
}

// CreateSurvey adds a survey result into db and returns its id
func (m *mongoDB) CreateSurvey(survey schema.Survey) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if survey.SurveyID == "" {
		return "", fmt.Errorf("survey_id should not be empty")
	}

	r, err := m.client.Database(m.database).Collection(schema.SurveyCollection).InsertOne(ctx, &survey)
	if err != nil {
		return "", err
	}

	id, ok := r.InsertedID.(primitive.ObjectID)
	if ok {
		return id.Hex(), nil
	}
	return "", fmt.Errorf("incorrect inserted id")
}
