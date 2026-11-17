"use client";

import { useState } from "react";
import {
  ArrowRight,
  BriefcaseBusiness,
  Building2,
  Check,
  CircleDollarSign,
  ShieldCheck,
  Store,
  Truck,
} from "lucide-react";
import styles from "./marketing-homepage.module.css";

const services = [
  {
    icon: Building2,
    title: "企业与资质",
    detail: "公司注册 · 开户 · 年审",
    description: "完善经营主体与基础资质，为跨境电商业务启动和长期运营做好准备。",
    offerings: ["公司注册", "银行开户", "年度审查", "税务登记", "主体变更", "资质办理", "受益人备案", "资料维护"],
    steps: ["确认主体", "准备材料", "提交办理", "结果交付"],
  },
  {
    icon: Store,
    title: "平台入驻",
    detail: "店铺开通 · 认证 · 申诉",
    description: "根据目标市场和经营品类，协助准备平台入驻资料、店铺认证与后续申诉。",
    offerings: ["店铺开通", "平台认证", "资料审核", "品牌授权", "类目审核", "资质补交", "账户申诉", "店铺诊断"],
    steps: ["评估平台", "准备资料", "提交审核", "开店交付"],
  },
  {
    icon: ShieldCheck,
    title: "品牌与合规",
    detail: "商标 · 专利 · 产品认证",
    description: "由专业机构协助处理商标、产品认证、备案与合规事项，帮助商品满足目标市场和销售平台要求。",
    offerings: ["商标注册", "品牌备案", "专利申请", "产品认证", "EPR合规", "责任保险", "侵权申诉", "合规咨询"],
    steps: ["提交需求", "匹配机构", "材料办理", "进度交付"],
  },
  {
    icon: Truck,
    title: "履约与仓储",
    detail: "头程 · 海外仓 · 尾程",
    description: "结合商品属性、销售区域和履约要求，规划头程运输、海外仓储与末端配送服务。",
    offerings: ["头程运输", "国际物流", "海外仓储", "库存管理", "尾程配送", "退换货", "物流追踪", "履约咨询"],
    steps: ["确认货量", "匹配线路", "安排入仓", "跟踪交付"],
  },
  {
    icon: CircleDollarSign,
    title: "财税与支付",
    detail: "VAT · 报税 · 跨境收款",
    description: "对接跨境经营所需的税务申报、账务支持与收款服务，帮助业务流程更清晰。",
    offerings: ["VAT注册", "税务申报", "账务整理", "税务咨询", "跨境收款", "收款账户", "费用核对", "财税规划"],
    steps: ["梳理业务", "确认地区", "匹配服务", "持续支持"],
  },
  {
    icon: BriefcaseBusiness,
    title: "法律与保险",
    detail: "合同 · 责任险 · 争议处理",
    description: "按实际经营场景连接法律顾问与保险服务，支持合同审阅、责任保障及争议处理。",
    offerings: ["合同审阅", "法律咨询", "产品责任险", "商业保险", "知识产权", "纠纷协助", "风险梳理", "合规建议"],
    steps: ["描述问题", "评估风险", "对接顾问", "跟进处理"],
  },
] as const;

export function ProfessionalServices() {
  const [selectedIndex, setSelectedIndex] = useState(2);
  const selected = services[selectedIndex];

  return <div className={styles.servicesLayout}>
    <div className={styles.serviceNetwork}>
      <div className={styles.serviceNetworkHeading}>
        <strong>专业服务网络</strong>
        <span>选择服务类别，查看对应的办理支持</span>
      </div>
      <div className={styles.serviceCore}>
        <strong>AI 方案协调中心</strong>
        <span>需求识别 · 服务匹配</span>
      </div>
      {services.map(({ icon: Icon, title, detail }, index) => <button
        aria-pressed={selectedIndex === index}
        className={`${styles.serviceOption} ${selectedIndex === index ? styles.serviceOptionActive : ""}`}
        key={title}
        onClick={() => setSelectedIndex(index)}
        type="button"
      >
        <Icon aria-hidden="true" size={20} />
        <span><strong>{title}</strong><small>{detail}</small></span>
      </button>)}
    </div>
    <div className={styles.serviceDetail} aria-live="polite" aria-atomic="true">
      <small>SELECTED SERVICE</small>
      <h3 key={`${selected.title}-title`} className={styles.serviceDetailAnimated}>{selected.title}</h3>
      <p key={`${selected.title}-description`} className={styles.serviceDetailAnimated}>{selected.description}</p>
      <div key={`${selected.title}-offerings`} className={styles.serviceOfferings}>
        {selected.offerings.map(item => <span key={item}><Check aria-hidden="true" size={13} /> {item}</span>)}
      </div>
      <ol key={`${selected.title}-steps`} className={styles.serviceSteps}>
        {selected.steps.map(step => <li key={step}>{step}</li>)}
      </ol>
      <a className={styles.serviceCta} href="#pricing">获取专业服务方案 <ArrowRight size={16} /></a>
    </div>
  </div>;
}
