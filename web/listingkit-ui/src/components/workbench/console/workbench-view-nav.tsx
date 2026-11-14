import type {ReactNode} from "react";
import {cn} from "@/lib/utils/cn";
import styles from "./workbench-view-nav.module.css";

export const workbenchViewPage = styles.page;

/** Shared presentation for page views; links and buttons retain their native semantics. */
export function WorkbenchViewNav({label,children,className,group=false}:{label:string;children:ReactNode;className?:string;group?:boolean}) {
 const props={"aria-label":label,className:cn(styles.root,className)};
 return group?<div {...props} role="group">{children}</div>:<nav {...props}>{children}</nav>;
}
