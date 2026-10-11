import { PrimaryIntroduction } from "@/components/workbench/console/primary-introduction";

export default function Page() {
  const enabled = process.env.LISTINGKIT_TOOL_MARKET_ENABLED === "true";
  return <PrimaryIntroduction kind="tools" available={[enabled, enabled, enabled]} />;
}
