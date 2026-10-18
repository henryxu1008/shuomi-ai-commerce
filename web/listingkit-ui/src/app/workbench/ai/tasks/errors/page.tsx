import {connection} from "next/server";
import {configuredAgentConfigurationOrigin} from "@/lib/server/agent-configuration-origin";
import { BusinessTaskPage } from "@/components/workbench/ai-workbench/task-page";
export default async function Page() { await connection(); return <BusinessTaskPage nativeRunsAvailable={configuredAgentConfigurationOrigin() !== null} mode="errors" />; }
