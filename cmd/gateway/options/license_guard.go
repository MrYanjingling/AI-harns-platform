package options

import (
	"context"
	"time"

	"harns-platform/pkg/gateway/collectionmgr"
	"harns-platform/pkg/gateway/licensemgr"
	"harns-platform/pkg/gateway/pushmgr"
	"harns-platform/pkg/logging"
)

const licenseGuardMinute = 5

func startLicenseRunGuard(stopCh <-chan struct{}, license *licensemgr.Service, collections *collectionmgr.Service, push *pushmgr.Service) {
	if license == nil {
		return
	}
	go func() {
		timer := time.NewTimer(time.Until(nextLicenseGuardTime(time.Now())))
		defer timer.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-timer.C:
				checkLicenseRunDeadline(license, collections, push)
				timer.Reset(time.Until(nextLicenseGuardTime(time.Now())))
			}
		}
	}()
}

func nextLicenseGuardTime(now time.Time) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), licenseGuardMinute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(time.Hour)
	}
	return next
}

func checkLicenseRunDeadline(license *licensemgr.Service, collections *collectionmgr.Service, push *pushmgr.Service) {
	deadline, ok := license.SoftwareRunDeadline()
	if !ok {
		logging.Warnf("License software run deadline is unavailable")
		return
	}
	now := time.Now()
	if !now.After(deadline) {
		return
	}
	logging.Warnf("License software run deadline reached: deadline=%s now=%s", deadline.Format(time.RFC3339), now.Format(time.RFC3339))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if collections != nil {
		if err := collections.StopAll(ctx); err != nil {
			logging.ErrorS(err, "Failed to stop all collections after license deadline")
		}
	}
	if push != nil {
		if err := push.StopAll(ctx); err != nil {
			logging.ErrorS(err, "Failed to stop all push connectors after license deadline")
		}
	}
}
