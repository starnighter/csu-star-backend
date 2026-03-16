package utils

import (
	"log"

	"github.com/bwmarrin/snowflake"
)

var (
	node *snowflake.Node
)

func InitSnowflake(nodeID int64) {
	var err error
	node, err = snowflake.NewNode(nodeID)
	if err != nil {
		log.Fatalf("Snowflake节点初始化失败：%v\n", err)
	}
}

func GenerateID() int64 {
	return node.Generate().Int64()
}
