import { z } from "zod";
import { AccountReadError, accountErrorCode } from "./account";
import { readBoundedStrictJSON } from "./strict-json-response";

export const regionCatalogMaxBytes = 2 * 1024 * 1024;
const version = z.string().regex(/^gn-[a-f0-9]{16}$/);
export const countryCode = z.string().regex(/^[A-Z]{2}$/);
const geoId = z.string().regex(/^[1-9][0-9]{0,11}$/);
const name = z.string().min(1).max(128);
const option = z.object({ id: geoId, name }).strict();
const countryOption = z.object({ id: countryCode, name }).strict();

export const regionIndexSchema = z.object({
  version,
  countries: z.array(countryOption).max(300),
}).strict();
export const regionCountrySchema = z.object({
  version,
  country: countryOption.extend({
    provinces: z.array(option.extend({ cities: z.array(option) }).strict()),
    cities: z.array(option),
  }).strict(),
}).strict();
export const regionSelectionSchema = z.object({
  catalogVersion: version,
  country: z.union([countryCode, z.literal("")]),
  province: z.union([geoId, z.literal("direct"), z.literal("")]),
  city: z.union([geoId, z.literal("")]),
}).strict();

export type RegionSelection = Omit<z.infer<typeof regionSelectionSchema>, "catalogVersion">;
export type RegionOption = z.infer<typeof option>;
export type RegionCountry = z.infer<typeof regionCountrySchema>["country"];
export const emptyRegion: RegionSelection = { country: "", province: "", city: "" };

export function regionCities(country: RegionCountry | undefined, province: string) {
  if (!country) return [];
  if (!country.provinces.length || province === "direct") return country.cities;
  return country.provinces.find(option => option.id === province)?.cities ?? [];
}

export function validRegionSelection(selection: RegionSelection, country: RegionCountry | undefined) {
  if (!country || country.id !== selection.country) return false;
  const provinceValid = country.provinces.length
    ? country.provinces.some(province => province.id === selection.province) || (selection.province === "direct" && country.cities.length > 0)
    : selection.province === "";
  if (!provinceValid) return false;
  const cities = regionCities(country, selection.province);
  return cities.length ? cities.some(city => city.id === selection.city) : selection.city === "";
}

export function matchSavedRegion(labels: RegionSelection, country: RegionCountry | undefined) {
  if (!country || country.name !== labels.country) return null;
  const province = labels.province
    ? country.provinces.find(option => option.name === labels.province)?.id
    : country.provinces.length && country.cities.length ? "direct" : "";
  if (province === undefined) return null;
  const city = labels.city ? regionCities(country, province).find(option => option.name === labels.city)?.id : "";
  if (city === undefined) return null;
  const selection = { country: country.id, province, city };
  return validRegionSelection(selection, country) ? selection : null;
}

export function getRegionIndex(subject: string, signal: AbortSignal) {
  return request("", regionIndexSchema, subject, signal);
}

export async function getRegionCountry(subject: string, country: string, signal: AbortSignal) {
  if (!countryCode.safeParse(country).success) throw new AccountReadError(400, "INVALID_REQUEST");
  const detail = await request(`?country=${country}`, regionCountrySchema, subject, signal);
  if (detail.country.id !== country) throw new AccountReadError(502, "INVALID_UPSTREAM_RESPONSE");
  return detail;
}

export async function getShippingRegion(subject: string, signal: AbortSignal) {
  const detail = await request("?country=CN&view=shipping", regionCountrySchema, subject, signal);
  if (detail.country.id !== "CN" || detail.country.provinces.length === 0)
    throw new AccountReadError(502, "INVALID_UPSTREAM_RESPONSE");
  return detail;
}

async function request<T>(query: string, schema: z.ZodType<T>, subject: string, signal: AbortSignal) {
  const response = await fetch(`/api/account/region-catalog${query}`, {
    headers: { Accept: "application/json", "X-Expected-User-ID": subject },
    cache: "no-store",
    credentials: "same-origin",
    redirect: "error",
    signal,
  });
  const payload = await readBoundedStrictJSON(response, regionCatalogMaxBytes, signal);
  if (response.status !== 200) throw new AccountReadError(response.status, accountErrorCode(response.status, payload));
  const parsed = schema.safeParse(payload);
  if (!parsed.success) throw new AccountReadError(502, "INVALID_UPSTREAM_RESPONSE");
  return parsed.data;
}
