package mq

type Header map[string]string

type Message struct {
	// 消息本体，存储业务消息
	Value []byte
	// 对标kafka中的key，用于分区的。可以省略
	Key []byte
	// 对标kafka的header，用于传递一些自定义的元数据
	Hander Header
	// 消息主题
	Topic string
	// 分区ID
	Partition int64
	// 偏移量
	Offset int64
}
