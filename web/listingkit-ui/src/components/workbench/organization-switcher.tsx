"use client";

import { useId, useState } from "react";

import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select } from "@/components/ui/select";

export function OrganizationSwitcher({ onConfirm, onCancel }: { onConfirm?: () => void; onCancel?: () => void }) {
  const selectId = useId();
  const [selectedOrganizationId, setSelectedOrganizationId] = useState<string | null>(null);
  const {
    organizations,
    effectiveOrganization,
    selectionRequired,
    isLoading,
    isSwitching,
    blockingError,
    switchOrganization,
  } = useWorkbenchContext();

  if (blockingError) {
    return (
      <p className="text-sm text-destructive" role="alert">
        {workbenchErrorMessage(blockingError.code)}
      </p>
    );
  }

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">正在加载企业...</p>;
  }

  if (organizations.length <= 1) {
    return (
      <div aria-label="当前企业" className="min-w-0">
        <p className="text-xs text-muted-foreground">当前企业</p>
        <p className="truncate text-sm font-medium">
          {organizations[0]?.name ?? "暂无可用企业"}
        </p>
      </div>
    );
  }

  const needsPrompt = selectionRequired || !effectiveOrganization;
  const currentId = effectiveOrganization?.id ?? "";
  const selectedId = organizations.some(organization => organization.id === selectedOrganizationId)
    ? selectedOrganizationId ?? currentId
    : currentId;
  const canConfirm = !isSwitching && selectedId !== "" && selectedId !== currentId;

  return (
    <form onSubmit={(event) => {
      event.preventDefault();
      if (!canConfirm) return;
      setSelectedOrganizationId(null);
      onConfirm?.();
      switchOrganization(selectedId);
    }}>
      <div className="grid gap-1.5">
        <Label htmlFor={selectId}>选择企业</Label>
        <Select
          aria-busy={isSwitching}
          disabled={isSwitching}
          id={selectId}
          onChange={event => setSelectedOrganizationId(event.currentTarget.value)}
          value={selectedId}
        >
          {needsPrompt ? <option disabled value="">请选择企业</option> : null}
          {organizations.map(organization => (
            <option key={organization.id} value={organization.id}>{organization.name}</option>
          ))}
        </Select>
      </div>
      <footer className="mt-5 flex justify-end gap-3">
        {onCancel ? <Button type="button" variant="outline" onClick={onCancel}>取消</Button> : null}
        <Button type="submit" disabled={!canConfirm}>确认切换</Button>
      </footer>
    </form>
  );
}

export function workbenchErrorMessage(code: string) {
  switch (code) {
    case "AUTHENTICATION_REQUIRED":
      return "登录状态已失效，请重新登录";
    case "ORGANIZATION_ACCESS_DENIED":
      return "你没有访问所选企业的权限";
    case "ORGANIZATION_ACCESS_REVOKED":
      return "你对所选企业的访问权限已被撤销";
    case "ORGANIZATION_SUSPENDED":
      return "所选企业当前不可用";
    case "DEPENDENCY_UNAVAILABLE":
      return "企业权限服务暂时不可用，请稍后重试";
    default:
      return "工作台访问状态无法确认，请重试";
  }
}
