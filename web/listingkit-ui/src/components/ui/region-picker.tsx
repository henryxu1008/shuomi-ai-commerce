"use client";

import { useId, useRef, useState, type KeyboardEvent } from "react";
import * as Popover from "@radix-ui/react-popover";
import { ChevronDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { regionCities, type RegionCountry, type RegionOption, type RegionSelection } from "@/lib/api/region-catalog";
import styles from "./region-picker.module.css";

const chineseRegions = new Intl.DisplayNames(["zh-CN"], { type: "region", fallback: "none" });
const listLimit = 100;

function RegionField({ label, value, options, placeholder, disabled, onChange, localized = false }: {
  label: string;
  value: string;
  options: readonly RegionOption[];
  placeholder: string;
  disabled: boolean;
  onChange: (value: string) => void;
  localized?: boolean;
}) {
  const labelId = useId();
  const valueId = useId();
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const list = useRef<HTMLUListElement>(null);
  const name = (item: RegionOption) => localized ? chineseRegions.of(item.id) ?? item.name : item.name;
  const term = search.trim().toLocaleLowerCase();
  const matches = term ? options.filter(option => [name(option), option.name, ...(localized ? [option.id] : [])]
    .some(text => text.toLocaleLowerCase().includes(term))) : options;
  const shown = matches.slice(0, listLimit);
  const current = options.find(option => option.id === value);

  const moveFocus = (event: KeyboardEvent, fromSearch = false) => {
    if (event.nativeEvent.isComposing || (event.key !== "ArrowDown" && event.key !== "ArrowUp")) return;
    const buttons = Array.from(list.current?.querySelectorAll("button") ?? []);
    if (!buttons.length) return;
    const step = event.key === "ArrowDown" ? 1 : -1;
    const index = fromSearch ? (step === 1 ? -1 : 0) : buttons.indexOf(event.target as HTMLButtonElement);
    event.preventDefault();
    buttons[(index + step + buttons.length) % buttons.length].focus();
  };

  return (
    <div className={styles.field}>
      <span id={labelId} className={styles.label}>{label}</span>
      <Popover.Root open={open && !disabled} onOpenChange={next => {
        if (next) setSearch("");
        setOpen(next);
      }}>
        <Popover.Trigger asChild>
          <Button type="button" variant="outline" className={styles.control}
            aria-labelledby={`${labelId} ${valueId}`} disabled={disabled}>
            <span id={valueId} className={styles.value} data-placeholder={!current || undefined}>{current ? name(current) : placeholder}</span>
            <ChevronDown size={16} aria-hidden="true" />
          </Button>
        </Popover.Trigger>
        <Popover.Portal>
          <Popover.Content className={styles.dropdown} aria-labelledby={labelId} align="start" sideOffset={8} collisionPadding={12}>
            {options.length > 20 ? (
              <input type="search" className={styles.search} aria-label={`搜索${label}`} placeholder={`搜索${label}`}
                value={search} onChange={event => setSearch(event.target.value)}
                onKeyDown={event => {
                  if (event.key === "Enter" && !event.nativeEvent.isComposing) event.preventDefault();
                  moveFocus(event, true);
                }} />
            ) : null}
            <ul ref={list} className={styles.options} onKeyDown={event => moveFocus(event)}>
              {shown.map(option => (
                <li key={option.id}>
                  <button type="button" className={styles.option} aria-pressed={value === option.id} onClick={() => {
                    if (value !== option.id) onChange(option.id);
                    setOpen(false);
                  }}>{name(option)}</button>
                </li>
              ))}
            </ul>
            {matches.length === 0 ? <p className={styles.hint} role="status">没有匹配项</p> :
              matches.length > listLimit ? <p className={styles.hint}>显示前 {listLimit} 项，可搜索缩小范围</p> : null}
          </Popover.Content>
        </Popover.Portal>
      </Popover.Root>
    </div>
  );
}

export function RegionPicker({ countries, country, selection, onChange, disabled = false, loading = false }: {
  countries: readonly RegionOption[];
  country?: RegionCountry;
  selection: RegionSelection;
  onChange: (selection: RegionSelection) => void;
  disabled?: boolean;
  loading?: boolean;
}) {
  const provinces = country?.provinces ?? [];
  const provinceOptions = provinces.length && country?.cities.length
    ? [...provinces, { id: "direct", name: "不适用省／州" }]
    : provinces;
  const cities = regionCities(country, selection.province);
  const waitingProvince = provinceOptions.length > 0 && !selection.province;

  return <div className={styles.fields}>
    <RegionField label="国家／地区" value={selection.country} options={countries} placeholder="请选择国家／地区" disabled={disabled} localized
      onChange={value => onChange({ country: value, province: "", city: "" })} />
    <RegionField key={selection.country} label="省／州" value={selection.province} options={provinceOptions}
      placeholder={loading ? "正在读取…" : !country ? "请先选择国家／地区" : provinceOptions.length ? "请选择省／州" : "不适用"}
      disabled={disabled || !provinceOptions.length} onChange={value => onChange({ ...selection, province: value, city: "" })} />
    <RegionField key={`city:${selection.country}:${selection.province}`} label="城市" value={selection.city} options={cities}
      placeholder={loading ? "正在读取…" : !country ? "请先选择国家／地区" : waitingProvince ? "请先选择省／州" : cities.length ? "请选择城市" : "该地区暂无预置城市"}
      disabled={disabled || !cities.length} onChange={value => onChange({ ...selection, city: value })} />
  </div>;
}
