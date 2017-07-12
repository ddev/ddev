package pantheon

import (
	"encoding/json"
	"fmt"
	"log"
)

// Site is a representation of a deployed pantheon site.
type Site struct {
	Archived    bool   `json:"archived"`
	ID          string `json:"id"`
	InvitedByID string `json:"invited_by_id"`
	Key         string `json:"key"`
	Role        string `json:"role"`
	Site        struct {
		Attributes struct {
			Label string `json:"label"`
		} `json:"attributes"`
		Created         int64  `json:"created"`
		CreatedByUserID string `json:"created_by_user_id"`
		Framework       string `json:"framework"`
		Frozen          bool   `json:"frozen"`
		Holder          struct {
			Email   string `json:"email"`
			ID      string `json:"id"`
			Profile struct {
				ActivityLevel   string `json:"activity_level"`
				Code            string `json:"code"`
				DashboardVisits []struct {
					Date string `json:"date"`
					Site string `json:"site"`
				} `json:"dashboard_visits"`
				Experiments                 struct{}    `json:"experiments"`
				Firstname                   string      `json:"firstname"`
				FullName                    string      `json:"full_name"`
				GoogleAdwordsPushedCodeSent int64       `json:"google_adwords_pushed_code_sent"`
				GuiltyOfAbuse               interface{} `json:"guilty_of_abuse"`
				InitialIdentityName         interface{} `json:"initial_identity_name"`
				InitialIdentityStrategy     interface{} `json:"initial_identity_strategy"`
				InvitesSent                 int64       `json:"invites_sent"`
				InvitesToSite               int64       `json:"invites_to_site"`
				InvitesToUser               int64       `json:"invites_to_user"`
				LastOrgSpinup               string      `json:"last-org-spinup"`
				Lastname                    string      `json:"lastname"`
				Maxdevsites                 int64       `json:"maxdevsites"`
				MinimizeJitDocs             bool        `json:"minimize_jit_docs"`
				Modified                    int64       `json:"modified"`
				Organization                string      `json:"organization"`
				Seens                       struct {
					NewSiteSupportInterface bool `json:"new-site-support-interface"`
				} `json:"seens"`
				TrackingFirstCodePush       int64 `json:"tracking_first_code_push"`
				TrackingFirstSiteCreate     int64 `json:"tracking_first_site_create"`
				TrackingFirstTeamInvite     int64 `json:"tracking_first_team_invite"`
				TrackingFirstWorkflowInLive int64 `json:"tracking_first_workflow_in_live"`
				Verify                      int64 `json:"verify"`
				WebServicesBusiness         bool  `json:"web_services_business"`
			} `json:"profile"`
		} `json:"holder"`
		HolderID     string `json:"holder_id"`
		HolderType   string `json:"holder_type"`
		ID           string `json:"id"`
		LastCodePush struct {
			Timestamp string      `json:"timestamp"`
			UserUUID  interface{} `json:"user_uuid"`
		} `json:"last_code_push"`
		LastFrozenAt  int64  `json:"last_frozen_at"`
		Name          string `json:"name"`
		Owner         string `json:"owner"`
		PreferredZone string `json:"preferred_zone"`
		Product       struct {
			ID       string `json:"id"`
			Longname string `json:"longname"`
		} `json:"product"`
		ProductID    string `json:"product_id"`
		ServiceLevel string `json:"service_level"`
		Upstream     struct {
			Branch    string `json:"branch"`
			ProductID string `json:"product_id"`
			URL       string `json:"url"`
		} `json:"upstream"`
		UpstreamUpdatesByEnvironment struct {
			HasCode bool `json:"has_code"`
		} `json:"upstream_updates_by_environment"`
		UserInCharge struct {
			Email   string `json:"email"`
			ID      string `json:"id"`
			Profile struct {
				ActivityLevel   string `json:"activity_level"`
				Code            string `json:"code"`
				DashboardVisits []struct {
					Date string `json:"date"`
					Site string `json:"site"`
				} `json:"dashboard_visits"`
				Experiments                 struct{}    `json:"experiments"`
				Firstname                   string      `json:"firstname"`
				FullName                    string      `json:"full_name"`
				GoogleAdwordsPushedCodeSent int64       `json:"google_adwords_pushed_code_sent"`
				GuiltyOfAbuse               interface{} `json:"guilty_of_abuse"`
				InitialIdentityName         interface{} `json:"initial_identity_name"`
				InitialIdentityStrategy     interface{} `json:"initial_identity_strategy"`
				InvitesSent                 int64       `json:"invites_sent"`
				InvitesToSite               int64       `json:"invites_to_site"`
				InvitesToUser               int64       `json:"invites_to_user"`
				LastOrgSpinup               string      `json:"last-org-spinup"`
				Lastname                    string      `json:"lastname"`
				Maxdevsites                 int64       `json:"maxdevsites"`
				MinimizeJitDocs             bool        `json:"minimize_jit_docs"`
				Modified                    int64       `json:"modified"`
				Organization                string      `json:"organization"`
				Seens                       struct {
					NewSiteSupportInterface bool `json:"new-site-support-interface"`
				} `json:"seens"`
				TrackingFirstCodePush       int64 `json:"tracking_first_code_push"`
				TrackingFirstSiteCreate     int64 `json:"tracking_first_site_create"`
				TrackingFirstTeamInvite     int64 `json:"tracking_first_team_invite"`
				TrackingFirstWorkflowInLive int64 `json:"tracking_first_workflow_in_live"`
				Verify                      int64 `json:"verify"`
				WebServicesBusiness         bool  `json:"web_services_business"`
			} `json:"profile"`
		} `json:"user_in_charge"`
		UserInChargeID string `json:"user_in_charge_id"`
	} `json:"site"`
	SiteID string `json:"site_id"`
	UserID string `json:"user_id"`
}

// SiteList represents a grouping of deployed Pantheon sites.
type SiteList struct {
	Sites []Site
}

// NewSiteList creates an SiteList. The user will be specified by which session you use to make the GET request. You are responsible for making the HTTP request.
func NewSiteList() *SiteList {
	return &SiteList{}
}

// Path returns the API endpoint which can be used to get a SiteList for the current user.
func (sl SiteList) Path(method string, auth AuthSession) string {
	userid, err := auth.GetUser()
	if err != nil {
		log.Fatalf("Could not determine user for request: %v", err)
	}

	return fmt.Sprintf("/users/%s/memberships/sites", userid)
}

// JSON prepares the SiteList for HTTP transport.
func (sl SiteList) JSON() ([]byte, error) {
	return json.Marshal(sl.Sites)
}

// Unmarshal is responsible for converting a HTTP response into a SiteList struct.
func (sl *SiteList) Unmarshal(data []byte) error {
	return json.Unmarshal(data, &sl.Sites)
}
