package pubsub

import (
	"cloud.google.com/go/pubsub"
	"context"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	pubsubConfig "mxtransporter/config/pubsub"
	"mxtransporter/pkg/errors"
	"strings"
	"time"
)

type (
	pubsubClient interface {
		pubsubTopic(ctx context.Context, topicID string) error
		pubsubSubscription(ctx context.Context, topicID string, subscriptionID string) error
		publishMessage(ctx context.Context, topicID string, csArray []string) error
	}

	PubsubImpl struct {
		Pubsub pubsubClient
	}

	PubsubClientImpl struct {
		PubsubClient *pubsub.Client
		Log          *zap.SugaredLogger
	}
)

func (p *PubsubClientImpl) pubsubTopic(ctx context.Context, topicID string) error {
	topic := p.PubsubClient.Topic(topicID)
	defer topic.Stop()

	topicExistence, err := topic.Exists(ctx)
	if err != nil {
		return errors.InternalServerErrorPubSubFind.Wrap("Failed to check topic existence.", err)
	}
	if topicExistence == false {
		p.Log.Info("Topic is not exists. Creating a topic.")

		var err error
		_, err = p.PubsubClient.CreateTopic(ctx, topicID)
		if err != nil {
			return errors.InternalServerErrorPubSubCreate.Wrap("Failed to create topic.", err)
		}
		p.Log.Info("Successed to create topic. ")
	}

	return nil
}

func (p *PubsubClientImpl) pubsubSubscription(ctx context.Context, topicID string, subscriptionID string) error {
	subscription := p.PubsubClient.Subscription(subscriptionID)

	subscriptionExistence, err := subscription.Exists(ctx)
	if err != nil {
		return errors.InternalServerErrorPubSubFind.Wrap("Failed to check subscription existence.", err)
	}
	if subscriptionExistence == false {
		p.Log.Info("Subscription is not exists. Creating a subscription.")

		var err error
		_, err = p.PubsubClient.CreateSubscription(ctx, subscriptionID, pubsub.SubscriptionConfig{
			Topic:             p.PubsubClient.Topic(topicID),
			AckDeadline:       60 * time.Second,
			RetentionDuration: 24 * time.Hour,
		})
		if err != nil {
			return errors.InternalServerErrorPubSubCreate.Wrap("Failed to create subscription.", err)
		}
		p.Log.Info("Successed to create subscription. ")
	}
	return nil
}

func (p *PubsubClientImpl) publishMessage(ctx context.Context, topicID string, csArray []string) error {
	topic := p.PubsubClient.Topic(topicID)
	defer topic.Stop()

	topic.Publish(ctx, &pubsub.Message{
		Data: []byte(strings.Join(csArray, "|")),
	})

	return nil
}

func (p *PubsubImpl) ExportToPubsub(ctx context.Context, cs primitive.M) error {
	psCfg := pubsubConfig.PubSubConfig()

	topicID := psCfg.MongoDbDatabase

	if err := p.Pubsub.pubsubTopic(ctx, topicID); err != nil {
		return err
	}

	subscriptionID := psCfg.MongoDbCollection

	if err := p.Pubsub.pubsubSubscription(ctx, topicID, subscriptionID); err != nil {
		return err
	}

	id, err := json.Marshal(cs["_id"])
	if err != nil {
		errors.InternalServerErrorJsonMarshal.Wrap("Failed to marshal json.", err)
	}
	opType := cs["operationType"].(string)
	clusterTime := cs["clusterTime"].(primitive.Timestamp).T
	fullDoc, err := json.Marshal(cs["fullDocument"])
	if err != nil {
		errors.InternalServerErrorJsonMarshal.Wrap("Failed to marshal json.", err)
	}
	ns, err := json.Marshal(cs["ns"])
	if err != nil {
		errors.InternalServerErrorJsonMarshal.Wrap("Failed to marshal json.", err)
	}
	docKey, err := json.Marshal(cs["documentKey"])
	if err != nil {
		errors.InternalServerErrorJsonMarshal.Wrap("Failed to marshal json.", err)
	}
	updDesc, err := json.Marshal(cs["updateDescription"])
	if err != nil {
		errors.InternalServerErrorJsonMarshal.Wrap("Failed to marshal json.", err)
	}

	r := []string{
		string(id),
		opType,
		time.Unix(int64(clusterTime), 0).Format("2006-01-02 15:04:05"),
		string(fullDoc),
		string(ns),
		string(docKey),
		string(updDesc),
	}

	if err := p.Pubsub.publishMessage(ctx, topicID, r); err != nil {
		return err
	}

	return nil
}