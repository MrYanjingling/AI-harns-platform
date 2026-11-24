// Package a1e implements Mitsubishi MC 1E (A1E) collection.
package a1e

import "harns-platform/pkg/gateway/collection/mc"

const (
	DriverType  = "Mitsubishi"
	DriverModel = "MC-1E"
)

type CreateReq = mc.CreateReq
type UpdateReq = mc.UpdateReq
type Collection = mc.Collection

func NewCollection(req *CreateReq) (*Collection, error) {
	return mc.NewCollection(mc.InfoA1E, req)
}

func NewCollectionFromConfig(raw []byte) (*Collection, error) {
	item := mc.NewEmptyCollection(mc.InfoA1E)
	if err := item.ApplyConfig(raw); err != nil {
		return nil, err
	}
	return item, nil
}

func NewCreateHandler() *mc.CreateHandler       { return mc.NewCreateHandler(mc.InfoA1E) }
func NewCollectorFactory() *mc.CollectorFactory { return mc.NewCollectorFactory(mc.InfoA1E) }
