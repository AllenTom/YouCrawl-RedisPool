# YouCrawl-RedisPool
[YouCrawl](https://github.com/AllenTom/YouCrawl) 中的TaskPool组件，使用Redis实现。

## 特性
- 支持分布式部署
- 支持任务中断与恢复
- 支持不同Engine同时使用一个Task Pool

## 示例
```go
func main() {
	e := youcrawl.NewEngine(&youcrawl.EngineOption{
		MaxRequest: 3,
		Daemon:     false,
	})
	// 创建一个TaskPool
	pool := redispool.NewRedisTaskPool(e,&redispool.DefaultItemSerializer{})
	// 连接至Redis
	err := pool.InitRedis(&redis.Options{
		DB: 0,
		Addr: "localhost:6379",
	})
	if err != nil {
		log.Fatal(err)
	}
	e.UseTaskPool(pool)
	e.AddURLs("http://www.example.com")
	e.RunAndWait()
}
```

### Item序列化
YouCrawl 使用Item，在存储时使用JSON字符串的形式储存在Redis中，对于`youcrawl.DefaultItem`可以使用内置的`redispool.DefaultItemSerializer`。

如果需要使用自定义的Item，需要实现RedisItemSerializer接口

```go
type RedisItemSerializer interface {
	Unmarshal(rawData map[string]interface{}) (interface{}, error)
}
```