import {afterEach,expect,it,vi} from "vitest";
import {configuredAgentConfigurationOrigin} from "./agent-configuration-origin";
afterEach(()=>vi.unstubAllEnvs());
it.each(["http://backend:8080/api/v1","https://backend.invalid/api/v1/"])("uses the original BFF origin for %s",value=>{
 vi.stubEnv("LISTINGKIT_SERVICE_API_BASE",value);expect(configuredAgentConfigurationOrigin()).toBe(new URL(value).origin);
});
it.each(["","http://user:password@backend/api/v1","file:///api/v1","http://backend/wrong","https://backend/api/v1?token=value","https://backend/api/v1#part"])("does not advertise an invalid backend %s",value=>{
 vi.stubEnv("LISTINGKIT_SERVICE_API_BASE",value);expect(configuredAgentConfigurationOrigin()).toBeNull();
});
