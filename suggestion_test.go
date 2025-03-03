package redissug_test

import (
	"context"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"testing"

	"github.com/redis/go-redis/v9"

	redissug "github.com/ndx-technologies/go-redis-suggest"
)

func TestRedisSuggest(t *testing.T) {
	if testing.Short() {
		t.Skip("network; redis;")
	}

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	ctx := t.Context()
	id := "test-sug:" + strconv.Itoa(rand.Int())
	s := redissug.RedisSuggestionClient{DB: rdb}
	t.Cleanup(func() { s.DelAll(context.Background(), id) })

	t.Run("when adding suggestions", func(t *testing.T) {
		size, err := s.SugAdd(ctx, id, "text", 1, false, "payload1")
		if err != nil {
			t.Error("SugAdd failed:", err)
		}
		if size != 1 {
			t.Error("expected size 1, got", size)
		}

		size, err = s.SugAdd(ctx, id, "test", 2, false, "payload2")
		if err != nil {
			t.Error("SugAdd failed:", err)
		}
		if size != 2 {
			t.Error("expected size 2, got", size)
		}

		size, err = s.SugAdd(ctx, id, "tent", 2, false, "")
		if err != nil {
			t.Error("SugAdd failed:", err)
		}
		if size != 3 {
			t.Error("expected size 3, got", size)
		}

		size, err = s.SugAdd(ctx, id, "tent", 1, true, "")
		if err != nil {
			t.Error("SugAdd failed:", err)
		}
		if size != 3 {
			t.Error("expected size 3, got", size)
		}

		t.Run("when getting length, then 3", func(t *testing.T) {
			count, err := s.SugLen(ctx, id)
			if err != nil {
				t.Error("SugLen failed:", err)
			}
			if count != 3 {
				t.Error("expected count 3, got", count)
			}
		})

		t.Run("when prefix, then return all matched in highest to lowest score order", func(t *testing.T) {
			suggestions, err := s.SugGet(ctx, id, "te", 10, redissug.SugGetOptions{})
			if err != nil {
				t.Error("SugGet failed:", err)
			}
			exp := []redissug.Suggestion{{Text: "tent"}, {Text: "test"}, {Text: "text"}}
			if !slices.Equal(suggestions, exp) {
				t.Error("expected", exp, "got", suggestions)
			}
		})

		t.Run("when exact prefix, then only exact is returned", func(t *testing.T) {
			suggestions, err := s.SugGet(ctx, id, "tex", 10, redissug.SugGetOptions{})
			if err != nil {
				t.Error("SugGet failed:", err)
			}
			exp := []redissug.Suggestion{{Text: "text"}}
			if !slices.Equal(suggestions, exp) {
				t.Error("expected", exp, "got", suggestions)
			}
		})

		t.Run("when fuzzy prefix, then multiple returned", func(t *testing.T) {
			suggestions, err := s.SugGet(ctx, id, "tex", 10, redissug.SugGetOptions{Fuzzy: true})
			if err != nil {
				t.Error("SugGet failed:", err)
			}
			exp := []redissug.Suggestion{{Text: "text"}, {Text: "tent"}, {Text: "test"}}
			if !slices.Equal(suggestions, exp) {
				t.Error("expected", exp, "got", suggestions)
			}
		})

		t.Run("when getting with payloads, then payloads are returned", func(t *testing.T) {
			suggestions, err := s.SugGet(ctx, id, "te", 10, redissug.SugGetOptions{WithPayloads: true})
			if err != nil {
				t.Error("SugGet failed:", err)
			}
			exp := []redissug.Suggestion{{Text: "tent"}, {Text: "test", Payload: "payload2"}, {Text: "text", Payload: "payload1"}}
			if !slices.Equal(suggestions, exp) {
				t.Error("expected", exp, "got", suggestions)
			}
		})

		t.Run("when adding same suggestion with different score and payload", func(t *testing.T) {
			size, err := s.SugAdd(ctx, id, "text", 5, false, "new_payload")
			if err != nil {
				t.Error("SugAdd failed:", err)
			}
			if size != 3 {
				t.Error("expected size 3, got", size)
			}

			t.Run("then new payload is set", func(t *testing.T) {
				suggestions, err := s.SugGet(ctx, id, "tex", 10, redissug.SugGetOptions{WithPayloads: true})
				if err != nil {
					t.Error("SugGet failed:", err)
				}
				exp := []redissug.Suggestion{{Text: "text", Payload: "new_payload"}}
				if !slices.Equal(suggestions, exp) {
					t.Error("expected", exp, "got", suggestions)
				}
			})
		})
	})

	t.Run("when deleting suggestion", func(t *testing.T) {
		if err := s.SugDel(ctx, id, "test"); err != nil {
			t.Error("SugDel failed:", err)
		}

		t.Run("when deleting non existing, then no error and ok", func(t *testing.T) {
			if err := s.SugDel(ctx, id, "asdf"); err != redis.Nil {
				t.Error("expected redis.Nil, got", err)
			}
		})

		t.Run("when getting suggestions, then no deleted entry", func(t *testing.T) {
			suggestions, err := s.SugGet(ctx, id, "te", 10, redissug.SugGetOptions{})
			if err != nil {
				t.Error("SugGet failed:", err)
			}
			exp := []redissug.Suggestion{{Text: "text"}, {Text: "tent"}}
			if !slices.Equal(suggestions, exp) {
				t.Error("expected", exp, "got", suggestions)
			}
		})

		t.Run("then length is decremented", func(t *testing.T) {
			count, err := s.SugLen(ctx, id)
			if err != nil {
				t.Error("SugLen failed:", err)
			}
			if count != 2 {
				t.Error("expected count 2, got", count)
			}
		})
	})

	t.Run("when non existing key", func(t *testing.T) {
		nonExistentKey := "non-existent:" + strconv.Itoa(rand.Int())

		t.Run("when getting suggestions, then redis.Nil", func(t *testing.T) {
			suggestions, err := s.SugGet(ctx, nonExistentKey, "te", 10, redissug.SugGetOptions{})
			if err != redis.Nil {
				t.Error("expected redis.Nil, got", err)
			}
			if len(suggestions) != 0 {
				t.Error("expected empty slice, got", suggestions)
			}
		})

		t.Run("when getting length, then 0", func(t *testing.T) {
			count, err := s.SugLen(ctx, nonExistentKey)
			if err != nil {
				t.Error("SugLen failed:", err)
			}
			if count != 0 {
				t.Error("expected count 0, got", count)
			}
		})

		t.Run("when deleting non existing, then redis nil", func(t *testing.T) {
			if err := s.SugDel(ctx, nonExistentKey, "asdf"); err != redis.Nil {
				t.Error("expected redis.Nil, got", err)
			}
		})
	})
}
