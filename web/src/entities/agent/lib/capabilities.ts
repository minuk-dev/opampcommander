// Decode an OpAMP AgentCapabilities bitmask into the individual capability
// names. Shared by the agents list and detail pages so the bit table lives in
// one place.
export function capabilityNames(bitmask: number | undefined): string[] {
  if (!bitmask) return [];
  const table: Array<[number, string]> = [
    [1, 'ReportsStatus'],
    [2, 'AcceptsRemoteConfig'],
    [4, 'ReportsEffectiveConfig'],
    [8, 'AcceptsPackages'],
    [16, 'ReportsPackageStatuses'],
    [32, 'ReportsOwnTraces'],
    [64, 'ReportsOwnMetrics'],
    [128, 'ReportsOwnLogs'],
    [256, 'AcceptsOpAMPConnectionSettings'],
    [512, 'AcceptsOtherConnectionSettings'],
    [1024, 'AcceptsRestartCommand'],
    [2048, 'ReportsHealth'],
    [4096, 'ReportsRemoteConfig'],
    [8192, 'ReportsHeartbeat'],
    [16384, 'ReportsAvailableComponents'],
  ];
  return table.filter(([b]) => (bitmask & b) !== 0).map(([, name]) => name);
}
