import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {collectionFailureText,type useCollectionCommand} from "./use-collection-command";

export function CollectionCommandFeedback({state}:{state:ReturnType<typeof useCollectionCommand>}){
 return <>{state.pending?<Card className="space-y-3 border-border bg-muted p-4"><h2 className="font-semibold">{state.foreign?"原身份下有待核实的操作":"结果待核实"}</h2><p className="break-all text-sm text-muted-foreground">{state.foreign?"请回到原身份与企业后处理。":"保存内容已锁定，请先核实原操作。"}操作键：{state.pending.key}</p>{!state.foreign?<div className="flex flex-wrap gap-2"><Button type="button" disabled={state.busy} onClick={()=>void state.run(state.pending!.command,state.pending!,true)}>核实原操作</Button>{state.error==="NOT_FOUND"?<Button type="button" variant="outline" disabled={state.busy} onClick={()=>void state.run(state.pending!.command,state.pending!)}>重试原请求</Button>:null}</div>:null}</Card>:null}{state.error&&state.error!=="OUTCOME_UNKNOWN"?<p role="alert" className="text-sm text-destructive">{collectionFailureText[state.error]??"暂时无法完成请求，请刷新或核实原操作。"}</p>:null}</>;
}
