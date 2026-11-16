import Image from "next/image";
import { formatRetailMoney, singleRetailOffer, type RetailPriceCatalog } from "@/lib/api/retail-prices";
import Link from "next/link";
import {
  ArrowRight, BarChart3, BrainCircuit, BookOpenCheck, BriefcaseBusiness, Building2,
  Check, CircleDollarSign, Database, Megaphone,
  Network, PackageSearch, Palette, ShieldCheck,
  Store, Timer, Truck, Users,
} from "lucide-react";

import { ContactPanel } from "./contact-panel";
import { BrandClosing, MarketingTeam } from "./marketing-illustrations";
import { MarketingHero } from "./marketing-hero";
import styles from "./marketing-homepage.module.css";
import { PracticeTabs } from "./practice-tabs";
import { RoleSolutionTabs } from "./role-solution-tabs";

const ASSET = "/sumi";
const LOGIN_HREF = "/login?returnTo=%2Fworkbench";

const industrySteps = ["市场研究", "寻找商品", "对接供应链", "制作内容", "运营店铺", "分析增长"];
const painPoints = [[Timer, "人力成本高", "市场调研、商品筛选到内容制作，重复操作占用团队时间。", "团队精力被日常执行切碎，难以投入高价值判断。"], [BookOpenCheck, "经验难复制", "关键判断沉淀在个人脑中，新成员接手慢，经营质量不稳定。", "有效方法难在团队与店铺之间持续复用。"], [Database, "数据相互割裂", "商品、店铺、供应链信息散落多处，难形成统一经营视角。", "信息来回整理，发现问题和机会都更慢。"], [Network, "业务响应缓慢", "多个环节依靠人工沟通推动，机会出现后难快速验证和落地。", "市场变化快，协作链路却常常跟不上。"]] as const;
const agents = [
  [PackageSearch, "AI选品经理", "发现市场机会，找到值得销售的商品。", "市场扫描 · 潜力商品 · 竞争与利润评估", "寻找适合美国市场的宠物用品。", "blue"],
  [Store, "AI运营经理", "诊断店铺问题，持续提升销售表现。", "店铺诊断 · Listing优化 · 增长规划", "分析店铺最近销量下降的原因。", "violet"],
  [Palette, "AI设计师", "生成商品视觉和营销内容。", "商品主图 · 场景图 · 详情页与广告素材", "为户外水杯设计Amazon商品图片。", "cyan"],
  [BarChart3, "AI数据分析师", "从复杂数据中发现市场规律。", "趋势分析 · 商品数据 · 竞品与消费洞察", "分析德国小家电市场增长趋势。", "blue"],
  [Truck, "AI供应链经理", "连接商品、供应商与销售机会。", "资源匹配 · 供应商比较 · 成本利润优化", "寻找供应商并比较产品报价。", "green"],
  [Megaphone, "AI广告经理", "制定广告策略，提高投放回报。", "广告分析 · 关键词 · 预算与效果优化", "优化Amazon广告投放方案。", "amber"],
] as const;
const processSteps = [["01", "市场洞察", ["国家与行业趋势", "消费需求分析", "市场容量评估", "竞争环境判断"], "市场机会报告"], ["02", "AI选品", ["潜力商品发现", "爆品趋势分析", "竞争强度评估", "利润空间测算"], "选品机会清单"], ["03", "商品开发", ["用户需求提取", "产品卖点规划", "差异化方向设计", "商品信息优化"], "商品开发方案"], ["04", "供应链匹配", ["商品货盘搜索", "供应商与工厂匹配", "报价成本比较", "供应风险评估"], "供应链解决方案"], ["05", "店铺运营", ["自动生成产品图片", "生成内容并上架", "店铺经营诊断", "广告投放优化"], "店铺增长方案"], ["06", "数据增长", ["销售数据分析", "竞品持续监控", "经营问题发现", "AI自动优化策略"], "持续增长建议"]] as const;
const supplySources = [["硕米自营", "自有工厂与自营产品，供应稳定，快速对接销售。"], ["硕米优选", "平台筛选并经AI评估的优质商品资源。"], ["全球货盘", "连接1688及第三方货盘，整合报价与起订量。"], ["用户供应链", "上传产品、供应商与工厂，统一数字化管理。"]];
const supplyOutputs = [["数字化商品档案", "自动建立标准化产品资料与供应记录。"], ["优质供应链", "完成供应评分、成本利润与风险评估。"], ["硕米优选商品", "符合标准的产品可申请进入硕米优选。"], ["全球销售机会", "让商品连接平台卖家与全球销售渠道。"]];
const dataInsights = [["市场数据", "市场规模、国家区域、增长趋势与消费变化。", "市场机会判断"], ["商品数据", "商品销量、价格、品类趋势与生命周期。", "潜力商品发现"], ["竞品数据", "竞品价格排名、评价、卖点与变化监控。", "商品开发方向"], ["消费洞察", "搜索需求、用户评价、消费场景与产品痛点。", "店铺增长策略"]];
const dataOutcomes = [["市场机会判断", "识别值得进入的国家、行业和细分市场。"], ["潜力商品发现", "结合趋势、竞争和利润寻找商品机会。"], ["商品开发方向", "从评价与消费需求提取产品改进方向。"], ["店铺增长策略", "用经营与竞品数据生成增长建议。"]];
const queryPlatforms = [["Amazon", "市场趋势"], ["TikTok Shop", "内容热度"], ["Temu", "商品价格"], ["1688", "供应资源"]];
const services = [[Building2, "企业与资质", "公司注册 · 开户 · 年审"], [Store, "平台入驻", "店铺开通 · 认证 · 申诉"], [ShieldCheck, "品牌与合规", "商标 · 专利 · 产品认证"], [Truck, "履约与仓储", "头程 · 海外仓 · 尾程"], [CircleDollarSign, "财税与支付", "VAT · 报税 · 跨境收款"], [BriefcaseBusiness, "法律与保险", "合同 · 责任险 · 争议处理"]] as const;
const footerGroups = [["产品能力", [["人工智能员工", "#agents"], ["人工智能选品", "#agents"], ["店铺智能运营", "#agents"], ["全球数据能力", "#data"]]], ["解决方案", [["跨境新手创业", "#solutions"], ["跨境卖家增长", "#solutions"], ["工厂产品出海", "#solutions"], ["企业智能协同", "#solutions"], ["OPC电商社区落地", "#solutions"]]], ["服务支持", [["创业陪跑", "#pricing"], ["店铺联合运营", "#pricing"], ["品牌出海方案", "#pricing"], ["功能定制开发", "#pricing"]]], ["关于硕米", [["品牌介绍", "#home"], ["应用实践", "#practices"], ["价格与服务", "#pricing"], ["联系我们", "#contact-launcher"]]]] as const;
const plans = [["01", "新手创业陪跑", "¥4,999 / 半年", "方向规划 · AI选品培训 · 店铺启动指导", "blue"], ["02", "店铺联合运营", "¥5,999 + 利润分成30%", "商品与市场分析 · 内容上架 · 运营执行", "green"], ["03", "品牌出海方案", "定制方案 · 咨询报价", "市场研究 · 产品组合 · 品牌本地化", "violet"], ["04", "功能定制开发", "按需求评估 · 咨询报价", "专属AI智能体 · 企业知识库 · 自动化工作流", "cyan"], ["05", "一人公司电商社区落地", "项目定制 · 咨询报价", "AI服务平台 · 资源接入 · 团队培训 · 运营陪跑", "green"]];

