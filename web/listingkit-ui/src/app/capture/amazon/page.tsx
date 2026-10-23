import type { Metadata } from "next";
import { AmazonCaptureReceiver } from "./capture-receiver";
export const metadata:Metadata={title:"Amazon 插件采集",referrer:"no-referrer",robots:{index:false,follow:false}};
export default function AmazonCapturePage(){return <AmazonCaptureReceiver/>;}
