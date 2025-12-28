package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/escalopa/quran-read-bot/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	stateKeyPrefix = "fsm:state:"
	dataKeyPrefix  = "fsm:data:"
	defaultTTL     = 24 * time.Hour
)

type FSM struct {
	client *redis.Client
}

func NewFSM(uri string) (*FSM, error) {
	opts, err := redis.ParseURL(uri)
	if err != nil {
		return nil, fmt.Errorf("parse redis URI: %w", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect to redis: %w", err)
	}

	return &FSM{client: client}, nil
}

func (f *FSM) Close() error {
	return f.client.Close()
}

// SetState sets the current state for a user
func (f *FSM) SetState(ctx context.Context, userID string, state domain.State) error {
	key := stateKeyPrefix + userID
	return f.client.Set(ctx, key, string(state), defaultTTL).Err()
}

// GetState gets the current state for a user
func (f *FSM) GetState(ctx context.Context, userID string) (domain.State, error) {
	key := stateKeyPrefix + userID
	val, err := f.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return domain.StateStart, nil
	}
	if err != nil {
		return "", fmt.Errorf("get state: %w", err)
	}
	return domain.State(val), nil
}

// DeleteState deletes the state for a user
func (f *FSM) DeleteState(ctx context.Context, userID string) error {
	key := stateKeyPrefix + userID
	return f.client.Del(ctx, key).Err()
}

// SetData sets temporary data for a user's current session
func (f *FSM) SetData(ctx context.Context, userID, key, value string) error {
	dataKey := fmt.Sprintf("%s%s:%s", dataKeyPrefix, userID, key)
	return f.client.Set(ctx, dataKey, value, defaultTTL).Err()
}

// GetData gets temporary data for a user's current session
func (f *FSM) GetData(ctx context.Context, userID, key string) (string, error) {
	dataKey := fmt.Sprintf("%s%s:%s", dataKeyPrefix, userID, key)
	val, err := f.client.Get(ctx, dataKey).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("data not found")
	}
	if err != nil {
		return "", fmt.Errorf("get data: %w", err)
	}
	return val, nil
}

// DeleteData deletes temporary data for a user
func (f *FSM) DeleteData(ctx context.Context, userID, key string) error {
	dataKey := fmt.Sprintf("%s%s:%s", dataKeyPrefix, userID, key)
	return f.client.Del(ctx, dataKey).Err()
}
