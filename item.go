package redispool

import "github.com/allentom/youcrawl"

type RedisItemSerializer interface {
	Unmarshal(rawData map[string]interface{}) (interface{}, error)
}

type DefaultItemSerializer struct{}

func (s *DefaultItemSerializer) Unmarshal(rawData map[string]interface{}) (interface{}, error) {
	return youcrawl.DefaultItem{Store: rawData["Store"].(map[string]interface{})}, nil
}
