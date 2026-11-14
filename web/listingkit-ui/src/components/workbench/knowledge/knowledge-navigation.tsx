import Link from "next/link";
import "./knowledge.css";
import {WorkbenchViewNav} from "../console/workbench-view-nav";

export function KnowledgeNavigation({current, enterpriseAvailable}: {current: "home" | "official" | "mine"; enterpriseAvailable: boolean}) {
 return <WorkbenchViewNav className="knowledge-categories" label="知识库分类">
  <Link href="/workbench/ai/knowledge" prefetch={false} aria-current={current === "home" ? "page" : undefined}>知识库首页</Link>
  <Link href="/workbench/ai/knowledge/official" prefetch={false} aria-label="切换到官方知识库" aria-current={current === "official" ? "page" : undefined}>官方知识库</Link>
  {enterpriseAvailable ? <Link href="/workbench/ai/knowledge/mine" prefetch={false} aria-label="切换到我的知识库" aria-current={current === "mine" ? "page" : undefined}>我的知识库</Link> : <span aria-disabled="true">我的知识库暂未开放</span>}
 </WorkbenchViewNav>;
}
