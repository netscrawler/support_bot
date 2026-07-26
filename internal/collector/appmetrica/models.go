package appmetrica

import "time"

type ReportDataResponse struct {
	Query struct {
		Ids                     []int    `json:"ids"`
		Dimensions              []string `json:"dimensions"`
		Metrics                 []string `json:"metrics"`
		Sort                    []string `json:"sort"`
		Date1                   string   `json:"date1"`
		Date2                   string   `json:"date2"`
		Filters                 string   `json:"filters"`
		Limit                   int      `json:"limit"`
		Offset                  int      `json:"offset"`
		Group                   string   `json:"group"`
		Currency                string   `json:"currency"`
		BenchmarksVersion       string   `json:"benchmarks_version"`
		BenchmarksSubIndustry   string   `json:"benchmarks_sub_industry"`
		PcaIntegerIntervals     string   `json:"pca_integer_intervals"`
		AutoGroupSize           string   `json:"auto_group_size"`
		PcaIntervalsLength      string   `json:"pca_intervals_length"`
		InactivityWindow        string   `json:"inactivity_window"`
		InstallationAttribution string   `json:"installation_attribution"`
		FunnelWindow            string   `json:"funnel_window"`
		EventAttribution        string   `json:"event_attribution"`
		InappRevenueType        string   `json:"inapp_revenue_type"`
		Quantile                string   `json:"quantile"`
		From                    string   `json:"from"`
		To                      string   `json:"to"`
		FunnelPattern           string   `json:"funnel_pattern"`
		FunnelRestriction       string   `json:"funnel_restriction"`
		ProfileAttributeId      string   `json:"profile_attribute_id"`
	} `json:"query"`
	Data []struct {
		Dimensions []struct {
			Name    string `json:"name"`
			Comment any    `json:"comment"`
		} `json:"dimensions"`
		Metrics []float64 `json:"metrics"`
	} `json:"data"`
	TotalRows             int       `json:"total_rows"`
	TotalRowsRounded      bool      `json:"total_rows_rounded"`
	Sampled               bool      `json:"sampled"`
	ContainsSensitiveData bool      `json:"contains_sensitive_data"`
	SampleShare           float64   `json:"sample_share"`
	SampleSize            int       `json:"sample_size"`
	SampleSpace           int       `json:"sample_space"`
	DataLag               int       `json:"data_lag"`
	Totals                []float64 `json:"totals"`
	Min                   []float64 `json:"min"`
	Max                   []float64 `json:"max"`
}

type GetMyApplicationResponse struct {
	Applications []struct {
		Name                                 string   `json:"name"`
		TimeZoneName                         string   `json:"time_zone_name"`
		OrganizationId                       int      `json:"organization_id"`
		HideAddress                          bool     `json:"hide_address"`
		GdprAgreementAccepted                bool     `json:"gdpr_agreement_accepted"`
		Category                             int      `json:"category,omitempty"`
		UseUniversalLinks                    bool     `json:"use_universal_links"`
		PartialExportEnabled                 bool     `json:"partial_export_enabled"`
		RedownloadExclusionWindowSeconds     int      `json:"redownload_exclusion_window_seconds"`
		RedownloadInactivityWindowSeconds    int      `json:"redownload_inactivity_window_seconds"`
		ReattributionEnabled                 bool     `json:"reattribution_enabled"`
		ReattributionInactivityWindowSeconds int      `json:"reattribution_inactivity_window_seconds"`
		ReengagementInactivityWindowSeconds  int      `json:"reengagement_inactivity_window_seconds"`
		MetrikaCounters                      []any    `json:"metrika_counters"`
		Id                                   int      `json:"id"`
		Uid                                  int      `json:"uid"`
		OwnerLogin                           string   `json:"owner_login"`
		Permission                           string   `json:"permission"`
		Features                             []string `json:"features"`
		OrganizationFeatures                 []struct {
			Name string `json:"name"`
		} `json:"organization_features"`
		VendorData     bool      `json:"vendor_data"`
		TimeZoneOffset int       `json:"time_zone_offset"`
		PermissionDate time.Time `json:"permission_date"`
		CreateDate     string    `json:"create_date"`
	} `json:"applications"`
}

type SupportedApplications struct {
	AppName string
	ID      int
}
