# smoke-linux

Linux-only BLE smoke test for pigsydust/ble against BlueZ.

Confirms the end-to-end path works on a real BlueZ stack:

1. Scan for the named mesh via BlueZ D-Bus (SetDiscoveryFilter + InterfacesAdded).
2. Connect, wait for ServicesResolved, discover the three Pixie characteristics.
3. Log in over CHAR_PAIR (Write Request + Read).
4. Subscribe to CHAR_NOTIFY. StartNotify is expected to fail ATT 0x0e on Telink
   firmware; the 0x01 Write Request primes the firmware regardless, and
   notifications arrive via PropertiesChanged on the characteristic's Value.
5. Send a SetUTC command over CHAR_CMD (Write Command / without response).
6. Observe notifications for a short window.

## Usage

On the target Linux host:

```
GOOS=linux GOARCH=amd64 go build -o /tmp/smoke-linux ./cmd/smoke-linux
scp /tmp/smoke-linux host:~
ssh host 'PIGSY_MESH_PASSWORD=... ./smoke-linux'
```

Or build natively on the target:

```
go build -o smoke-linux ./cmd/smoke-linux
PIGSY_MESH_PASSWORD=... ./smoke-linux
```

Requires `bluetoothd` running and no other client holding the adapter
(e.g. stop Home Assistant's `sal_pixie` integration first if you want a
clean run — otherwise contention can show up as transient Connect failures).
