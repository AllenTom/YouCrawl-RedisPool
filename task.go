package redispool

import (
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"github.com/mitchellh/mapstructure"
)

type TaskEntity struct {
	ID   string      `mapstructure:"id"`
	Url  string      `mapstructure:"url"`
	Item interface{} `mapstructure:"not_item"`
}

func (e *TaskEntity) Serialize() (map[string]interface{}, error) {
	data := map[string]interface{}{
		"id":  e.ID,
		"url": e.Url,
	}
	if e.Item != nil {
		rawData, err := json.Marshal(e.Item)
		if err != nil {
			return nil, err
		}
		data["item"] = string(rawData)
	}
	return data, nil
}

func (e *TaskEntity) Load(db *redis.Client, serializer RedisItemSerializer) error {
	rawDataMap := db.HGetAll(ctx, e.ID).Val()
	err := mapstructure.Decode(rawDataMap, e)
	if item, itemExist := rawDataMap["item"]; itemExist && item != "" {
		rawData := make(map[string]interface{})
		err = json.Unmarshal([]byte(rawDataMap["item"]), &rawData)
		if err != nil {
			return err
		}
		e.Item, err = serializer.Unmarshal(rawData)
		if err != nil {
			return err
		}
	}
	return err
}
