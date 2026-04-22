//go:build !linux

package ble

import (
	"context"

	"github.com/tcslater/pigsydust"
	"tinygo.org/x/bluetooth"
)

// scan delivers parsed ScanResults on the returned channel until ctx is
// cancelled. Duplicate advertisements aren't deduped — callers filter by MAC.
func (a *tinygoAdapter) scan(ctx context.Context, filter pigsydust.ScanFilter) (<-chan ScanResult, error) {
	ch := make(chan ScanResult, 16)

	go func() {
		defer close(ch)

		errCh := make(chan error, 1)
		go func() {
			errCh <- a.bt.Scan(func(_ *bluetooth.Adapter, result bluetooth.ScanResult) {
				sr, ok := parseTinygoScanResult(result, filter)
				if !ok {
					return
				}
				select {
				case ch <- sr:
				case <-ctx.Done():
				}
			})
		}()

		select {
		case <-ctx.Done():
			a.bt.StopScan()
		case <-errCh:
		}
	}()

	return ch, nil
}

func (a *tinygoAdapter) stopScan() error { return a.bt.StopScan() }

// parseTinygoScanResult extracts Pixie advertisement data from a raw tinygo
// scan result, applying the given filter. Returns false if no match.
func parseTinygoScanResult(result bluetooth.ScanResult, filter pigsydust.ScanFilter) (ScanResult, bool) {
	if !result.AdvertisementPayload.HasServiceUUID(toTinygoUUID(ServiceUUID)) {
		return ScanResult{}, false
	}

	localName := result.AdvertisementPayload.LocalName()
	if filter.MeshName != "" && localName != filter.MeshName {
		return ScanResult{}, false
	}

	var mfgData []byte
	for _, md := range result.AdvertisementPayload.ManufacturerData() {
		if md.CompanyID == ManufacturerIDSkytone {
			mfgData = md.Data
			break
		}
	}
	if mfgData == nil {
		return ScanResult{}, false
	}

	adv, err := pigsydust.ParseManufacturerData(ManufacturerIDSkytone, mfgData)
	if err != nil {
		return ScanResult{}, false
	}

	adv.MeshName = localName

	if filter.NetworkID != 0 && adv.NetworkID != filter.NetworkID {
		return ScanResult{}, false
	}
	if filter.GatewayOnly && adv.DeviceType != pigsydust.DeviceTypeGateway {
		return ScanResult{}, false
	}

	return ScanResult{
		Advertisement: adv,
		Address:       tinygoAddress{a: result.Address},
		RSSI:          result.RSSI,
	}, true
}
