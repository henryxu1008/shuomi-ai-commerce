import { areaList } from "@vant/area-data";
import { Select } from "@/components/ui/select";

const cities = Object.entries(areaList.city_list);
const counties = Object.entries(areaList.county_list);
const provinces = Object.entries(areaList.province_list).map(([code, name]) => ({
  name,
  // Municipalities have one shipping city, not the dataset's extra "县" group.
  cities: name.endsWith("市") ? [name] : cities
    .filter(([cityCode]) => cityCode.startsWith(code.slice(0, 2)))
    .flatMap(([cityCode, cityName]) => cityName.includes("直辖县")
      ? counties.filter(([countyCode]) => countyCode.startsWith(cityCode.slice(0, 4))).map(([, countyName]) => countyName)
      : [cityName]),
}));

type Location = { province: string; city: string };

export function SupplyLocationFields({ province, city, onChange, required = true }: Location & {
  onChange: (location: Location) => void;
  required?: boolean;
}) {
  const current = provinces.find((option) => option.name === province);
  return <>
    <label>发货省份
      <Select required={required} value={province} onChange={(event) => onChange({ province: event.target.value, city: "" })}>
        <option value="">请选择省份</option>
        {provinces.map((option) => <option key={option.name} value={option.name}>{option.name}</option>)}
      </Select>
    </label>
    <label>发货城市
      <Select required={required} disabled={!current} value={current?.cities.includes(city) ? city : ""}
        onChange={(event) => onChange({ province, city: event.target.value })}>
        <option value="">{current ? "请选择城市" : "请先选择省份"}</option>
        {current?.cities.map((name) => <option key={name} value={name}>{name}</option>)}
      </Select>
    </label>
  </>;
}
