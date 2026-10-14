"use client";
import { Input } from "@/components/ui/input";
import { contentFieldLabels, type ContentField, type ContentSelection } from "@/lib/contracts/content-selection";

export function ContentSelectionFields({ value, onChange, disabled = false }: { value: ContentSelection; onChange: (value: ContentSelection) => void; disabled?: boolean }) {
  return <fieldset disabled={disabled} className="space-y-3">
    <legend className="font-medium">内容与语言</legend>
    <label className="block">目标语言<Input value={value.targetLanguage} maxLength={128} placeholder="例如 en-US、zh-Hans、ja" required onChange={(event) => onChange({ ...value, targetLanguage: event.target.value })} /></label>
    <p className="text-sm text-muted-foreground">使用语言标签明确指定目标语言，建议生成前仍需确认。</p>
    <fieldset className="flex flex-wrap gap-4"><legend>待处理字段</legend>
      {Object.entries(contentFieldLabels).map(([field, label]) => <label key={field} className="flex items-center gap-2"><input type="checkbox" checked={!!value[field as ContentField]} onChange={(event) => { const next = { ...value }; if (event.target.checked) next[field as ContentField] = true; else delete next[field as ContentField]; onChange(next); }} />{label}</label>)}
    </fieldset>
  </fieldset>;
}
