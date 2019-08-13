# model
## 解决问题
- 封装增删改查与redis缓存加载，使用方法见example
- 分页列表加载，使用zset，见example/zset.go
- 结构体格式化为json字符串，加载到redis的string类型中，见example/string.go
- 结构体按照字段tag，加载到redis的hash类型中，见example/hash.go

## 待解决问题
- 批量获取不支持go-redis的cluster模式，因为在此模式执行mget mset，当key不属于同一个节点会报错
