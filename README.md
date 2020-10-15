# YouCrawl-RedisPool
TaskPool component in [YouCrawl](https://github.com/AllenTom/YouCrawl) implemented using Redis
## Features
- Support distributed deployment
- Support pause and recovery
- Support different Engines to use one Task Pool at the same time

## Example
```go
func main() {
	e := youcrawl.NewEngine(&youcrawl.EngineOption{
		MaxRequest: 3,
		Daemon:     false,
	})
	// create TaskPool
	pool := redispool.NewRedisTaskPool(e,&redispool.DefaultItemSerializer{})
	// connect to Redis
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

### Item serialization
YouCrawl Items are stored in Redis using JSON strings. For `youcrawl.DefaultItem`, you can use the built-in `redispool.DefaultItemSerializer`.

If you need to use a custom Item, you need to implement the `RedisItemSerializer` interface


```go
type RedisItemSerializer interface {
	Unmarshal(rawData map[string]interface{}) (interface{}, error)
}
```