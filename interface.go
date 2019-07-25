package model

type RedisStub interface {
	Do(commandName string, args ...interface{}) (reply interface{}, err error)
}

type Locker interface {
	Lock() error
	Unlock() error
}
