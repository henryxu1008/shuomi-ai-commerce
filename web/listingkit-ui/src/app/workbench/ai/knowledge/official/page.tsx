import {isKnowledgeAvailable} from "@/lib/server/knowledge-availability";
import {OfficialKnowledgePage} from "@/components/workbench/knowledge/official-knowledge-page";

export default function Page() {
  return <OfficialKnowledgePage enterpriseAvailable={isKnowledgeAvailable()}/>;
}
