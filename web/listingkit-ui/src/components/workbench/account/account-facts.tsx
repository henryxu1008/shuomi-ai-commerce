"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState, useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { RegionPicker } from "@/components/ui/region-picker";
import regionStyles from "@/components/ui/region-picker.module.css";
import { AccountReadError } from "@/lib/api/account";
import {
  emptyRegion, getRegionCountry, getRegionIndex, matchSavedRegion, validRegionSelection,
  type RegionSelection,
} from "@/lib/api/region-catalog";
import {
  getAccountFacts,
  getAccountSessionAuthentication,
  getAccountPreferences,
  saveAccountPreferences,
  type RegionInput,
  type AccountPreferences,
} from "@/lib/api/account-facts";
import styles from "./account.module.css";

const date = (value: string) =>
  new Date(value).toLocaleString("zh-CN", {
    timeZone: "Asia/Singapore",
    hour12: false,
  });
export function IdentityDates({ subject }: { subject: string }) {
  const query = useQuery({
    queryKey: ["account-facts", subject],
    queryFn: ({ signal }) =>
      getAccountFacts({ expectedUserId: subject, signal }),
    retry: false,
    gcTime: 0,
    staleTime: 0,
  });
  const authentication = useQuery({
    queryKey: ["account-session-authentication", subject],
    queryFn: ({ signal }) => getAccountSessionAuthentication({ expectedUserId: subject, signal }),
    retry: false, gcTime: 0, staleTime: 0,
  });
  return (
    <span className={styles.identityMeta}>
      注册时间：
      {query.isPending || query.isFetching ? "正在读取…" : query.isError ? "暂不可用" :
        <time dateTime={query.data.registeredAt}>{date(query.data.registeredAt)}</time>}{" "}
      · 当前会话认证时间：
      {authentication.isPending || authentication.isFetching ? "正在读取…" :
        authentication.isError || !authentication.data.authenticatedAt ? "当前会话认证时间暂不可用" :
          <time dateTime={authentication.data.authenticatedAt}>{date(authentication.data.authenticatedAt)}</time>}
    </span>
  );
}
function usePreferences(subject: string) {
  return useQuery({
    queryKey: ["account-preferences", subject],
    queryFn: ({ signal }) =>
      getAccountPreferences({ expectedUserId: subject, signal }),
    retry: false,
    gcTime: 0,
    staleTime: 0,
  });
}
export function RegionSettings({ subject }: { subject: string }) {
  const query = usePreferences(subject);
  if (query.isPending || query.isFetching)
    return <p className={styles.note}>正在读取地区资料…</p>;
  if (query.isError)
    return (
      <p className={styles.errorText} role="status">
        地区资料暂不可用，请刷新后重试。
      </p>
    );
  return (
    <RegionForm key={`${subject}:${query.data.updatedAt}`} data={query.data} />
  );
}
function RegionForm({ data }: { data: AccountPreferences }) {
  const [draft, setDraft] = useState<RegionSelection | null>(null);
  const [explicitClear, setExplicitClear] = useState(false);
  const subject = data.userId;
  const index = useQuery({
    queryKey: ["region-catalog", subject],
    queryFn: ({ signal }) => getRegionIndex(subject, signal),
    retry: false,
    refetchOnWindowFocus: false,
  });
  const savedLabel = [data.country, data.province, data.city].filter(Boolean).join(" / ");
  const initialCountry = index.data?.countries.find(option =>
    savedLabel ? option.name === data.country : option.id === "CN",
  )?.id ?? "";
  const countryId = draft?.country ?? initialCountry;
  const detail = useQuery({
    queryKey: ["region-catalog", subject, index.data?.version, countryId],
    queryFn: ({ signal }) => getRegionCountry(subject, countryId, signal),
    enabled: Boolean(index.data && countryId),
    retry: false,
    refetchOnWindowFocus: false,
  });
  const versionMismatch = Boolean(index.data && detail.data && index.data.version !== detail.data.version);
  const country = versionMismatch ? undefined : detail.data?.country;
  const savedSelection = matchSavedRegion(data, country);
  const selection = draft ?? savedSelection ?? { ...emptyRegion, country: countryId };

  const controller = useRef<AbortController | null>(null);
  useEffect(() => {
    const c = new AbortController();
    controller.current = c;
    return () => c.abort();
  }, []);
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: (input: RegionInput) => saveAccountPreferences({
      expectedUserId: subject,
      input,
      signal: controller.current?.signal,
    }),
    onSuccess: (saved) => {
      if (!controller.current?.signal.aborted)
        queryClient.setQueryData(["account-preferences", subject], saved);
    },
  });
  const failedCatalog = index.isError || detail.isError || versionMismatch;
  const refreshing = index.isFetching || (Boolean(countryId) && detail.isFetching);
  const ready = Boolean(index.data && !failedCatalog && !refreshing &&
    (explicitClear || ((draft || savedSelection) && validRegionSelection(selection, country))));
  const savedNeedsSelection = savedLabel && !savedSelection && !refreshing;
  const errorText = mutation.error instanceof AccountReadError && mutation.error.status === 400
    ? "地区选项已无效，请刷新地区目录后重新选择"
    : "保存结果未确认，请刷新资料核对后重试。";

  return (
    <form className={regionStyles.form} onSubmit={event => {
      event.preventDefault();
      if (ready && !mutation.isPending && index.data)
        mutation.mutate({ catalogVersion: index.data.version, ...selection });
    }}>
      {savedNeedsSelection ? <p className={styles.note}>
        已保存地区：{savedLabel}{draft === null ? "。请从目录重新选择，或明确清空。" : null}
      </p> : null}
      {explicitClear ? <p className={styles.note} role="status">保存后将清空已保存的地区资料</p> : null}
      <RegionPicker countries={index.data?.countries ?? []} country={country}
        selection={selection} disabled={mutation.isPending || !index.data || index.isFetching}
        loading={Boolean(countryId) && detail.isFetching} onChange={next => {
          setExplicitClear(false);
          setDraft(next);
          mutation.reset();
        }} />
      {index.isPending ? <p className={styles.note} role="status">正在读取地区目录…</p> : null}
      {failedCatalog ? <p className={styles.errorText} role="alert">地区目录暂时无法读取，请重试</p> : null}
      <div className={regionStyles.actions}>
        <Button type="submit" variant="outline" disabled={!ready || mutation.isPending}>
          {mutation.isPending ? "正在保存…" : "保存地区资料"}
        </Button>
        {savedLabel && !explicitClear ? <Button type="button" variant="ghost" disabled={mutation.isPending} onClick={() => {
          setDraft(emptyRegion);
          setExplicitClear(true);
          mutation.reset();
        }}>清空地区</Button> : null}
        {failedCatalog || (mutation.error instanceof AccountReadError && mutation.error.status === 400)
          ? <Button type="button" variant="ghost" onClick={() => {
              void index.refetch();
              if (countryId) void detail.refetch();
            }}>刷新地区目录</Button> : null}
        {mutation.isSuccess ? <span className={styles.successText} role="status">地区资料已保存</span> : null}
        {mutation.isError ? <span className={styles.errorText} role="alert">{errorText}</span> : null}
      </div>
    </form>
  );
}
