"use client";

import { useState } from "react";
import { Select } from "@/components/ui/select";
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
  const [search, setSearch] = useState("");
  const name = (item: RegionOption) => localized ? chineseRegions.of(item.id) ?? item.name : item.name;
  const term = search.trim().toLocaleLowerCase();
  const matches = term ? options.filter(option => [name(option), option.name, ...(localized ? [option.id] : [])]
    .some(text => text.toLocaleLowerCase().includes(term))) : options;
  const shown = matches.slice(0, listLimit);
  const current = options.find(option => option.id === value);
  if (current && !shown.some(option => option.id === value)) shown.unshift(current);

  return <div className={styles.field}>
    {options.length > 20 ? <label className={styles.search}>
      搜索{label}
      <input type="search" value={search} onChange={event => setSearch(event.target.value)} disabled={disabled} />
    </label> : null}
    <label>
      {label}
      <Select className={styles.control} value={value} onChange={event => onChange(event.target.value)} disabled={disabled}>
        <option value="" disabled>{placeholder}</option>
        {shown.map(option => <option key={option.id} value={option.id}>{name(option)}</option>)}
      </Select>
    </label>
    {term && matches.length === 0 ? <small role="status">没有匹配项</small> : matches.length > listLimit ? <small>显示前 {listLimit} 项，可搜索缩小范围</small> : null}
  </div>;
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
