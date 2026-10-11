import type { AIWorkbenchEntryCapabilities } from "./ai-workbench-entry";

export type ConsolePrimaryCapabilities = AIWorkbenchEntryCapabilities & {
  operationsCockpitAvailable?: boolean;
  cockpitPermissions?: readonly string[];
  productAcquisitionAvailable?: boolean;
  supplyMarketAvailable?: boolean;
  supplyChainAvailable?: boolean;
  supplyPreparationAvailable?: boolean;
  dataServicesAvailable?: boolean;
  productCollectionsAvailable?: boolean;
  toolMarketAvailable?: boolean;
  ecoservicesAvailable?: boolean;
};

// An independent homepage is available separately from its child business services.
// Returning the homepage never authorizes its owner or enables a child capability.
export function firstConnectedConsoleEntry(pathname: string, capabilities: ConsolePrimaryCapabilities): string | null {
  switch (pathname) {
    case "/workbench":
      return "/workbench";
    case "/workbench/ai": return "/workbench/ai";
    case "/workbench/supply": return "/workbench/supply";
    case "/workbench/agents": return "/workbench/agents";
    case "/workbench/tools": return "/workbench/tools";
    case "/workbench/services": return capabilities.ecoservicesAvailable ? "/workbench/services" : null;
    case "/workbench/data":
      return "/workbench/data";
    case "/workbench/store-center": return "/workbench/store-center";
    case "/workbench/plans": return "/workbench/plans";
    case "/workbench/account": return "/workbench/account";
    default: return null;
  }
}
