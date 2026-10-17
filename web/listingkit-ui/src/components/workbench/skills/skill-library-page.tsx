"use client";

import Link from "next/link";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {ConsolePage, ConsoleState} from "../console/console-page";
import {SkillManager} from "./skill-manager";
import styles from "./skills.module.css";

export type SkillCategory = "official" | "mine" | "team";
const root = "/workbench/agents/skills";
const categories = [
  {id: "official", label: "官方模板", path: `${root}/official`},
  {id: "mine", label: "我的技能", path: `${root}/mine`},
  {id: "team", label: "团队技能", path: `${root}/team`},
] as const;

export function SkillLibraryPage({expectedUserId, category}: {expectedUserId: string; category: SkillCategory}) {
  const ctx = useWorkbenchContext();
  const current = categories.find(item => item.id === category)!;
  const authenticationFailed = [ctx.error, ctx.blockingError].some(error => error?.code === "AUTHENTICATION_REQUIRED");
  const identityChanged = !!ctx.user && ctx.user.id !== expectedUserId;
  const organizationReady = !ctx.error && !ctx.blockingError && !ctx.isLoading && !ctx.isSwitching && !ctx.selectionRequired && !!ctx.user && !!ctx.effectiveOrganization;
  const organization = organizationReady ? ctx.effectiveOrganization : null;
  let content;
  if (authenticationFailed || identityChanged) {
    content = <ConsoleState kind="error" title={identityChanged ? "登录身份已变化" : "登录已失效"}><Button asChild variant="outline"><Link href={`/login?returnTo=${encodeURIComponent(current.path)}`} prefetch={false}>重新登录</Link></Button></ConsoleState>;
  } else if (ctx.isSwitching || category === "team" && ctx.isLoading) {
    content = <ConsoleState kind="loading" title="正在确认当前企业"/>;
  } else if (category === "team" && (ctx.error || ctx.blockingError)) {
    content = <ConsoleState kind="error" title="企业上下文不可用"><Button onClick={() => void ctx.retry()}>重新确认</Button></ConsoleState>;
  } else if (category === "team" && !organization) {
    content = <ConsoleState kind="unavailable" title="请先选择企业">团队技能属于当前企业，选择企业后按已有权限管理。</ConsoleState>;
  } else {
    content = <SkillManager key={JSON.stringify([expectedUserId, ctx.user?.id, organization?.id, ctx.permissions, category])}
      scope={{userId: expectedUserId, organizationId: organization?.id}}
      home={category === "team" ? "organization" : "personal"}
      initialSpace={category === "official" ? "official" : category === "team" ? "organization" : "personal"}
      canConfigure={organizationReady && ctx.permissions.includes("workbench.agent.configure")}
      presentation="page"/>;
  }
  return <ConsolePage title="技能库" description={category === "team" && organization ? `管理可复用的业务方法 · 当前企业：${organization.name}` : "查看官方模板，管理个人和团队技能，在业务操作中选择使用。"}
    breadcrumbs={[{label: "智能市场", href: "/workbench/agents"}, {label: "技能库", href: root}, {label: current.label}]}>
    <nav className={styles.categories} aria-label="技能库分类">{categories.map(item => <Link key={item.id} href={item.path} prefetch={false} aria-current={item.id === category ? "page" : undefined}>{item.label}</Link>)}</nav>
    {content}
  </ConsolePage>;
}
