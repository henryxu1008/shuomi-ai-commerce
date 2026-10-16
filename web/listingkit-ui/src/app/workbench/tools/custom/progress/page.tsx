import { ToolPage } from "@/components/workbench/tools/tool-page";

export default async function Page({
  searchParams,
}: {
  searchParams: Promise<{ request?: string | string[] }>;
}) {
  const { request } = await searchParams;
  return (
    <ToolPage
      mode="progress"
      requestId={typeof request === "string" ? request : undefined}
    />
  );
}
