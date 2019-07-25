# model
## 解决问题
- 封装增删改查与redis缓存加载，减少编码工作,使用方法见example
- 封装分页列表加载，使用zset

## 待解决问题
- 批量获取不支持go-redis的cluster模式，因为在此模式执行mget mset，当key不属于同一个节点会报错