package store

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/bitmark-inc/autonomy-api/schema"
)

type Feedback interface {
	CreateFeedback(symptom schema.Feedback) (string, error)
}

func (m *mongoDB) CreateFeedback(feedback schema.Feedback) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c := m.client.Database(m.database)

	r, err := c.Collection(schema.FeedbackCollection).InsertOne(ctx, &feedback)
	if err != nil {
		return "", err
	}

	id, ok := r.InsertedID.(primitive.ObjectID)
	if ok {
		return id.Hex(), nil
	}
	return "", fmt.Errorf("incorrect inserted id")
}
