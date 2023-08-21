package storage

import (
	"encoding/json"

	"google.golang.org/protobuf/proto"
)

func (cdc *Codec) Encode(obj interface{}) ([]byte, error) {
	if t, ok := obj.(proto.Message); ok {
		return cdc.Encode(t)
	}
	return json.Marshal(obj)
}

func (cdc *Codec) Decode(bz []byte, obj interface{}) error {
	if t, ok := obj.(proto.Message); ok {
		return cdc.Decode(bz, t)
	}
	return json.Unmarshal(bz, obj)
}

type Codec struct {
	opts proto.MarshalOptions
}

func NewCodec() *Codec {
	return &Codec{
		opts: proto.MarshalOptions{
			Deterministic: true,
		},
	}
}
