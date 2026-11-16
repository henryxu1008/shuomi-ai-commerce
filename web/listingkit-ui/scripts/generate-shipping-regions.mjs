import { writeFileSync } from "node:fs";
import { areaList } from "@vant/area-data";

const cities = Object.entries(areaList.city_list);
const counties = Object.entries(areaList.county_list);
const provinces = Object.entries(areaList.province_list).map(([id, name]) => {
  const options = name.endsWith("市")
    ? [[id, name]]
    : cities.filter(([code]) => code.startsWith(id.slice(0, 2))).flatMap(([code, label]) =>
        label.includes("直辖县")
          ? counties.filter(([county]) => county.startsWith(code.slice(0, 4)))
          : [[code, label]],
      );
  return { id, name, cities: options.map(([id, name]) => ({ id, name })) };
});

const data = { id: "CN", name: "China", provinces, cities: [] };
writeFileSync(new URL("../../../internal/geography/shipping-cn.json", import.meta.url), JSON.stringify(data) + "\n");
