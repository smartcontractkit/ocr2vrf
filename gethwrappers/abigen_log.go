package gethwrappers

import (
	"github.com/ethereum/go-ethereum/common"
)

type AbigenLog interface {
	Topic() common.Hash
}
