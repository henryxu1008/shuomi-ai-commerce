import type { RegionCountry } from "@/lib/api/region-catalog";

function division(id: string, name: string, cities: [string, string][]) {
  return { id, name, cities: cities.map(([id, name]) => ({ id, name })) };
}

export const shippingRegions: { version: string; country: RegionCountry } = {
  version: "gn-0123456789abcdef",
  country: {
    id: "CN",
    name: "China",
    provinces: [
      division("110000", "北京市", [["110000", "北京市"]]),
      division("310000", "上海市", [["310000", "上海市"]]),
      division("330000", "浙江省", [["330100", "杭州市"], ["330200", "宁波市"]]),
      division("420000", "湖北省", [["429004", "仙桃市"]]),
      division("440000", "广东省", [["440100", "广州市"], ["440300", "深圳市"]]),
      division("500000", "重庆市", [["500000", "重庆市"]]),
      division("650000", "新疆维吾尔自治区", [["659001", "石河子市"]]),
    ],
    cities: [],
  },
};
