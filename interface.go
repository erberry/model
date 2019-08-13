package model

type RedisStub interface {
	Do(commandName string, args ...interface{}) (reply interface{}, err error)
	Send(commandName string, args ...interface{}) error
	Flush() (err error)
	Receive() (reply interface{}, err error)
}

type Locker interface {
	Lock() error
	Unlock() error
}
