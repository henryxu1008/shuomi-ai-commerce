const timestamp = new Intl.DateTimeFormat("zh-CN", {
  timeZone: "Asia/Shanghai",
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
  hourCycle: "h23",
});

export function EntitlementTime({ value }: { value: string }) {
  return <time dateTime={value} title={value}>{timestamp.format(new Date(value))}（UTC+8）</time>;
}
