package ble

import (
	"context"

	"github.com/tcslater/pigsydust"
	"tinygo.org/x/bluetooth"
)

// Scan discovers Pixie mesh devices matching the given filter.
//
// Results are delivered on the returned channel. The channel is closed
// when the context is cancelled or [Adapter.StopScan] is called.
// Duplicate advertisements from the same device are not deduplicated —
// the caller should filter by MAC if needed.
//
// Scan filters by the 16-bit service UUID 0xCDAB and manufacturer ID
// 0x0211 (Skytone). Additional filtering by mesh name, network ID,
// and gateway-only is applied according to the filter.
func (a *Adapter) Scan(ctx context.Context, filter pigsydust.ScanFilter) (<-chan ScanResult, error) {
	ch := make(chan ScanResult, 16)

	go func() {
		defer close(ch)

		errCh := make(chan error, 1)
		go func() {
			errCh <- a.bt.Scan(func(_ *bluetooth.Adapter, result bluetooth.ScanResult) {
				sr, ok := parseScanResult(result, filter)
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

// ScanResult pairs parsed advertisement data with the raw BLE address
// needed for [Adapter.Connect].
type ScanResult struct {
	Advertisement pigsydust.AdvertisementData
	Address       bluetooth.Address
	RSSI          int16
}

// StopScan stops an active BLE scan.
func (a *Adapter) StopScan() error {
	return a.bt.StopScan()
}

// parseScanResult extracts Pixie advertisement data from a raw BLE scan result,
// applying the given filter. Returns false if the result doesn't match.
func parseScanResult(result bluetooth.ScanResult, filter pigsydust.ScanFilter) (ScanResult, bool) {
	// Check for the Pixie service UUID.
	if !result.AdvertisementPayload.HasServiceUUID(ServiceUUID) {
		return ScanResult{}, false
	}

	// Check mesh name filter.
	localName := result.AdvertisementPayload.LocalName()
	if filter.MeshName != "" && localName != filter.MeshName {
		return ScanResult{}, false
	}

	// Find Skytone manufacturer data.
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

	// Parse the manufacturer data.
	adv, err := pigsydust.ParseManufacturerData(ManufacturerIDSkytone, mfgData)
	if err != nil {
		return ScanResult{}, false
	}

	adv.MeshName = localName

	// Apply network ID filter.
	if filter.NetworkID != 0 && adv.NetworkID != filter.NetworkID {
		return ScanResult{}, false
	}

	// Apply gateway-only filter.
	if filter.GatewayOnly && adv.DeviceType != pigsydust.DeviceTypeGateway {
		return ScanResult{}, false
	}

	return ScanResult{
		Advertisement: adv,
		Address:       result.Address,
		RSSI:          result.RSSI,
	}, true
}
