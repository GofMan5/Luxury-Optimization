package main

import "time"

const (
	profileRecommended = "recommended"
	profileMaximum     = "maximum"
)

type Hardware struct {
	OS         OSInfo    `json:"os"`
	CPUs       []CPUInfo `json:"cpus"`
	GPUs       []GPUInfo `json:"gpus"`
	HasBattery bool      `json:"has_battery"`
	GOARCH     string    `json:"go_arch"`
}

type OSInfo struct {
	Caption      string `json:"caption"`
	Version      string `json:"version"`
	BuildNumber  string `json:"build_number"`
	Architecture string `json:"architecture"`
}

type CPUInfo struct {
	Name         string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	Cores        int    `json:"cores"`
	Logical      int    `json:"logical_processors"`
}

type GPUInfo struct {
	Name          string `json:"name"`
	PNPDeviceID   string `json:"pnp_device_id"`
	DriverVersion string `json:"driver_version"`
	Vendor        string `json:"vendor"`
}

type RegChange struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Hive        string `json:"hive"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	Kind        uint32 `json:"kind"`
	DWord       uint32 `json:"dword,omitempty"`
	String      string `json:"string,omitempty"`
}

type Profile struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	Changes         []RegChange `json:"changes"`
	PerformancePlan bool        `json:"performance_plan"`
	NetworkLatency  bool        `json:"network_latency"`
}

type RegSnapshot struct {
	Change  RegChange `json:"change"`
	Existed bool      `json:"existed"`
	Kind    uint32    `json:"kind,omitempty"`
	DWord   uint32    `json:"dword,omitempty"`
	QWord   uint64    `json:"qword,omitempty"`
	String  string    `json:"string,omitempty"`
	Strings []string  `json:"strings,omitempty"`
	Binary  []byte    `json:"binary,omitempty"`
}

type NetProperty struct {
	InterfaceGUID string   `json:"interface_guid"`
	AdapterName   string   `json:"adapter_name"`
	Keyword       string   `json:"keyword"`
	Values        []string `json:"values"`
}

type PowerSnapshot struct {
	PreviousGUID string         `json:"previous_guid,omitempty"`
	CreatedGUID  string         `json:"created_guid,omitempty"`
	Settings     []PowerSetting `json:"settings,omitempty"`
}

type PowerSetting struct {
	Subgroup string `json:"subgroup"`
	Setting  string `json:"setting"`
	Value    uint32 `json:"value"`
	Name     string `json:"-"`
}

type MouseSnapshot struct {
	Threshold1 int32 `json:"threshold_1"`
	Threshold2 int32 `json:"threshold_2"`
	Speed      int32 `json:"speed"`
	Captured   bool  `json:"captured"`
	Applied    bool  `json:"applied"`
}

type Backup struct {
	FormatVersion   int           `json:"format_version"`
	CatalogVersion  int           `json:"catalog_version"`
	CatalogDigest   string        `json:"catalog_digest"`
	ID              string        `json:"id"`
	CreatedAt       time.Time     `json:"created_at"`
	Profile         string        `json:"profile"`
	TargetUserSID   string        `json:"target_user_sid"`
	Status          string        `json:"status"`
	Registry        []RegSnapshot `json:"registry"`
	AppliedRegistry int           `json:"applied_registry"`
	Network         []NetProperty `json:"network,omitempty"`
	NetworkApplied  bool          `json:"network_applied"`
	Power           PowerSnapshot `json:"power"`
	PowerApplied    bool          `json:"power_applied"`
	Mouse           MouseSnapshot `json:"mouse"`
	Error           string        `json:"error,omitempty"`
	Path            string        `json:"-"`
}

type PlanItem struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Current  string `json:"current"`
	Desired  string `json:"desired"`
	Changed  bool   `json:"changed"`
}

type Plan struct {
	Profile  Profile    `json:"profile"`
	Hardware Hardware   `json:"hardware"`
	Items    []PlanItem `json:"items"`
	Warnings []string   `json:"warnings,omitempty"`
}

type Finding struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Evidence string `json:"evidence"`
	Action   string `json:"action"`
}

type Audit struct {
	GeneratedAt     time.Time `json:"generated_at"`
	Version         string    `json:"version"`
	Hardware        Hardware  `json:"hardware"`
	Administrator   bool      `json:"administrator"`
	ActivePowerGUID string    `json:"active_power_guid,omitempty"`
	Findings        []Finding `json:"optimization_findings"`
	Warnings        []string  `json:"warnings,omitempty"`
}

type OperationResult struct {
	Title      string
	Summary    string
	BackupPath string
	Err        error
}
