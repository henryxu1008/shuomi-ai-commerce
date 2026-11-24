import Link from "next/link";

export const metadata = { title: "连接 Amazon 店铺" };

export default async function AmazonStoreConnectionResult({
  searchParams,
}: {
  searchParams: Promise<{ reason?: string | string[] }>;
}) {
  const { reason } = await searchParams;
  return (
    <section className="space-y-4">
      <h1>连接 Amazon 店铺</h1>
      <p>{reason === "signin" ? "工作台登录已失效，请重新登录后连接店铺。" : "这次未能确认授权结果，请回到店铺中心查看店铺状态。"}</p>
      <Link href="/workbench/stores">返回店铺中心</Link>
    </section>
  );
}
