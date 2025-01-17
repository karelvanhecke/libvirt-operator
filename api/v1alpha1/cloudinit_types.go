/*
Copyright 2024 Karel Van Hecke

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CloudInitMetadata struct {
	// +kubebuilder:validation:Required
	InstanceID string `json:"instanceID" yaml:"instance-id"`
	// +kubebuilder:validation:Required
	LocalHostname string `json:"localHostname" yaml:"local-hostname"`
}

type CloudInitUser struct {
	// +kubebuilder:validation:Required
	Name string `json:"name" yaml:"name"`
	// +kubebuilder:validation:Optional
	Doas []string `json:"doas,omitempty" yaml:"doas,omitempty"`
	// +kubebuilder:validation:Optional
	ExpireDate *string `json:"expireDate,omitempty" yaml:"expiredate,omitempty"`
	// +kubebuilder:validation:Optional
	Gecos *string `json:"gecos,omitempty" yaml:"gecos,omitempty"`
	// +kubebuilder:validation:Optional
	Groups *string `json:"groups,omitempty" yaml:"groups,omitempty"`
	// +kubebuilder:validation:Optional
	HomeDir *string `json:"homedir,omitempty" yaml:"homedir,omitempty"`
	// +kubebuilder:validation:Optional
	Intactive *string `json:"inactive,omitempty" yaml:"inactive,omitempty"`
	// +kubebuilder:validation:Optional
	LockPasswd *bool `json:"lockPassword,omitempty" yaml:"lock_password,omitempty"`
	// +kubebuilder:validation:Optional
	NoCreateHome *bool `json:"noCreateHome,omitempty" yaml:"no_create_home,omitempty"`
	// +kubebuilder:validation:Optional
	NoLogInit *bool `json:"noLogInit,omitempty" yaml:"no_log_init,omitempty"`
	// +kubebuilder:validation:Optional
	NoUserGroup *bool `json:"noUserGroup,omitempty" yaml:"no_user_group,omitempty"`
	// +kubebuilder:validation:Optional
	Passwd *string `json:"passwd,omitempty" yaml:"passwd,omitempty"`
	// +kubebuilder:validation:Optional
	HashedPasswd *string `json:"hashedPasswd,omitempty" yaml:"hashed_passwd,omitempty"`
	// +kubebuilder:validation:Optional
	PlaintextPasswd *string `json:"plainTextPasswd,omitempty" yaml:"plain_text_passwd,omitempty"`
	// +kubebuilder:validation:Optional
	CreateGroups *bool `json:"createGroups,omitempty" yaml:"create_groups,omitempty"`
	// +kubebuilder:validation:Optional
	PrimaryGroup *string `json:"primaryGroup,omitempty" yaml:"primary_group,omitempty"`
	// +kubebuilder:validation:Optional
	SELinuxUser *string `json:"selinuxUser,omitempty" yaml:"selinux_user,omitempty"`
	// +kubebuilder:validation:Optional
	Shell *string `json:"shell,omitempty" yaml:"shell,omitempty"`
	// +kubebuilder:validation:Optional
	SnapUser *string `json:"snapuser,omitempty" yaml:"snapuser,omitempty"`
	// +kubebuilder:validation:Optional
	SSHAuthorizedKeys []string `json:"sshAuthorizedKeys,omitempty" yaml:"ssh_authorized_keys,omitempty"`
	// +kubebuilder:validation:Optional
	SSHImportID []string `json:"sshImportID,omitempty" yaml:"ssh_import_id,omitempty"`
	// +kubebuilder:validation:Optional
	SSHRedirectUser *bool `json:"sshRedirectUser,omitempty" yaml:"ssh_redirect_user,omitempty"`
	// +kubebuilder:validation:Optional
	System *bool `json:"system,omitempty" yaml:"system,omitempty"`
	// +kubebuilder:validation:Optional
	Sudo []string `json:"sudo,omitempty" yaml:"sudo,omitempty"`
	// +kubebuilder:validation:Optional
	UID *int32 `json:"uid,omitempty" yaml:"uid,omitempty"`
}

type CloudInitWriteFileSource struct {
	// +kubebuilder:validation:Required
	URI string `json:"uri" yaml:"uri"`
	// +kubebuilder:validation:Optional
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.content) ? !has(self.source) : has(self.source)",message="either content or source needs to be set"
type CloudInitWriteFile struct {
	// +kubebuilder:validation:Required
	Path string `json:"path" yaml:"path"`
	// +kubebuilder:validation:Optional
	Content *string `json:"content,omitempty" yaml:"content,omitempty"`
	// +kubebuilder:validation:Optional
	Source *CloudInitWriteFileSource `json:"source,omitempty" yaml:"source,omitempty"`
	// +kubebuilder:validation:Optional
	Owner *string `json:"owner,omitempty" yaml:"owner,omitempty"`
	// +kubebuilder:validation:Optional
	Permissions *string `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	// +kubebuilder:validation:Optional
	Encoding *string `json:"encoding,omitempty" yaml:"encoding,omitempty"`
	// +kubebuilder:validation:Optional
	Append *bool `json:"append,omitempty" yaml:"append,omitempty"`
	// +kubebuilder:validation:Optional
	Defer *bool `json:"defer,omitempty" yaml:"defer,omitempty"`
}

type CloudInitGrowpart struct {
	// +kubebuilder:validation:Optional
	Mode *string `json:"mode,omitempty" yaml:"mode,omitempty"`
	// +kubebuilder:validation:Optional
	Devices []string `json:"devices,omitempty" yaml:"devices,omitempty"`
	// +kubebuilder:validation:Optional
	IgnoreGrowrootDisabled *bool `json:"ignoreGrowrootDisabled,omitempty" yaml:"ignore_growroot_disabled,omitempty"`
}

type CloudInitMountSwap struct {
	// +kubebuilder:validation:Optional
	Filename *string `json:"filename,omitempty" yaml:"filename,omitempty"`
	// +kubebuilder:validation:Optional
	Size *string `json:"size,omitempty" yaml:"size,omitempty"`
	// +kubebuilder:validation:Optional
	MaxSize *string `json:"maxsize,omitempty" yaml:"maxsize,omitempty"`
}

type CloudInitDisk struct {
	// +kubebuilder:validation:Optional
	TableType *string `json:"tableType,omitempty" yaml:"table_type,omitempty"`
	// +kubebuilder:validation:Optional
	Layout []string `json:"layout,omitempty" yaml:"layout,omitempty"`
	// +kubebuilder:validation:Optional
	Overwrite *bool `json:"overwrite,omitempty" yaml:"overwrite,omitempty"`
}

type CloudInitFS struct {
	// +kubebuilder:validation:Optional
	Label *string `json:"label,omitempty" yaml:"label,omitempty"`
	// +kubebuilder:validation:Required
	Filesystem string `json:"filesystem" yaml:"filesystem"`
	// +kubebuilder:validation:Required
	Device string `json:"device" yaml:"device"`
	// +kubebuilder:validation:Optional
	Partitions *string `json:"partition,omitempty" yaml:"partition,omitempty"`
	// +kubebuilder:validation:Optional
	Overwrite *bool `json:"overwrite,omitempty" yaml:"overwrite,omitempty"`
	// +kubebuilder:validation:Optional
	ReplaceFS *string `json:"replaceFs,omitempty" yaml:"replace_fs,omitempty"`
	// +kubebuilder:validation:Optional
	ExtraOpts []string `json:"extraOpts,omitempty" yaml:"extra_opts,omitempty"`
	// +kubebuilder:validation:Optional
	CMD []string `json:"cmd,omitempty" yaml:"cmd,omitempty"`
}

type CloudInitCACerts struct {
	// +kubebuilder:validation:Optional
	RemoveDefaults *bool `json:"removeDefaults,omitempty" yaml:"remove_defaults,omitempty"`
	// +kubebuilder:validation:Optional
	Trusted []string `json:"trusted,omitempty" yaml:"trusted,omitempty"`
}

type CloudInitNTPConfig struct {
	ConfPath *string `json:"confPath,omitempty" yaml:"confpath,omitempty"`
	// +kubebuilder:validation:Optional
	CheckEXE *string `json:"checkExe,omitempty" yaml:"check_exe,omitempty"`
	// +kubebuilder:validation:Optional
	Packages []string `json:"packages,omitempty" yaml:"packages,omitempty"`
	// +kubebuilder:validation:Optional
	ServiceName *string `json:"serviceName,omitempty" yaml:"service_name,omitempty"`
	// +kubebuilder:validation:Optional
	Template *string `json:"template,omitempty" yaml:"template,omitempty"`
}

type CloudInitNTP struct {
	// +kubebuilder:validation:Optional
	Pools []string `json:"pools,omitempty" yaml:"pools,omitempty"`
	// +kubebuilder:validation:Optional
	Servers []string `json:"servers,omitempty" yaml:"servers,omitempty"`
	// +kubebuilder:validation:Optional
	Peers []string `json:"peers,omitempty" yaml:"peers,omitempty"`
	// +kubebuilder:validation:Optional
	Allow []string `json:"allow,omitempty" yaml:"allow,omitempty"`
	// +kubebuilder:validation:Optional
	NTPClient *string `json:"ntpClient,omitempty" yaml:"ntp_client,omitempty"`
	// +kubebuilder:validation:Optional
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	// +kubebuilder:validation:Optional
	Config *CloudInitNTPConfig `json:"config,omitempty" yaml:"config,omitempty"`
}

type CloudInitKeyboard struct {
	// +kubebuilder:validation:Required
	Layout string `json:"layout" yaml:"layout"`
	// +kubebuilder:validation:Optional
	Model *string `json:"model,omitempty" yaml:"model,omitempty"`
	// +kubebuilder:validation:Optional
	Variant *string `json:"variant,omitempty" yaml:"variant,omitempty"`
	// +kubebuilder:validation:Optional
	Options *string `json:"options,omitempty" yaml:"options,omitempty"`
}

type CloudInitPhoneHome struct {
	// +kubebuilder:validation:Required
	URL string `json:"url" yaml:"url"`
	// +kubebuilder:validation:Optional
	Post []string `json:"post,omitempty" yaml:"post,omitempty"`
	// +kubebuilder:validation:Optional
	Tries *int32 `json:"tries,omitempty" yaml:"tries,omitempty"`
}

type CloudInitCloudConfig struct {
	// +kubebuilder:validation:Optional
	BootCMD [][]string `json:"bootcmd,omitempty" yaml:"bootcmd,omitempty"`
	// +kubebuilder:validation:Optional
	Groups map[string][]string `json:"groups,omitempty" yaml:"groups,omitempty"`
	// +kubebuilder:validation:Optional
	User *CloudInitUser `json:"user,omitempty" yaml:"user,omitempty"`
	// +kubebuilder:validation:Optional
	Users []CloudInitUser `json:"users,omitempty" yaml:"users,omitempty"`
	// +kubebuilder:validation:Optional
	WriteFiles []CloudInitWriteFile `json:"writeFiles,omitempty" yaml:"write_files,omitempty"`
	// +kubebuilder:validation:Optional
	RunCMD [][]string `json:"runcmd,omitempty" yaml:"runcmd,omitempty"`
	// +kubebuilder:validation:Optional
	Growpart *CloudInitGrowpart `json:"growpart,omitempty" yaml:"growpart,omitempty"`
	// +kubebuilder:validation:Optional
	Mounts [][]string `json:"mounts,omitempty" yaml:"mounts,omitempty"`
	// +kubebuilder:validation:Optional
	MountDefaultFields []string `json:"mountDefaultFields,omitempty" yaml:"mount_default_fields,omitempty"`
	// +kubebuilder:validation:Optional
	Swap *CloudInitMountSwap `json:"swap,omitempty" yaml:"swap,omitempty"`
	// +kubebuilder:validation:Optional
	DeviceAliases map[string]string `json:"deviceAliases,omitempty" yaml:"device_aliases,omitempty"`
	// +kubebuilder:validation:Optional
	DiskSetup map[string]CloudInitDisk `json:"diskSetup,omitempty" yaml:"disk_setup,omitempty"`
	// +kubebuilder:validation:Optional
	FSSetup []CloudInitFS `json:"fsSetup,omitempty" yaml:"fs_setup,omitempty"`
	// +kubebuilder:validation:Optional
	CACerts *CloudInitCACerts `json:"caCerts,omitempty" yaml:"ca_certs,omitempty"`
	// +kubebuilder:validation:Optional
	PreserveHostname *bool `json:"preserveHostname,omitempty" yaml:"preserve_hostname,omitempty"`
	// +kubebuilder:validation:Optional
	PreferFQDNOverHostname *bool `json:"preferFqdnOverHostname,omitempty" yaml:"prefer_fqdn_over_hostname,omitempty"`
	// +kubebuilder:validation:Optional
	CreateHostnameFile *bool `json:"createHostnameFile,omitempty" yaml:"create_hostname_file,omitempty"`
	// +kubebuilder:validation:Optional
	FQDN *string `json:"fqdn,omitempty" yaml:"fqdn,omitempty"`
	// +kubebuilder:validation:Optional
	Hostname *string `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	// +kubebuilder:validation:Optional
	Timezone *string `json:"timezone,omitempty" yaml:"timezone,omitempty"`
	// +kubebuilder:validation:Optional
	NTP *CloudInitNTP `json:"ntp,omitempty" yaml:"ntp,omitempty"`
	// +kubebuilder:validation:Optional
	Locale *string `json:"locale,omitempty" yaml:"locale,omitempty"`
	// +kubebuilder:validation:Optional
	LocaleConfigfile *string `json:"localeConfigFile,omitempty" yaml:"locale_config_file,omitempty"`
	// +kubebuilder:validation:Optional
	Keyboard *CloudInitKeyboard `json:"keyboard,omitempty" yaml:"keyboard,omitempty"`
	// +kubebuilder:validation:Optional
	PhoneHome *CloudInitPhoneHome `json:"phoneHome,omitempty" yaml:"phone_home,omitempty"`
}

type CloudInitDHCPOverrides struct {
	// +kubebuilder:validation:Optional
	UseDNS *bool `json:"useDNS,omitempty" yaml:"use-dns,omitempty"`
	// +kubebuilder:validation:Optional
	UseNTP *bool `json:"useNTP,omitempty" yaml:"use-ntp,omitempty"`
	// +kubebuilder:validation:Optional
	SendHostName *bool `json:"sendHostname,omitempty" yaml:"send-hostname,omitempty"`
	// +kubebuilder:validation:Optional
	UseMTU *bool `json:"useMTU,omitempty" yaml:"use-mtu,omitempty"`
	// +kubebuilder:validation:Optional
	Hostname *string `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	// +kubebuilder:validation:Optional
	UseRoutes *bool `json:"useRoutes,omitempty" yaml:"use-routes,omitempty"`
	// +kubebuilder:validation:Optional
	RouteMetric *int32 `json:"routeMetric,omitempty" yaml:"route-metric,omitempty"`
	// +kubebuilder:validation:Optional
	UseDomains *bool `json:"useDomains,omitempty" yaml:"use-domains,omitempty"`
}

type CloudInitNameservers struct {
	// +kubebuilder:validation:Optional
	Addresses []string `json:"addresses,omitempty" yaml:"addresses,omitempty"`
	// +kubebuilder:validation:Optional
	Search []string `json:"search,omitempty" yaml:"search,omitempty"`
}

type CloudInitRoute struct {
	// +kubebuilder:validation:Required
	To string `json:"to" yaml:"to"`
	// +kubebuilder:validation:Required
	Via string `json:"via" yaml:"via"`
	// +kubebuilder:validation:Optional
	Metric *int32 `json:"metric,omitempty" yaml:"metric,omitempty"`
}

type NetplanRAOverrides struct {
	// +kubebuilder:validation:Optional
	UseDNS *bool `json:"useDNS,omitempty" yaml:"use-dns,omitempty"`
	// +kubebuilder:validation:Optional
	UseDomains *string `json:"useDomains,omitempty" yaml:"use-domains,omitempty"`
	// +kubebuilder:validation:Optional
	Table *int32 `json:"table,omitempty" yaml:"table,omitempty"`
}

type NetplanPassthrough struct {
	// +kubebuilder:validation:Optional
	AcceptRA *bool `json:"acceptRA,omitempty" yaml:"accept-ra,omitempty"`
	// +kubebuilder:validation:Optional
	RAOverrides *NetplanRAOverrides `json:"raOverrides,omitempty" yaml:"ra-overrides,omitempty"`
}

type CloudInitNetworkProperties struct {
	// +kubebuilder:validation:Optional
	Renderer *string `json:"renderer,omitempty" yaml:"renderer,omitempty"`
	// +kubebuilder:validation:Optional
	DHCP4 *bool `json:"dhcp4,omitempty" yaml:"dhcp4,omitempty"`
	// +kubebuilder:validation:Optional
	DHCP6 *bool `json:"dhcp6,omitempty" yaml:"dhcp6,omitempty"`
	// +kubebuilder:validation:Optional
	DHCP4Overrides *CloudInitDHCPOverrides `json:"dhcp4Overrides,omitempty" yaml:"dhcp4-overrides,omitempty"`
	// +kubebuilder:validation:Optional
	DHCP6Overrides *CloudInitDHCPOverrides `json:"dhcp6Overrides,omitempty" yaml:"dhcp6-overrides,omitempty"`
	// +kubebuilder:validation:Optional
	Addresses []string `json:"addresses,omitempty" yaml:"addresses,omitempty"`
	// +kubebuilder:validation:Optional
	MTU *int32 `json:"mtu,omitempty" yaml:"mtu,omitempty"`
	// +kubebuilder:validation:Optional
	Optional *bool `json:"optional,omitempty" yaml:"optional,omitempty"`
	// +kubebuilder:validation:Optional
	Nameservers *CloudInitNameservers `json:"nameservers,omitempty" yaml:"nameservers,omitempty"`
	// +kubebuilder:validation:Optional
	Routes             []CloudInitRoute `json:"routes,omitempty" yaml:"routes,omitempty"`
	NetplanPassthrough `json:",inline" yaml:",inline"`
}

type CloudInitPhysicalProperties struct {
	// +kubebuilder:validation:Optional
	Match *CloudInitMatch `json:"match,omitempty" yaml:"match,omitempty"`
	// +kubebuilder:validation:Optional
	SetName *string `json:"setName,omitempty" yaml:"set-name,omitempty"`
	// +kubebuilder:validation:Optional
	WakeOnLAN *bool `json:"wakeOnLAN,omitempty" yaml:"wakeonlan,omitempty"`
}

type CloudInitMatch struct {
	// +kubebuilder:validation:Optional
	Name *string `json:"name,omitempty" yaml:"name,omitempty"`
	// +kubebuilder:validation:Optional
	MacAddress *string `json:"macAddress,omitempty" yaml:"macaddress,omitempty"`
	// +kubebuilder:validation:Optional
	Driver *string `json:"driver,omitempty" yaml:"driver,omitempty"`
}

type CloudInitEthernet struct {
	CloudInitPhysicalProperties `json:",inline" yaml:",inline"`
	CloudInitNetworkProperties  `json:",inline" yaml:",inline"`
}

type CloudInitBondParameters struct {
	// +kubebuilder:validation:Optional
	Mode *string `json:"mode,omitempty" yaml:"mode,omitempty"`
	// +kubebuilder:validation:Optional
	LACPRate *string `json:"lacpRate,omitempty" yaml:"lacp-rate,omitempty"`
	// +kubebuilder:validation:Optional
	MIIMonitorInterval *int32 `json:"miiMonitorInterval,omitempty" yaml:"mii-monitor-interval,omitempty"`
	// +kubebuilder:validation:Optional
	MinLinks *int32 `json:"minLinks,omitempty" yaml:"min-links,omitempty"`
	// +kubebuilder:validation:Optional
	TransmitHashPolicy *string `json:"transmitHashPolicy,omitempty" yaml:"transmit-hash-policy,omitempty"`
	// +kubebuilder:validation:Optional
	ADSelect *string `json:"adSelect,omitempty" yaml:"ad-select,omitempty"`
	// +kubebuilder:validation:Optional
	AllSlavesActive *bool `json:"allSlavesActive,omitempty" yaml:"all-slaves-active,omitempty"`
	// +kubebuilder:validation:Optional
	ARPInterval *int32 `json:"arpInterval,omitempty" yaml:"arp-interval,omitempty"`
	// +kubebuilder:validation:Optional
	ARPIPTargets []string `json:"arpIPTargets,omitempty" yaml:"arp-ip-targets,omitempty"`
	// +kubebuilder:validation:Optional
	ARPValidate *string `json:"arpValidate,omitempty" yaml:"arp-validate,omitempty"`
	// +kubebuilder:validation:Optional
	ARPAllTargets *string `json:"arpAllTargets,omitempty" yaml:"arp-all-targets,omitempty"`
	// +kubebuilder:validation:Optional
	UpDelay *int32 `json:"upDelay,omitempty" yaml:"up-delay,omitempty"`
	// +kubebuilder:validation:Optional
	DownDelay *int32 `json:"downDelay,omitempty" yaml:"down-delay,omitempty"`
	// +kubebuilder:validation:Optional
	FailOverMACPolicy *string `json:"failOverMacPolicy,omitempty" yaml:"fail-over-mac-policy,omitempty"`
	// +kubebuilder:validation:Optional
	GratuitousARP *int32 `json:"gratuitousArp,omitempty" yaml:"gratuitous-arp,omitempty"`
	// +kubebuilder:validation:Optional
	PacketsPerSlave *int32 `json:"packetsPerSlave,omitempty" yaml:"packets-per-slave,omitempty"`
	// +kubebuilder:validation:Optional
	PrimaryReselectPolicy *string `json:"primaryReselectPolicy,omitempty" yaml:"primary-reselect-policy,omitempty"`
	// +kubebuilder:validation:Optional
	LearnPacketInterval *int32 `json:"learnPacketInterval,omitempty" yaml:"learn-packet-interval,omitempty"`
}

type CloudInitBond struct {
	CloudInitNetworkProperties `json:",inline" yaml:",inline"`
	// +kubebuilder:validation:Required
	Interfaces []string `json:"interfaces" yaml:"interfaces"`
	// +kubebuilder:validation:Optional
	Parameters *CloudInitBondParameters `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type CloudInitBridgeParameters struct {
	// +kubebuilder:validation:Optional
	AgeingTime *int32 `json:"ageingTime,omitempty" yaml:"ageing-time,omitempty"`
	// +kubebuilder:validation:Optional
	Priority *int32 `json:"priority,omitempty" yaml:"priority,omitempty"`
	// +kubebuilder:validation:Optional
	ForwardDelay *int32 `json:"forwardDelay,omitempty" yaml:"forward-delay,omitempty"`
	// +kubebuilder:validation:Optional
	HelloTime *int32 `json:"helloTime,omitempty" yaml:"hello-time,omitempty"`
	// +kubebuilder:validation:Optional
	MaxAge *int32 `json:"maxAge,omitempty" yaml:"max-age,omitempty"`
	// +kubebuilder:validation:Optional
	PathCost *int32 `json:"pathCost,omitempty" yaml:"path-cost,omitempty"`
	// +kubebuilder:validation:Optional
	STP *bool `json:"stp,omitempty" yaml:"stp,omitempty"`
}

type CloudInitBridge struct {
	CloudInitNetworkProperties `json:",inline" yaml:",inline"`
	// +kubebuilder:validation:Required
	Interfaces []string `json:"interfaces" yaml:"interfaces"`
	// +kubebuilder:validation:Optional
	Parameters *CloudInitBridgeParameters `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type CloudInitVLAN struct {
	CloudInitNetworkProperties `json:",inline" yaml:",inline"`
	// +kubebuilder:validation:Required
	ID int32 `json:"id" yaml:"id"`
	// +kubebuilder:validation:Required
	Link string `json:"link" yaml:"link"`
}

type CloudInitNetworkConfig struct {
	// +kubebuilder:validation:Optional
	Ethernets map[string]CloudInitEthernet `json:"ethernets,omitempty" yaml:"ethernets,omitempty"`
	// +kubebuilder:validation:Optional
	Bonds map[string]CloudInitBond `json:"bonds,omitempty" yaml:"bonds,omitempty"`
	// +kubebuilder:validation:Optional
	Bridges map[string]CloudInitBridge `json:"bridges,omitempty" yaml:"bridges,omitempty"`
	// +kubebuilder:validation:Optional
	VLANs map[string]CloudInitVLAN `json:"vlans,omitempty" yaml:"vlans,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="can not change existing cloud init data store"
type CloudInitSpec struct {
	// +kubebuilder:validation:Required
	PoolRef ResourceRef `json:"poolRef"`
	// +kubebuilder:validation:Optional
	Metadata *CloudInitMetadata `json:"metadata,omitempty"`
	// +kubebuilder:validation:Optional
	UserData *CloudInitCloudConfig `json:"userData,omitempty"`
	// +kubebuilder:validation:Optional
	VendorData *CloudInitCloudConfig `json:"vendorData,omitempty"`
	// +kubebuilder:validation:Optional
	NetworkConfig *CloudInitNetworkConfig `json:"networkConfig,omitempty"`
}

// +kubebuilder:validation:Optional
type CloudInitStatus struct {
	Pool       string             `json:"pool,omitempty"`
	Host       string             `json:"host,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type CloudInit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudInitSpec   `json:"spec,omitempty"`
	Status CloudInitStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type CloudInitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []CloudInit `json:"items"`
}

func (ci *CloudInit) ResourceName() string {
	return ci.Name + ".cidata.iso"
}

func init() {
	SchemeBuilder.Register(&CloudInit{}, &CloudInitList{})
}
