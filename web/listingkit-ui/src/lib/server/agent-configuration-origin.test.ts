import {afterEach,expect,it,vi} from "vitest";
import {configuredAgentConfigurationOrigin,isAgentConfigurationAvailable} from "./agent-configuration-origin";
afterEach(()=>vi.unstubAllEnvs());
it.each(["http://backend:8080/api/v1","https://backend.invalid/api/v1/"])("uses the original BFF origin for %s",value=>{
 vi.stubEnv("LISTINGKIT_SERVICE_API_BASE",value);expect(configuredAgentConfigurationOrigin()).toBe(new URL(value).origin);
});
it.each(["","http://user:password@backend/api/v1","file:///api/v1","http://backend/wrong","https://backend/api/v1?token=value","https://backend/api/v1#part"])("does not advertise an invalid backend %s",value=>{
 vi.stubEnv("LISTINGKIT_SERVICE_API_BASE",value);expect(configuredAgentConfigurationOrigin()).toBeNull();
});

it.each([undefined,"false","TRUE"])("does not advertise an unmounted module with flag %s",value=>{
 vi.stubEnv("LISTINGKIT_SERVICE_API_BASE","http://backend:8080/api/v1");
 vi.stubEnv("LISTINGKIT_AGENT_CONFIGURATION_ENABLED",value);
 expect(isAgentConfigurationAvailable()).toBe(false);
});
it("requires a trusted origin even when the module is declared mounted",()=>{
 vi.stubEnv("LISTINGKIT_AGENT_CONFIGURATION_ENABLED","true");vi.stubEnv("LISTINGKIT_SERVICE_API_BASE","");expect(isAgentConfigurationAvailable()).toBe(false);
 vi.stubEnv("LISTINGKIT_SERVICE_API_BASE","http://backend:8080/api/v1");expect(isAgentConfigurationAvailable()).toBe(true);
});
