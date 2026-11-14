package setup

func requiredServices(services map[string]any) map[string]string {
	required := map[string]string{}
	for name, raw := range services {
		v := raw.(map[string]any)
		if p, ok := v["profiles"].([]any); ok && len(p) > 0 {
			continue
		}
		required[name] = "running"
		if _, ok := v["healthcheck"].(map[string]any); ok {
			required[name] = "healthy"
		}
	}
	for name := range required {
		v := services[name].(map[string]any)
		if dependencies, ok := v["depends_on"].(map[string]any); ok {
			for dep, raw := range dependencies {
				if definition, ok := raw.(map[string]any); ok && definition["condition"] == "service_completed_successfully" {
					if _, exists := required[dep]; exists {
						required[dep] = "completed"
					}
				}
			}
		}
	}
	return required
}
func requiredHealthy(cs []container, required map[string]string) bool {
	if len(required) == 0 {
		return false
	}
	for name, kind := range required {
		c, ok := service(cs, name)
		if !ok {
			return false
		}
		switch kind {
		case "completed":
			if c.State.Status != "exited" || c.State.ExitCode != 0 {
				return false
			}
		case "healthy":
			if !healthy(c) || c.State.Health == nil {
				return false
			}
		case "running":
			if !healthy(c) {
				return false
			}
		default:
			return false
		}
	}
	return true
}
