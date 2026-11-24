// Package qna4e implements Mitsubishi MC 4E collection.
package qna4e

import "harns-platform/pkg/gateway/collection/mc"

const (
	DriverType  = "Mitsubishi"
	DriverModel = "MC-4E"
)

type CreateReq = mc.CreateReq
type UpdateReq = mc.UpdateReq
type Collection = mc.Collection

func NewCollection(req *CreateReq) (*Collection, error) {
	return mc.NewCollection(mc.Info4E, req)
}

func NewCollectionFromConfig(raw []byte) (*Collection, error) {
	item := mc.NewEmptyCollection(mc.Info4E)
	if err := item.ApplyConfig(raw); err != nil {
		return nil, err
	}
	return item, nil
}

func NewCreateHandler() *mc.CreateHandler       { return mc.NewCreateHandler(mc.Info4E) }
func NewCollectorFactory() *mc.CollectorFactory { return mc.NewCollectorFactory(mc.Info4E) }
