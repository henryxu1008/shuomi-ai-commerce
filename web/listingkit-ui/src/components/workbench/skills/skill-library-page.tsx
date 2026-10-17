"use client";

import Link from "next/link";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {ConsolePage, ConsoleState} from "../console/console-page";
import {SkillManager} from "./skill-manager";
import styles from "./skills.module.css";
import introStyles from "../data-services/data-services-overview.module.css";
import homeStyles from "../console/primary-introduction.module.css";

export type SkillCategory = "official" | "mine" | "team";
const root = "/workbench/agents/skills";
const categories = [
  {id: "official", label: "官方模板", path: `${root}/official`, summary: "发现标准方法，按许可复用", description: "查看已发布启用的标准方法，按当前版本许可复制与复用。", notes: [
    ["许可与准确版本", "复制、导出与使用取决于当前版本许可；复制后生成独立技能。"],
    ["执行边界", "技能提供方法与参考材料，执行继续使用现有智能体的权限、预算与审核。"],
  ]},
  {id: "mine", label: "我的技能", path: `${root}/mine`, summary: "整理个人方法，持续积累", description: "整理个人方法与参考材料，管理版本，需要时提交团队审核。", notes: [
    ["个人归属与版本", "个人技能归属本人，不要求加入企业；保存新版本时保留已保存正文。"],
    ["提交团队审核", "本人同意并提交指定版本后，由企业维护者审核并生成独立团队副本。"],
  ]},
  {id: "team", label: "团队技能", path: `${root}/team`, summary: "沉淀企业方法，有序共享", description: "维护当前企业可复用的业务方法，按已有权限审批与管理准入。", notes: [
    ["当前企业范围", "团队技能属于当前企业；切换企业后重新读取，访问与管理按已有权限确认。"],
    ["团队复用边界", "个人提交须经本人同意和维护者审核；组织副本与个人后续改动相互独立。"],
  ]},
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
  return <ConsolePage className={`${introStyles.page} ${homeStyles.page} ${styles.libraryPage}`} title={current.label} description={<>{current.description}{category === "team" && organization && <span className={styles.organization}>当前企业：{organization.name}</span>}</>}
    breadcrumbs={[{label: "智能市场", href: "/workbench/agents"}, {label: "技能库", href: root}, {label: current.label}]}>
    <nav className={styles.categories} aria-label="技能库分类">{categories.map((item, index) => <Link key={item.id} href={item.path} prefetch={false} aria-label={item.label} aria-current={item.id === category ? "page" : undefined}><span className={styles.categoryHeading}><span aria-hidden="true">{String(index + 1).padStart(2, "0")}</span><strong>{item.label}</strong></span><span>{item.summary}</span></Link>)}</nav>
    {content}
    <aside className={styles.notes} aria-label={`${current.label}使用说明`}>{current.notes.map(([title, description]) => <Card key={title}><h2>{title}</h2><p>{description}</p></Card>)}</aside>
  </ConsolePage>;
}
