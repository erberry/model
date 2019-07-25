package model

import (
	"github.com/go-redis/redis"
)

//GoRedisClient GoRedisClient
type GoRedisClient interface {
	Do(args ...interface{}) *redis.Cmd
}

//Compatible Compatible
type Compatible struct {
	Client GoRedisClient
}

//Do Do
func (c Compatible) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	aa := make([]interface{}, 0, len(args)+1)
	aa = append(aa, commandName)
	for _, arg := range args {
		aa = append(aa, arg)
	}
	cmd := c.Client.Do(aa...)
	return cmd.Result()
}
