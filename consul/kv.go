package consul

import (
	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"

	"github.com/mia0x75/copycat/g"
)

// Kv TODO
type Kv struct {
	kv *api.KV
}

// IKv TODO
type IKv interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

// NewKv TODO
func NewKv(kv *api.KV) IKv {
	return &Kv{kv: kv}
}

// Set set key value
func (k *Kv) Set(key string, value []byte) error {
	log.Debugf("[D] write %s=%s", key, value)
	kv := &api.KVPair{
		Key:   key,
		Value: value,
	}
	_, err := k.kv.Put(kv, nil)
	return err
}

// Get get key value
// if key does not exists, return error:KvDoesNotExists
func (k *Kv) Get(key string) ([]byte, error) {
	kv, m, err := k.kv.Get(key, nil)
	log.Infof("[I] kv == %+v,", kv)
	log.Infof("[I] m == %+v,", m)
	if err != nil {
		log.Errorf("[E] %+s", err.Error())
		return nil, err
	}
	if kv == nil {
		return nil, g.ErrKvDoesNotExists
	}
	return kv.Value, nil
}

// Delete delete key value
func (k *Kv) Delete(key string) error {
	_, err := k.kv.Delete(key, nil)
	if err != nil {
		log.Errorf("[E] %s", err.Error())
	}
	return err
}
