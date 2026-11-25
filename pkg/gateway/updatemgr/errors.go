package updatemgr

import "errors"

var (
	errUpdateDisabled = errors.New("gateway update is disabled")
	errUpdateNotFound = errors.New("gateway update task not found")
	errUpdateBusy     = errors.New("gateway update task is not ready")
)
