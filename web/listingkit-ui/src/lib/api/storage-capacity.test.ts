import {describe,it,expect} from "vitest";
import {parseStorageSummary,parseStorageOffers} from "./storage-capacity";
const sample=()=>({organization_id:"org",summary:{more_lots:false,base_bytes:"5368709120",quota_bytes:"5368709120",occupied_bytes:"2147483648",reserved_bytes:"268435456",remaining_bytes:"2952790016",overage_bytes:"0",categories:{product_media:"2147483648"},active_lots:[],observed_at:"2026-10-12T00:00:00Z"}});
describe("enterprise storage facts",()=>{
  it("rejects negative, rounded, inconsistent and unscoped snapshots",()=>{
    expect(parseStorageSummary(sample())).not.toBeNull();
    for(const invalid of ["-1","1.0",1,"9223372036854775808"]){expect(parseStorageSummary({...sample(),summary:{...sample().summary,reserved_bytes:invalid}})).toBeNull()}
    expect(parseStorageSummary({...sample(),summary:{...sample().summary,remaining_bytes:"0"}})).toBeNull();
    expect(parseStorageSummary({...sample(),summary:{...sample().summary,categories:{}}})).toBeNull();
    expect(parseStorageSummary({...sample(),organization_id:""})).toBeNull();
  });
  it("keeps occupied data visible after capacity expires",()=>{
    const v=sample();v.summary.occupied_bytes="21474836480";v.summary.categories.product_media=v.summary.occupied_bytes;v.summary.reserved_bytes="0";v.summary.remaining_bytes="0";v.summary.overage_bytes="16106127360";
    expect(parseStorageSummary(v)?.summary.occupied_bytes).toBe("21474836480");
  });
  it("does not synthesize an offer from missing or malformed catalog data",()=>{
    expect(parseStorageOffers({schema_version:"storage-price-catalog-v1",organization_id:"org",items:[]})).not.toBeNull();
    expect(parseStorageOffers({organization_id:"org",items:[{unit_price_minor:"1990"}]})).toBeNull();
  });
});
