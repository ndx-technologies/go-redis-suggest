package redissug

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// RedisSuggestionClient that provides SUG commands.
type RedisSuggestionClient struct {
	DB *redis.Client
}

type Suggestion struct {
	Text    string
	Payload string
}

// SugAdd returns current size of suggestion dictionary
func (s RedisSuggestionClient) SugAdd(ctx context.Context, key string, suggestion string, score float64, incr bool, payload string) (int, error) {
	args := []any{"FT.SUGADD", key, suggestion, score}

	if incr {
		args = append(args, "INCR")
	}

	if payload != "" {
		args = append(args, "PAYLOAD", payload)
	}

	return s.DB.Do(ctx, args...).Int()
}

type SugGetOptions struct {
	Fuzzy        bool
	WithPayloads bool
}

// SugGet returns suggestions sorted from highest score to lowest score.
// Scores may not be the same as the scores used in SugAdd, which is why they are not exposed.
func (s RedisSuggestionClient) SugGet(ctx context.Context, key string, prefix string, max int, opts SugGetOptions) ([]Suggestion, error) {
	args := []any{"FT.SUGGET", key, prefix}

	if opts.Fuzzy {
		args = append(args, "FUZZY")
	}
	if opts.WithPayloads {
		args = append(args, "WITHPAYLOADS")
	}
	if max > 0 {
		args = append(args, "MAX", max)
	}

	result, err := s.DB.Do(ctx, args...).Slice()
	if err != nil {
		return nil, err
	}

	suggestions := make([]Suggestion, 0, max)

	for i := 0; i < len(result); {
		var s Suggestion

		s.Text, _ = result[i].(string)
		i++

		if opts.WithPayloads {
			s.Payload, _ = result[i].(string)
			i++
		}

		suggestions = append(suggestions, s)
	}

	return suggestions, nil
}

// SugDel deletes a suggestion from a suggestion index
func (s RedisSuggestionClient) SugDel(ctx context.Context, key string, suggestion string) error {
	v, err := s.DB.Do(ctx, "FT.SUGDEL", key, suggestion).Int()
	if err != nil {
		return err
	}
	if v == 0 {
		return redis.Nil
	}
	return nil
}

// SugLen returns size of an auto-complete suggestion dictionary
func (s RedisSuggestionClient) SugLen(ctx context.Context, key string) (int, error) {
	return s.DB.Do(ctx, "FT.SUGLEN", key).Int()
}

func (s RedisSuggestionClient) DelAll(ctx context.Context, keys ...string) error {
	_, err := s.DB.Del(ctx, keys...).Result()
	return err
}
