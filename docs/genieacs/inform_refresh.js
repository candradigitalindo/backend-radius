var now = Date.now();
var params = [
  "InternetGatewayDevice.DeviceInfo.SoftwareVersion",
  "InternetGatewayDevice.DeviceInfo.UpTime",
  "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID",
  "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.BeaconType",
  "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase",
  "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase",
  "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.PreSharedKey",
  "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_CT-COM_WPSKeyWord",
  "InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID",
  "InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase",
  "InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase",
  "InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.PreSharedKey",
  "InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.X_CT-COM_WPSKeyWord",
  "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.ExternalIPAddress",
  "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Username",
  "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPONInterfaceConfig.RXPower",
  "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPONInterfaceConfig.TXPower",
  "InternetGatewayDevice.LANDevice.1.Hosts.HostNumberOfEntries",
  "InternetGatewayDevice.ManagementServer.PeriodicInformEnable",
  "InternetGatewayDevice.ManagementServer.PeriodicInformInterval"
];

for (var i = 0; i < params.length; i++) {
  declare(params[i], { value: now });
}

for (var wlanPasswordIndex = 1; wlanPasswordIndex <= 2; wlanPasswordIndex++) {
  for (var preSharedIndex = 1; preSharedIndex <= 10; preSharedIndex++) {
    declare(
      "InternetGatewayDevice.LANDevice.1.WLANConfiguration." + wlanPasswordIndex + ".PreSharedKey." + preSharedIndex + ".PreSharedKey",
      { value: now }
    );
    declare(
      "InternetGatewayDevice.LANDevice.1.WLANConfiguration." + wlanPasswordIndex + ".PreSharedKey." + preSharedIndex + ".KeyPassphrase",
      { value: now }
    );
  }
  for (var wepIndex = 1; wepIndex <= 4; wepIndex++) {
    declare(
      "InternetGatewayDevice.LANDevice.1.WLANConfiguration." + wlanPasswordIndex + ".WEPKey." + wepIndex + ".WEPKey",
      { value: now }
    );
  }
}

var hostProps = ["IPAddress", "MACAddress", "HostName", "Active", "InterfaceType"];
for (var hostIndex = 1; hostIndex <= 16; hostIndex++) {
  for (var hostPropIndex = 0; hostPropIndex < hostProps.length; hostPropIndex++) {
    declare(
      "InternetGatewayDevice.LANDevice.1.Hosts.Host." + hostIndex + "." + hostProps[hostPropIndex],
      { value: now }
    );
  }
}

var assocProps = [
  "AssociatedDeviceIPAddress",
  "AssociatedDeviceMACAddress",
  "AssociatedDeviceAuthenticationState"
];
for (var wlanIndex = 1; wlanIndex <= 2; wlanIndex++) {
  declare("InternetGatewayDevice.LANDevice.1.WLANConfiguration." + wlanIndex + ".AssociatedDevice.", { path: now, value: now });
  for (var assocIndex = 1; assocIndex <= 16; assocIndex++) {
    for (var assocPropIndex = 0; assocPropIndex < assocProps.length; assocPropIndex++) {
      declare(
        "InternetGatewayDevice.LANDevice.1.WLANConfiguration." + wlanIndex + ".AssociatedDevice." + assocIndex + "." + assocProps[assocPropIndex],
        { value: now }
      );
    }
  }
}
