// Provision "inform_refresh" — refresh parameter perangkat tiap Inform.
//
// CATATAN PERBAIKAN (2026-06-24):
// Versi lama membuat channel "default" FAULT dengan "Invalid parameter path"
// karena baris:
//   declare("...WLANConfiguration.N.AssociatedDevice.", { path: now, value: now });
// meminta `value` pada node OBJEK (path berakhiran titik) — itu invalid, dan
// sekali channel fault GenieACS berhenti refresh (inform cuma balas 204).
//
// Versi ini memakai idiom wildcard (*) GenieACS sehingga instance ditemukan
// otomatis dan hanya leaf yang diminta value-nya. Tidak ada fault.
const now = Date.now();

// Parameter leaf tunggal
const params = [
  "InternetGatewayDevice.DeviceInfo.SoftwareVersion",
  "InternetGatewayDevice.DeviceInfo.HardwareVersion",
  "InternetGatewayDevice.DeviceInfo.UpTime",
  "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.ExternalIPAddress",
  "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Username",
  "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPONInterfaceConfig.RXPower",
  "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPONInterfaceConfig.TXPower",
  "InternetGatewayDevice.LANDevice.1.Hosts.HostNumberOfEntries",
  "InternetGatewayDevice.ManagementServer.PeriodicInformEnable",
  "InternetGatewayDevice.ManagementServer.PeriodicInformInterval",
];
for (let i = 0; i < params.length; i++) declare(params[i], { value: now });

// WLAN (SSID + sandi), wildcard atas instance WLANConfiguration & PreSharedKey
declare("InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.SSID", { value: now });
declare("InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.BeaconType", { value: now });
declare("InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.KeyPassphrase", { value: now });
declare("InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.X_CT-COM_WPSKeyWord", { value: now });
declare("InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.PreSharedKey.*.KeyPassphrase", { value: now });
declare("InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.PreSharedKey.*.PreSharedKey", { value: now });

// Host LAN yang terhubung
declare("InternetGatewayDevice.LANDevice.1.Hosts.Host.*.IPAddress", { value: now });
declare("InternetGatewayDevice.LANDevice.1.Hosts.Host.*.MACAddress", { value: now });
declare("InternetGatewayDevice.LANDevice.1.Hosts.Host.*.HostName", { value: now });
declare("InternetGatewayDevice.LANDevice.1.Hosts.Host.*.Active", { value: now });
declare("InternetGatewayDevice.LANDevice.1.Hosts.Host.*.InterfaceType", { value: now });

// Perangkat wireless yang terasosiasi
declare("InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.AssociatedDevice.*.AssociatedDeviceMACAddress", { value: now });
declare("InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.AssociatedDevice.*.AssociatedDeviceIPAddress", { value: now });
declare("InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.AssociatedDevice.*.AssociatedDeviceAuthenticationState", { value: now });
