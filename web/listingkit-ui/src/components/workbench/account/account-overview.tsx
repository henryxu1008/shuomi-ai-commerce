"use client";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import type { AccountBusinessProfile, AccountOrganization, AccountProfile } from "@/lib/api/account";
import { AccountReadError } from "@/lib/api/account";
import { getReferralEarnings } from "@/lib/api/account-referral-economics";
import { formatRetailMoney } from "@/lib/api/retail-prices";
import styles from "../console/primary-overview.module.css";

export function AccountOverview({ profile, business, organization }: { profile: AccountProfile; business: AccountBusinessProfile | null; organization: AccountOrganization }) {
  const verified = profile.emailVerified === true || profile.phoneNumberVerified === true;
  const identity = organization.roles.length ? organization.roles.join("、") : "未提供角色";
  const earnings = useQuery({ queryKey: ["account", "overview-earnings", profile.userId], queryFn: async ({ signal }) => {
    const result = await getReferralEarnings(profile.userId, AbortSignal.any([signal, AbortSignal.timeout(15000)]));
    if (result.referrer !== profile.userId) throw new AccountReadError(502, "INVALID_UPSTREAM_RESPONSE");
    return result;
  }, retry: false, staleTime: 0, gcTime: 0 });
  const earned = !earnings.isFetching && !earnings.error ? earnings.data : undefined;
  return <div className={styles.stack}>
    <Card className={styles.panel}><h2>账户概览</h2><p className={styles.description}>查看当前账户、企业与收益状态</p><div className={styles.metrics}><div data-tone="teal"><span>账号状态</span><strong>{verified ? "已验证" : "待完善"}</strong><small>{profile.emailVerified === true && profile.phoneNumberVerified === true ? "手机 / 邮箱已验证" : "验证状态以登录服务为准"}</small></div><div data-tone="blue"><span>企业状态</span><strong>当前授权有效</strong><small>{identity}</small></div><div data-tone="purple"><span>推广收益</span><strong>{earned ? `可用 ${formatRetailMoney(earned.availableMinor)}` : "—"}</strong><small>{earned ? `待结算 ${formatRetailMoney(earned.pendingMinor)}` : earnings.isFetching ? "正在读取个人收益…" : "收益读取暂不可用，请进入收益页重试"}</small></div></div></Card>
    <header className={styles.sectionHeading}><h2>账户管理</h2><p>统一管理账户资料、企业协作与推广收益。</p></header>
    <div className={styles.entriesThree}>
      <Card className={styles.entry} data-tone="teal"><h3>账户资料</h3><p className={styles.entrySubtitle}>{profile.displayName?.trim() || "当前账户"}</p><p className={styles.description}>维护显示名称、手机、邮箱、经营画像与身份认证信息。</p><div className={styles.entryFact}><span>经营画像</span><strong>{business ? "已保存" : "尚未取得"}</strong></div><Button asChild size="sm"><Link href="/workbench/account/profile" prefetch={false}>查看账户资料</Link></Button></Card>
      <Card className={styles.entry} data-tone="blue"><h3>企业空间</h3><p className={styles.entrySubtitle}>{organization.name?.trim() || "当前有效企业"}</p><p className={styles.description}>管理成员与权限、资源与额度以及操作记录。</p><div className={styles.entryFact}><span>当前身份</span><strong>{identity}</strong></div><Button asChild size="sm"><Link href="/workbench/account/organization" prefetch={false}>查看企业空间</Link></Button></Card>
      <Card className={styles.entry} data-tone="purple"><h3>推广与收益</h3><p className={styles.entrySubtitle}>推广关系、收益账本与人工提现</p><p className={styles.description}>管理推广入口、收益明细、提现与推广规则。</p><div className={styles.entryFact}><span>可用收益</span><strong>{earned ? formatRetailMoney(earned.availableMinor) : "—"}</strong></div><Button asChild size="sm"><Link href="/workbench/account/referrals" prefetch={false}>查看推广信息</Link></Button></Card>
    </div>
    <div className={styles.columns}><Card className={styles.panel}><h2>最近动态</h2><p className={styles.description}>账户相关的重要变化</p><p className={styles.empty}>账户动态汇总暂未接入；企业操作记录可在企业空间查看。</p><Link className={styles.textLink} href="/workbench/account/organization/audit" prefetch={false}>查看企业操作记录 →</Link></Card><Card className={styles.panel}><h2>待关注</h2><p className={styles.description}>需要留意的账户事项</p><div className={styles.notices}><div><div><strong>联系方式</strong><p>{profile.emailVerified === true && profile.phoneNumberVerified === true ? "手机与邮箱均已验证" : "可在账户资料确认手机与邮箱验证状态"}</p></div><Link href="/workbench/account/profile/verification" prefetch={false}>查看</Link></div><div><div><strong>推广收益</strong><p>{earned ? `${formatRetailMoney(earned.pendingMinor)} 待结算；提现须满足实名及审核要求` : "待结算金额以收益页实时账本为准"}</p></div><Link href="/workbench/account/referrals" prefetch={false}>查看</Link></div></div></Card></div>
    <details className={styles.description}><summary>资料来源与读取时间</summary><p>身份资料：ZITADEL 账户资料；企业角色：ZITADEL 项目授权。</p><p>读取时间：<time dateTime={profile.readAt}>{profile.readAt}</time></p></details>
  </div>;
}
