"use client";

import Image from "next/image";
import Link from "next/link";
import { ArrowRight, Menu } from "lucide-react";
import styles from "./marketing-hero.module.css";

const NAV_ITEMS = [
  ["首页", "#home"], ["电商智能体", "#agents"], ["供应链与数据", "#supply-chain"],
  ["解决方案", "#solutions"], ["服务生态", "#services"], ["应用实践", "#practices"], ["价格与服务", "#pricing"],
] as const;
const NODE_ASSETS = {"imgNode": {"src": "/sumi/figma-home-hero-imgNode.svg", "width": 27, "height": 27}, "imgNode1": {"src": "/sumi/figma-home-hero-imgNode1.svg", "width": 27, "height": 27}, "imgNode2": {"src": "/sumi/figma-home-hero-imgNode2.svg", "width": 27, "height": 27}, "imgNode3": {"src": "/sumi/figma-home-hero-imgNode3.svg", "width": 27, "height": 27}};
const CAPABILITIES = [
  ["293:232", "01", "全球商品数据", "洞察全球需求，发现潜力商品", "#data", "imgNode1"],
  ["293:253", "02", "供应链货盘", "匹配优质货盘，评估供应与交付", "#supply-chain", "imgNode2"],
  ["293:211", "03", "AI智能体", "理解业务目标，协同完成执行", "#agents", "imgNode"],
  ["293:274", "04", "生态服务", "接入专业资源，让方案快速落地", "#services", "imgNode3"],
] as const;

export function MarketingHero({ loginHref }: { loginHref: string }) {
  return <>
    <header className={styles.header} data-node-id="36:3"><div className={styles.headerInner}>
      <a aria-label="硕米智能引擎首页" className={styles.brand} href="#home">
        <Image alt="" height={44} width={44} preload src="/sumi/c3ea4c6f-992f-4c3d-ae2e-c14d4ec9735a.png" /><span>硕米智能引擎</span>
      </a>
      <nav aria-label="官网导航" className={styles.nav}>{NAV_ITEMS.map(([label,href]) => <a href={href} key={href}>{label}</a>)}</nav>
      <div className={styles.headerActions}><Link className={styles.navCta} href={loginHref}>进入硕米</Link>
        <details className={styles.mobileMenu} onClick={event => { if (event.target instanceof HTMLAnchorElement) event.currentTarget.open = false; }}>
          <summary aria-label="展开官网导航"><Menu size={20} /></summary>
          <nav aria-label="移动端官网导航">{NAV_ITEMS.map(([label,href]) => <a href={href} key={href}>{label}</a>)}</nav>
        </details>
      </div>
    </div></header>
    <section className={styles.hero} data-node-id="37:2" id="home">
      <div className={styles.heroInner}>
        <div className={styles.heroCopy} data-node-id="37:3">
          <p className={styles.eyebrow}>NEXT-GENERATION AI COMMERCE SYSTEM / 新一代 AI 电商智能操作系统</p>
          <h1 className={styles.title}><span>硕米智能引擎</span><span>新一代AI电商</span><span>智能操作系统</span></h1>
          <p className={styles.description}>从全球商品数据出发，洞察机会、匹配供应链，<br />再由 AI 智能体与专业服务协同完成业务。</p>
          <div className={styles.actions}><a className={styles.primaryAction} href="#agents">了解平台能力 <ArrowRight size={16} aria-hidden="true" /></a><a className={styles.secondaryAction} href="#solutions">查看解决方案</a></div>
          <p className={styles.capabilitySummary}>全球商品数据 · 供应链货盘 · AI智能体 · 生态服务</p>
        </div>
        <div className={styles.network} role="group" aria-label="全球商业智能网络能力示意">
          <svg className={styles.orbitField} viewBox="0 0 676 654" aria-hidden="true">
            <g className={`${styles.orbitDots} ${styles.orbitInner}`}>
              <circle className={styles.orbitBlue} cx="338" cy="169" r="3" /><circle className={styles.orbitCyan} cx="437" cy="210" r="2.5" />
              <circle className={styles.orbitBlue} cx="478" cy="309" r="3.5" /><circle className={styles.orbitCyan} cx="437" cy="408" r="2.5" />
              <circle className={styles.orbitBlue} cx="338" cy="449" r="3" /><circle className={styles.orbitCyan} cx="239" cy="408" r="3.5" />
              <circle className={styles.orbitBlue} cx="198" cy="309" r="2.5" /><circle className={styles.orbitCyan} cx="239" cy="210" r="3" />
            </g>
            <g className={`${styles.orbitDots} ${styles.orbitMiddle}`}>
              <circle className={styles.orbitCyan} cx="338" cy="62" r="3.5" /><circle className={styles.orbitBlue} cx="462" cy="95" r="2.5" />
              <circle className={styles.orbitCyan} cx="513" cy="133" r="3" /><circle className={styles.orbitBlue} cx="585" cy="309" r="3.5" />
              <circle className={styles.orbitCyan} cx="513" cy="485" r="2.5" /><circle className={styles.orbitBlue} cx="462" cy="523" r="3.5" />
              <circle className={styles.orbitCyan} cx="338" cy="556" r="3" /><circle className={styles.orbitBlue} cx="214" cy="523" r="2.5" />
              <circle className={styles.orbitCyan} cx="163" cy="485" r="3.5" /><circle className={styles.orbitBlue} cx="91" cy="309" r="2.5" />
              <circle className={styles.orbitCyan} cx="163" cy="133" r="3" /><circle className={styles.orbitBlue} cx="214" cy="95" r="3.5" />
            </g>
            <g className={`${styles.orbitDots} ${styles.orbitOuter}`}>
              <circle className={styles.orbitBlue} cx="338" cy="-11" r="3" /><circle className={styles.orbitCyan} cx="564" cy="83" r="3.5" />
              <circle className={styles.orbitBlue} cx="658" cy="309" r="2.5" /><circle className={styles.orbitCyan} cx="564" cy="535" r="3" />
              <circle className={styles.orbitBlue} cx="338" cy="629" r="3.5" /><circle className={styles.orbitCyan} cx="112" cy="535" r="2.5" />
              <circle className={styles.orbitBlue} cx="18" cy="309" r="3" /><circle className={styles.orbitCyan} cx="112" cy="83" r="3.5" />
            </g>
          </svg>
          <div className={styles.coreAnchor}><div className={styles.core}><small>硕米智能引擎</small><strong>全球商业智能网络</strong><span>四大能力协同</span></div></div>
          <ul className={styles.capabilities}>{CAPABILITIES.map(([id,number,title,detail,href,key],index) => <li key={id} className={styles['node'+index]}><div className={styles.nodeMotion}><a href={href} className={styles.capability}><span>{number}</span><div><strong>{title}</strong><small>{detail}</small></div><Image className={styles.nodeGlow} src={NODE_ASSETS[key].src} width={NODE_ASSETS[key].width} height={NODE_ASSETS[key].height} unoptimized alt="" /></a></div></li>)}</ul>
        </div>
      </div>
    </section>
  </>;
}
