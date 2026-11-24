// Package qna3e implements Mitsubishi MC 3E collection.
package qna3e

import "harns-platform/pkg/gateway/collection/mc"

const (
	DriverType  = "Mitsubishi"
	DriverModel = "MC-3E"
)

type CreateReq = mc.CreateReq
type UpdateReq = mc.UpdateReq
type Collection = mc.Collection

func NewCollection(req *CreateReq) (*Collection, error) {
	return mc.NewCollection(mc.Info3E, req)
}

func NewCollectionFromConfig(raw []byte) (*Collection, error) {
	item := mc.NewEmptyCollection(mc.Info3E)
	if err := item.ApplyConfig(raw); err != nil {
		return nil, err
	}
	return item, nil
}

func NewCreateHandler() *mc.CreateHandler       { return mc.NewCreateHandler(mc.Info3E) }
func NewCollectorFactory() *mc.CollectorFactory { return mc.NewCollectorFactory(mc.Info3E) }
