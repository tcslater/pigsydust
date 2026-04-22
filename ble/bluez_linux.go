//go:build linux

package ble

// BlueZ D-Bus constants shared across the Linux BLE code path.
const (
	bluezBusName        = "org.bluez"
	bluezRootPath       = "/"
	bluezAdapterIface   = "org.bluez.Adapter1"
	bluezDeviceIface    = "org.bluez.Device1"
	bluezGattSvcIface   = "org.bluez.GattService1"
	bluezGattCharIface  = "org.bluez.GattCharacteristic1"
	dbusObjectManager   = "org.freedesktop.DBus.ObjectManager"
	dbusProperties      = "org.freedesktop.DBus.Properties"
	propPropertiesChang = dbusProperties + ".PropertiesChanged"
	propInterfacesAdded = dbusObjectManager + ".InterfacesAdded"
)
