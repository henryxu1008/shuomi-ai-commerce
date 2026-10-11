import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { findConsoleRoute } from "@/lib/workbench/console-navigation";
import { ConsolePage, ConsoleState } from "./console-page";

export function ConsoleUnavailable({ pathname }: { pathname: string }) {
  const route = findConsoleRoute(pathname)!;
  const isChat = pathname === "/workbench/ai/chat";
  return <ConsolePage className={isChat ? "console-chat" : undefined} title={route.node.label} breadcrumbs={route.trail} description={isChat ? "围绕会话创建、持续沟通和重要内容沉淀，管理你的AI业务对话。" : "当前业务尚未接入。"}>
    <ConsoleState kind="unavailable" title="暂未启用">该页面提供结构预览，未连接业务服务；不会生成任务、发送会话或修改数据。</ConsoleState>
    {isChat ? <div className="console-chat-grid">{[
      ["新建会话", "开始一项新需求", "创建独立的业务会话，并选择相关项目或店铺作为背景。"],
      ["最近会话", "继续之前的工作", "按最近使用时间集中展示历史会话，并保留完整沟通上下文。"],
      ["收藏会话", "沉淀重要内容", "收藏具有长期参考价值的会话，将关键结论、分析过程和业务方案集中保存。"],
    ].map(([title, subtitle, description]) => <Card key={title} className="console-chat-card"><h2>{title}</h2><p className="console-chat-subtitle">{subtitle}</p><p className="console-description">{description}</p><Button disabled variant="outline">暂未启用</Button></Card>)}</div> : null}
  </ConsolePage>;
}
