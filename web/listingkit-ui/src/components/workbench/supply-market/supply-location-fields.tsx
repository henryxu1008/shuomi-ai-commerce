"use client";

import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { getRegionCountry, type RegionCountry } from "@/lib/api/region-catalog";

type Location = { province: string; city: string };

export function useShippingLocations(subject: string, enabled = true) {
  return useQuery({
    queryKey: ["region-catalog", subject, "shipping", "CN"],
    queryFn: ({ signal }) => getRegionCountry(subject, "CN", signal),
    enabled,
    staleTime: 15 * 60 * 1000,
    retry: false,
  });
}

export function isShippingLocation(country: RegionCountry | undefined, { province, city }: Location, requireCity = true) {
  const parent = country?.provinces.find(option => option.name === province);
  return Boolean(parent && ((!requireCity && !city) || parent.cities.some(option => option.name === city)));
}

export function SupplyLocationFields({ locations, province, city, onChange, required = true, filter = false }: Location & {
  locations: ReturnType<typeof useShippingLocations>;
  onChange: (location: Location) => void;
  required?: boolean;
  filter?: boolean;
}) {
  const provinces = locations.data?.country.provinces;
  const current = provinces?.find(option => option.name === province);
  return <>
    <label>发货省份
      <Select required={required} disabled={!provinces} value={current ? province : ""}
        onChange={event => onChange({ province: event.target.value, city: "" })}>
        <option value="">{provinces ? (filter ? "全部省份" : "请选择省份") : (locations.isError ? "暂时无法读取" : "正在读取…")}</option>
        {provinces?.map(option => <option key={option.id} value={option.name}>{option.name}</option>)}
      </Select>
    </label>
    <label>发货城市
      <Select required={required} disabled={!current} value={current?.cities.some(option => option.name === city) ? city : ""}
        onChange={event => onChange({ province, city: event.target.value })}>
        <option value="">{current ? (filter ? "全部城市" : "请选择城市") : "请先选择省份"}</option>
        {current?.cities.map(option => <option key={option.id} value={option.name}>{option.name}</option>)}
      </Select>
    </label>
    {locations.isError ? (
      <div className="col-span-full flex flex-wrap items-center gap-2 text-sm">
        <span role="status">发货地选项暂时无法读取</span>
        <Button type="button" size="sm" variant="outline" disabled={locations.isFetching} onClick={() => void locations.refetch()}>重试读取发货地</Button>
      </div>
    ) : null}
  </>;
}
