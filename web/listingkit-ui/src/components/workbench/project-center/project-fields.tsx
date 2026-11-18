import {Textarea} from "@/components/ui/textarea";
import {Input} from "@/components/ui/input";
import {kinds,type ProjectBody} from "@/lib/contracts/project-center";
import styles from "./project-page.module.css";

export function ProjectFields({body,onChange,researchOnly,disabled,autoFocus}:{
  body:ProjectBody;
  onChange:(key:keyof ProjectBody,value:string)=>void;
  researchOnly?:boolean;
  disabled?:boolean;
  autoFocus?:boolean;
}) {
  return <>
    <label>项目名称<Input value={body.title} onChange={event=>onChange("title",event.target.value)} required placeholder="例如：一号店经营提升" autoFocus={autoFocus} disabled={disabled} maxLength={256}/></label>
    <label>项目目标<Textarea value={body.goal} onChange={event=>onChange("goal",event.target.value)} required rows={4} maxLength={4096} disabled={disabled} placeholder="希望达成什么长期目标？"/></label>
    <div className={styles.formGrid}>
      <label>项目类型<select disabled={disabled||researchOnly} value={body.kind} onChange={event=>onChange("kind",event.target.value)}>{Object.entries(kinds).map(([key,label])=><option key={key} value={key}>{label}</option>)}</select></label>
      <label>截止日期（可选）<Input type="date" disabled={disabled} value={body.dueDate} onChange={event=>onChange("dueDate",event.target.value)}/></label>
    </div>
  </>;
}
