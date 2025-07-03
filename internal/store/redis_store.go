package store

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	script *redis.Script
}

// 쓰는 작업이 하나뿐이니 아예 스크립트 고정
func NewRedisStore(client *redis.Client) *RedisStore {
	lua := `
		local current = redis.call("GET", KEYS[1])
		if not current then
		  current = 0
		else
		  current = tonumber(current)
		end
		
		if current >= tonumber(ARGV[1]) then
		  return nil
		else
		  current = redis.call("INCR", KEYS[1])
		  return current
		end`
	return &RedisStore{
		client: client,
		script: redis.NewScript(lua),
	}
}

// TryIssueCoupon: Redis INCR + limit 확인
func (r *RedisStore) TryIssueCoupon(ctx context.Context, campaignID string, maxCoupons int) (int64, error) {

	key := "coupon_count:" + campaignID
	res, err := r.script.Run(ctx, r.client, []string{key}, strconv.Itoa(maxCoupons)).Result()
	if err == redis.Nil {
		return 0, fmt.Errorf("no coupons left")
	}
	if err != nil {
		return 0, err
	}

	count, ok := res.(int64)
	if !ok {
		return 0, fmt.Errorf("unexpected result type")
	}

	return count, nil
}

func (r *RedisStore) RollbackIssueCoupon(ctx context.Context, campaignID string) error {
	key := "coupon_count:" + campaignID
	_, err := r.client.Decr(ctx, key).Result()
	return err
}