export function MarketingHomepage({ pricing = null }: { pricing?: RetailPriceCatalog | null } = {}) {
  const storePrice = singleRetailOffer(pricing, "store_renewal_period");
  const aiPrice = singleRetailOffer(pricing, "ai_point");
  const renderAgentCard = (agent: (typeof agents)[number], index: number) => {
    const [Icon, name, summary, skills, prompt, tone] = agent;
    return <article className={`${styles.agentCard} ${styles[tone]}`} data-agent-index={index} key={name}>
      <div className={styles.cardTop}><span><Icon size={21} /></span><small>0{index + 1} · AI EMPLOYEE</small></div>
      <h3>{name}</h3><p>{summary}</p><b>{skills}</b><blockquote>“{prompt}”</blockquote>
    </article>;
  };
  return <main className={styles.page}>
    <MarketingHero loginHref={LOGIN_HREF} />

    <section className={`${styles.section} ${styles.industry}`}><SectionHeading label="FROM MANUAL WORKFLOWS TO SMART COLLABORATION / 从人工串联，到智能协同" title="电商正在进入AI时代" description="从市场洞察、商品开发到供应链和店铺增长，传统依赖人工串联的经营方式，正在被AI重新组织。" centered /><div className={styles.industryFlow}>{industrySteps.map((step, index) => <div className={styles.industryStep} key={step}><span>{String(index + 1).padStart(2, "0")}</span><strong>{step}</strong></div>)}</div><div className={styles.painGrid}>{painPoints.map(([Icon, title, description, detail]) => <article key={title}><Icon size={22} /><h3>{title}</h3><p>{description}</p><small>{detail}</small></article>)}</div><p className={styles.statement}>真正需要改变的，不是增加更多工具，而是让 <strong>AI</strong> 重新组织整个电商流程。</p></section>

    <section className={`${styles.section} ${styles.teamSection}`} id="architecture" data-node-id="63:2"><SectionHeading label="TASK-DRIVEN E-COMMERCE AUTOMATION / 定任务　自动化执行　帮用户走完电商最后一里路" title="一个 AI 团队，完成完整电商流程" description="只需说出你的业务目标，硕米智能引擎便会理解需求、拆解任务，并自动调度AI智能体、数据能力与供应链资源，协同完成电商业务。" centered /><MarketingTeam /></section>

    <section className={`${styles.section} ${styles.agentSection}`} id="agents"><SectionHeading label="AI TEAM FOR YOUR BUSINESS / 打造适合自身业务的AI团队" title="每一个电商岗位，都可以拥有一位 AI 员工" description="从发现市场机会到店铺增长，硕米为不同业务岗位提供专业AI智能体。它们理解目标、调用数据、执行任务，并在协作中持续优化业务结果。" centered /><div className={styles.agentShowcase} role="group" aria-label="AI 团队岗位与控制中心">
      <div className={`${styles.agentColumn} ${styles.agentColumnLeft}`}>{agents.slice(0, 3).map(renderAgentCard)}</div>
      <div className={styles.agentControl}>
        <div className={styles.agentControlHeading}><span><Users size={22} /></span><div><small>AI TEAM CONTROL CENTER</small><strong>AI 团队控制中心</strong></div></div>
        <div className={styles.agentObjective}><small>用户目标</small><strong>打造一个可持续增长的商品业务</strong></div>
        <div className={styles.agentHub}><span className={styles.agentHubKicker}>一个目标</span><strong>多智能体协作</strong><span>理解 · 拆解 · 调度 · 优化</span></div>
        <div className={styles.agentRoleGrid}>{agents.map(([, name], index) => <div className={styles.agentRole} data-agent-index={index} key={name}><span>0{index + 1}</span>{name}</div>)}</div>
        <p className={styles.agentControlFooter}>一个任务，由多位 AI 员工自动协作完成</p>
      </div>
      <div className={`${styles.agentColumn} ${styles.agentColumnRight}`}>{agents.slice(3).map((agent, index) => renderAgentCard(agent, index + 3))}</div>
    </div></section>

    <section className={`${styles.section} ${styles.processSection}`} id="workflow" data-node-id="81:56"><div className={styles.processAmbient}><Image src="/sumi/figma-home-process-img.svg" alt="" width={1120} height={700} unoptimized /></div><SectionHeading label="END-TO-END AI COMMERCE / 全流程 AI 电商" title="从发现机会到实现增长，全流程由 AI 协同完成" description="硕米将市场、商品、供应链、内容、店铺和数据连接成完整业务流程，让每一步都可以被理解、执行和持续优化。" centered /><div className={styles.processGrid}>{processSteps.map(([number, title, detail, output]) => <article key={number}><div className={styles.processNode}><div className={styles.processHalo}><Image src="/sumi/figma-home-process-img1.svg" alt="" width={72} height={72} unoptimized /></div><div className={styles.processCore}><Image src="/sumi/figma-home-process-img2.svg" alt="" width={76} height={76} unoptimized /></div><span>{number}</span></div><div className={styles.processCard}><h3>{title}</h3><ul>{detail.map(item => <li key={item}>{item}</li>)}</ul><div className={styles.processOutput}><small>输出结果</small><b>{output}</b></div></div></article>)}</div><div className={styles.processFooter}><div><Network size={24} /><p><strong>一套系统，连接完整电商业务。</strong><span>不需要在多个平台、工具和服务之间反复切换，硕米智能引擎会根据任务自动调用所需能力。</span></p></div><a href="#solutions">查看解决方案 <ArrowRight size={16} /></a></div></section>

    <section className={`${styles.section} ${styles.supplySection}`} id="supply-chain"><SectionHeading label="AI SUPPLY CHAIN NETWORK / AI 智能供应链" title="连接商品、货盘与工厂，让好产品快速进入市场" description="硕米连接自营产品、精选供应商、全球货盘和用户自有供应链，并通过AI完成产品识别、信息优化、供应评估和销售机会匹配。" centered /><div className={styles.supplyMap}><SupplyColumn items={supplySources} offset={1} track="sourcing" eyebrow="SUPPLY BUSINESS LINE" title="供给资源业务线" description="多渠道货源接入与供应评估" /><div className={styles.supplyCore} aria-label="硕米 AI 供应链引擎"><span className={styles.supplyCoreEyebrow}>全链路智能协同</span><strong>硕米 AI<br />供应链引擎</strong><p>整合多元供给，驱动商品评估<br />与销售机会匹配</p><div className={styles.supplyCoreTags}><span>货源接入</span><span>商品评估</span><span>供应优化</span><span>销售匹配</span></div></div><SupplyColumn items={supplyOutputs} offset={1} track="commerce" eyebrow="COMMERCE BUSINESS LINE" title="商品经营业务线" description="从商品标准化到全球销售机会" /></div><div className={styles.onboardingBar}><div className={styles.onboardingIntro}><span>PRODUCT ONBOARDING</span><strong>你的产品，也可以进入硕米销售生态</strong></div><div className={styles.onboardingSteps}>{["上传产品资料", "AI识别产品", "生成产品档案", "供应链评分", "获得销售机会"].map((step, index) => <div className={styles.onboardingStep} key={step}><span>0{index + 1}</span><strong>{step}</strong></div>)}</div></div></section>

    <section className={`${styles.section} ${styles.dataSection}`} id="data"><SectionHeading label="GLOBAL COMMERCE DATA NETWORK / 全球商业数据网络" title="用数据理解市场，发现下一个增长机会" description="硕米连接全球市场、商品、竞品和消费数据，让AI的每一次分析、选品与经营决策都有数据依据。" centered /><div className={styles.dataNetwork}><div className={`${styles.dataColumn} ${styles.dataColumnInputs}`}><div className={styles.dataColumnHeading}><span>DATA INPUTS</span><h3>多维经营数据</h3></div>{dataInsights.map(([title, description], index) => <article key={title}><span>0{index + 1}</span><div><h3>{title}</h3><p>{description}</p></div></article>)}</div><div className={styles.dataEngineCard}><span className={styles.dataEngineEyebrow}>SUMI GLOBAL DATA ENGINE</span><h3>硕米全球数据引擎</h3><div className={styles.dataDecision}><div><strong>全球数据<br />智能决策</strong></div></div><div className={styles.dataSourceList}><div><strong>海外平台</strong><span>Amazon · eBay · Walmart · Etsy</span></div><div><strong>跨境平台</strong><span>TikTok Shop · Temu · SHEIN · AliExpress</span></div><div><strong>国内与货源</strong><span>淘宝 · 天猫 · 京东 · 拼多多 · 抖音 · 1688</span></div></div><p>基于公开、授权及用户接入数据<br />实际范围以平台接口与产品权限为准</p></div><div className={`${styles.dataColumn} ${styles.dataColumnOutcomes}`}><div className={styles.dataColumnHeading}><span>BUSINESS OUTCOMES</span><h3>业务决策结果</h3></div>{dataOutcomes.map(([title, description], index) => <article key={title}><span>A{index + 1}</span><div><h3>{title}</h3><p>{description}</p></div></article>)}</div></div><div className={styles.dataQuery}><div className={styles.dataQueryHeading}><h3>一个需求，同时调用多个平台数据</h3><p>“帮我分析美国宠物用品市场，并寻找国内供应资源。”</p></div><div className={styles.dataQueryFlow}>{queryPlatforms.map(([platform, capability]) => <div className={styles.dataPlatform} key={platform}><strong>{platform}</strong><span>{capability}</span></div>)}<div className={styles.dataEngineResult}><strong>硕米数据引擎</strong><span>清洗 · 识别 · 评分</span></div><div className={styles.dataAdviceResult}><strong>AI 商业建议</strong><span>市场 · 商品 · 供应链</span></div></div><div className={styles.dataQueryFooter}>数据不只是用来查看，而是直接驱动 AI 完成业务。</div></div></section>

    <section className={`${styles.section} ${styles.solutionSection}`} id="solutions"><SectionHeading label="ROLE-BASED SOLUTIONS / 角色解决方案" title="不同的业务起点，同一套 AI 增长能力" description="硕米根据不同用户的资源、经验和业务目标，组合AI智能体、数据、供应链和工具能力，提供针对性的业务路径。" centered /><RoleSolutionTabs /><div className={styles.opcBanner}><div className={styles.opcMain}><div className={styles.opcCopy}><span>OPC ECOMMERCE COMMUNITY</span><strong>一个人、一支 AI 团队、一个共同成长的电商创业社区</strong><small>创建个人项目，配置 AI 员工，接入商品、数据和供应链，并通过社区协作和经验分享获得业务与收益机会。</small></div><div className={styles.opcActions}><ContactPanel inlineTrigger inlineTriggerLabel="咨询 OPC 社区" showFloatingLauncher={false} /><a href="#practices">查看 OPC 创业案例</a></div></div><div className={styles.opcPills}>{["AI创业团队", "创业项目空间", "商品与数据资源", "社区协作", "能力与经验变现"].map(item => <span key={item}>{item}</span>)}</div></div></section>

    <section className={`${styles.section} ${styles.serviceSection}`} id="services"><SectionHeading label="GLOBAL PROFESSIONAL SERVICES / 全球专业服务生态" title="AI完成线上业务，专业机构负责关键落地" description="硕米连接企业设立、平台入驻、品牌合规、跨境履约、财税支付和法律保险等专业资源，让AI方案真正转化为可执行的全球业务。" centered /><div className={styles.servicesLayout}><div className={styles.serviceNetwork}><div className={styles.serviceNetworkHeading}><strong>专业服务网络</strong><span>AI识别需求后，连接对应的专业服务机构</span></div><div className={styles.serviceCore}><strong>AI 方案协调中心</strong><span>需求识别 · 服务匹配</span></div>{services.map(([Icon, title, detail]) => <article key={title}><Icon size={20} /><div><h3>{title}</h3><p>{detail}</p></div></article>)}</div><div className={styles.serviceDetail}><small>SELECTED SERVICE</small><h3>品牌、知识产权与合规</h3><p>由专业机构完成申请、认证、备案与争议处理，帮助商品满足目标市场和销售平台要求。</p><div>{["商标注册", "品牌备案", "专利申请", "产品认证", "EPR合规", "责任保险", "侵权申诉", "合规咨询"].map(item => <span key={item}><Check size={13} /> {item}</span>)}</div><ol><li>提交需求</li><li>匹配机构</li><li>材料办理</li><li>进度交付</li></ol><a className={styles.serviceCta} href="#pricing">获取专业服务方案 <ArrowRight size={16} /></a></div></div><div className={styles.serviceMatching}><div className={styles.serviceMatchingHeading}><span>AI SERVICE MATCHING / 专业服务智能匹配</span><h3>告诉 AI 你的经营目标，自动识别需要办理的专业服务</h3></div><div className={styles.serviceMatchingGrid}><div><small>经营目标</small><p>将儿童玩具销售到德国 Amazon</p></div><div><small>推荐专业服务组合</small><p>德国商标 → 产品认证 → EPR合规 → 平台入驻 → VAT注册 → 欧洲海外仓</p></div></div><div className={styles.serviceMatchingFooter}><span>AI负责识别、匹配和协调，专业机构负责实际办理与交付。</span></div></div></section>

    <section className={`${styles.section} ${styles.practiceSection}`} id="practices"><SectionHeading label="AI COMMERCE PRACTICES / AI电商应用实践" title="从一个业务目标，到一套可以执行的结果" description="硕米智能引擎组织AI员工、商品数据、供应链资源和专业服务，帮助不同类型的用户完成完整电商任务。" /><PracticeTabs /></section>

    <section className={`${styles.section} ${styles.pricingSection}`} id="pricing">
      <SectionHeading label="PRICING & SERVICE PLANS / 价格与服务方案" title="简单透明，只为实际使用付费" description="店铺按30天一期付费，AI点数预付使用；需要业务落地支持时，可咨询对应的专项服务。" centered />
      <div className={styles.trialBanner}>
        <span>新用户试用计划</span><strong>15天体验计划 · 待开放</strong>
        <p>计划提供1个店铺试用与¥5 AI算力券。试用计划尚未开放，注册不会自动赠送资源。</p>
        <Link href={LOGIN_HREF}>注册进入硕米 <ArrowRight size={15} /></Link>
      </div>
      <div className={styles.basePricing}>
        <article><small>店铺基础接入</small>
          <h3>{storePrice ? <><strong>{formatRetailMoney(storePrice.unit_price_minor)}</strong> / 店铺 / 30 天</> : <strong>价格暂不可用</strong>}</h3>
          <p>1期为30天；购买后需完成平台授权，再显式开通或续费。<br />各店铺分别计时，不自动续费；平台支持情况以工作台为准。</p>
          {storePrice ? <Link href="/login?returnTo=%2Fworkbench%2Fplans%2Foptions">购买店铺服务 <ArrowRight size={15} /></Link> : null}
        </article>
        <article><small>人工智能算力</small>
          <h3>{aiPrice ? <><strong>{formatRetailMoney(aiPrice.unit_price_minor)}</strong> / AI点</> : <strong>价格暂不可用</strong>}</h3>
          <p>AI点数预付使用，余额不按月清零。<br />各模型按对应费率和实际用量扣点，购买点数不会自动调用AI。</p>
          {aiPrice ? <Link href="/login?returnTo=%2Fworkbench%2Fplans%2Foptions">购买AI点数 <ArrowRight size={15} /></Link> : null}
        </article>
      </div>
      <div className={styles.professionalHeader}><div><span>专项落地服务</span><h3>从使用人工智能工具，到完成业务落地</h3></div><p>以下为专项服务参考报价；服务范围、周期与结算以双方确认的合同为准。</p></div>
      <div className={styles.planGrid}>{plans.map(([number, title, price, detail, tone]) => <article className={styles[tone]} key={title}><span>{number}</span><h3>{title}</h3><b>{price}</b><p>{detail}</p></article>)}</div>
      <p className={styles.pricingNote}>店铺与AI点数仅展示当前有效报价；暂无有效报价时暂不可购买。店铺按30天一期 · AI点数按模型和任务的实际用量扣减 · 联合运营需项目评估 · 不承诺固定销售额或固定利润</p>
    </section>

    <BrandClosing />
    <footer className={styles.footer} data-node-id="234:204"><div className={styles.footerIdentity}><div className={styles.footerBrand}><Image src="/sumi/c3ea4c6f-992f-4c3d-ae2e-c14d4ec9735a.png" width={48} height={48} alt="" /><div><strong>硕米智能引擎</strong><span>新一代人工智能电商操作系统</span></div></div><p>连接智能体、数据、商品、供应链与专业服务，帮助个人和组织完成电商业务。</p></div><nav aria-label="页尾导航" className={styles.footerLinks}>{footerGroups.map(([title,links]) => <div key={title}><h3>{title}</h3><ul>{links.map(([label,href]) => <li key={label}><>{label === "联系我们" ? <ContactPanel inlineTrigger /> : <a href={href}>{label}</a>}</></li>)}</ul></div>)}</nav><div className={styles.copyright}>© 2026 硕米智能引擎　保留所有权利 <span><Link href="/privacy-policy">隐私政策</Link><Link href="/user-agreement">用户协议</Link><Link href="/ai-compute-billing">算力计费说明</Link><Link href="/service-agreement">服务协议</Link></span></div></footer>
  </main>;
}

function SectionHeading({ label, title, description, centered = false }: { label?: string; title: string; description: string; centered?: boolean }) { return <div className={`${styles.sectionHeading} ${centered ? styles.centered : ""}`}>{label ? <p>{label}</p> : null}<h2>{title}</h2><span>{description}</span></div>; }
function SupplyColumn({ items, offset, track, eyebrow, title, description }: { items: string[][]; offset: number; track: "sourcing" | "commerce"; eyebrow: string; title: string; description: string }) { return <div className={styles.supplyColumn} data-track={track}><div className={styles.supplyColumnHeading}><span>{eyebrow}</span><h3>{title}</h3><p>{description}</p></div><div className={styles.supplyColumnCards}>{items.map(([title, description], index) => <article key={title}><span>0{index + offset}</span><div><h3>{title}</h3><p>{description}</p></div></article>)}</div></div>; }
