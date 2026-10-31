package config

type ListingKitConfig struct {
	RolePolicyDatabase *DatabaseConfig         `mapstructure:"rolePolicyDatabase" yaml:"rolePolicyDatabase"`
	PlatformAdminUsers []string                `mapstructure:"platformAdminUsers" yaml:"platformAdminUsers"`
	PlatformAdminRoles []string                `mapstructure:"platformAdminRoles" yaml:"platformAdminRoles"`
	Zitadel            ListingKitZitadelConfig `mapstructure:"zitadel" yaml:"zitadel"`
}

type ListingKitZitadelConfig struct {
	IssuerURL             string `mapstructure:"issuerURL" yaml:"issuerURL"`
	AuthorizationAPIURL   string `mapstructure:"authorizationAPIURL" yaml:"authorizationAPIURL"`
	ClientID              string `mapstructure:"clientID" yaml:"clientID"`
	ClientSecret          string `mapstructure:"clientSecret" yaml:"clientSecret"`
	TenantDirectoryToken  string `mapstructure:"tenantDirectoryToken" yaml:"tenantDirectoryToken"`
	MemberInvitationToken string `mapstructure:"memberInvitationToken" yaml:"memberInvitationToken"`
	ProjectID             string `mapstructure:"projectID" yaml:"projectID"`
	// AllowedUsernames is retained only to detect and reject obsolete configuration.
	// It must never participate in authorization decisions.
	AllowedUsernames                  []string `mapstructure:"allowedUsernames" yaml:"allowedUsernames"`
	LegacyUsernameAllowlistConfigured bool     `mapstructure:"-" yaml:"-"`
}
