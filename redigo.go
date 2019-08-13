package model

import (
	"github.com/go-redis/redis"
	redigo "github.com/gomodule/redigo/redis"
)

//CompatibleWithRedigo CompatibleWithRedigo
type CompatibleWithRedigo struct {
	Client    *redis.Client
	pipe      redis.Pipeliner
	cmds      []*redis.Cmd
	readIndex int
}

//Do Do
func (c CompatibleWithRedigo) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	cmd := c.Client.Do(redigo.Args{}.Add(commandName).Add(args...)...)
	return cmd.Result()
}

//Send Send
func (c *CompatibleWithRedigo) Send(commandName string, args ...interface{}) error {
	if c.pipe == nil {
		c.pipe = c.Client.Pipeline()
	}

	cmd := c.pipe.Do(redigo.Args{}.Add(commandName).Add(args...)...)
	c.cmds = append(c.cmds, cmd)
	return cmd.Err()
}

//Flush Flush
func (c *CompatibleWithRedigo) Flush() (err error) {
	_, err = c.pipe.Exec()
	return
}

//Receive Receive
func (c *CompatibleWithRedigo) Receive() (reply interface{}, err error) {
	reply, err = c.cmds[c.readIndex].Result()
	c.readIndex++
	return
}

//ScanStruct HGETALL：fields传空 HMGET: fields 按照顺序传入field
func ScanStruct(src []interface{}, dest interface{}, fields []string) (err error) {
	if len(src) > 0 {
		//redigo 返回值使用[]byte, go-redis 返回值使用string
		//此处统一转为[]byte
		for i := 0; i < len(src); i++ {
			if src[i] != nil {
				if _, ok := src[i].(string); ok {
					src[i] = []byte(src[i].(string))
				}
			}
		}
	}

	if fields != nil && len(fields) > 0 {
		vs := make([]interface{}, 0, len(src)*2)
		for i := 0; i < len(fields); i++ {
			if src[i] != nil {
				vs = append(vs, []byte(fields[i]), src[i])
			}
		}

		err = redigo.ScanStruct(vs, dest)
	} else {
		err = redigo.ScanStruct(src, dest)
	}
	return err
}
