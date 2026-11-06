import {expect,it} from "vitest";
import {commercialResourcesFixture} from "@/test/fixtures/commercial-resources";
import {parseCommercialResources} from "./commercial-billing";
it("retains an exact fractional AI balance and its declared unit",()=>{const value=commercialResourcesFixture();value.resources[1]={...value.resources[1],unit_version:"ai-micropoint-v1",available:"9007199254.740993",reserved:"0.000012",consumed:"0.000003"} as typeof value.resources[1];expect(parseCommercialResources(value)?.resources[1].available).toBe("9007199254.740993")});

it("rejects a missing or different AI quantity unit instead of guessing",()=>{const value=commercialResourcesFixture();for(const unit of [undefined,"point","ai-micropoint-v2"]){expect(parseCommercialResources({...value,resources:value.resources.map(row=>row.resource_type==="ai_point"?{...row,unit_version:unit}:row)})).toBeNull()}});
