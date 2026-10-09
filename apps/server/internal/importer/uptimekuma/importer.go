package uptimekuma

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Options controls intentionally explicit, potentially disruptive import behavior.
type Options struct {
	IncludeSecrets bool
	Activate       bool
	PublishPages   bool
}

type Record struct {
	Kind     string `json:"kind"`
	SourceID string `json:"source_id,omitempty"`
	Name     string `json:"name,omitempty"`
	Reason   string `json:"reason"`
}

type Summary struct {
	Monitors    int      `json:"monitors"`
	Channels    int      `json:"channels"`
	StatusPages int      `json:"status_pages"`
	Skipped     []Record `json:"skipped"`
	Warnings    []string `json:"warnings"`
}

// Plan contains the converted entities and safe-to-print results. Config data is
// deliberately private and never included in the JSON summary.
type Plan struct {
	Monitors    []Monitor
	Channels    []Channel
	StatusPages []StatusPage
	Summary     Summary
}

type Monitor struct {
	ID             string
	SourceID       string
	Name           string
	Type           string
	Interval       int
	Timeout        int
	MaxRetries     int
	RetryInterval  int
	ResendInterval int
	Active         bool
	Status         int
	Config         string
	ChannelIDs     []string
}

type Channel struct {
	ID        string
	SourceID  string
	Name      string
	Type      string
	Config    string
	IsDefault bool
}

type StatusPage struct {
	ID          string
	SourceID    string
	Slug        string
	Title       string
	Description string
	Icon        string
	Theme       string
	Published   bool
	FooterText  string
	AutoRefresh int
	MonitorIDs  []string
}

// Read builds a write-free import plan from an Uptime Kuma SQLite database.
// The source handle should be opened read-only by the caller.
func Read(ctx context.Context, source *sql.DB, options Options) (*Plan, error) {
	p := &Plan{Summary: Summary{Skipped: []Record{}, Warnings: []string{}}}
	monitors, err := readTable(ctx, source, "monitor", true)
	if err != nil {
		return nil, err
	}
	notifications, err := readTable(ctx, source, "notification", false)
	if err != nil {
		return nil, err
	}
	monitorNotifications, err := readTable(ctx, source, "monitor_notification", false)
	if err != nil {
		return nil, err
	}
	statusPages, err := readTable(ctx, source, "status_page", false)
	if err != nil {
		return nil, err
	}
	groups, err := readTable(ctx, source, "group", false)
	if err != nil {
		return nil, err
	}
	monitorGroups, err := readTable(ctx, source, "monitor_group", false)
	if err != nil {
		return nil, err
	}

	if len(statusPages) == 0 && len(monitors) > 0 {
		p.Summary.Warnings = append(p.Summary.Warnings, "No status_page table or rows were found; status pages cannot be imported")
	}
	if hasTable(ctx, source, "heartbeat") {
		p.Summary.Warnings = append(p.Summary.Warnings, "Historical heartbeat records are not imported; monitors will start with fresh Pulsewatch history")
	}
	if hasTable(ctx, source, "incident") {
		p.Summary.Warnings = append(p.Summary.Warnings, "Uptime Kuma status-page incidents are not imported")
	}
	if hasTable(ctx, source, "maintenance") {
		p.Summary.Warnings = append(p.Summary.Warnings, "Uptime Kuma maintenance schedules are not imported")
	}
	if hasTable(ctx, source, "status_page_cname") {
		p.Summary.Warnings = append(p.Summary.Warnings, "Uptime Kuma status-page custom domains are not imported")
	}
	if hasTable(ctx, source, "tag") {
		p.Summary.Warnings = append(p.Summary.Warnings, "Uptime Kuma tags are not imported")
	}

	channelIDs := map[string]string{}
	for _, row := range notifications {
		sourceID := text(row, "id")
		name := text(row, "name")
		if truth(row["is_default"]) {
			p.skip("notification setting", sourceID, name, "default-channel behavior is not preserved; imported as non-default")
		}
		channel, unmapped, reason := convertChannel(row, options)
		if reason != "" {
			p.skip("notification channel", sourceID, name, reason)
			continue
		}
		p.Channels = append(p.Channels, channel)
		channelIDs[sourceID] = channel.ID
		for _, setting := range unmapped {
			p.skip("notification setting", sourceID, name, "provider setting "+setting+" is not mapped")
		}
	}

	links := make(map[string][]string)
	for _, row := range monitorNotifications {
		monitorID := text(row, "monitor_id")
		channelID := text(row, "notification_id")
		if targetID, ok := channelIDs[channelID]; ok {
			links[monitorID] = append(links[monitorID], targetID)
		} else {
			p.skip("monitor notification link", monitorID, "", "referenced channel was unsupported or skipped")
		}
	}

	monitorIDs := map[string]string{}
	for _, row := range monitors {
		sourceID, name := text(row, "id"), text(row, "name")
		monitor, reason := convertMonitor(row)
		if reason != "" {
			p.skip("monitor", sourceID, name, reason)
			for range unique(links[sourceID]) {
				p.skip("monitor notification link", sourceID, name, "monitor was unsupported or skipped")
			}
			continue
		}
		if integer(row["interval"]) > 0 && integer(row["interval"]) < 20 ||
			integer(row["timeout"]) < 16 ||
			integer(row["retry_interval"]) < 20 {
			p.warn("Pulsewatch minimum interval, timeout, and retry-interval values are applied when source values are lower")
		}
		for _, setting := range unmappedMonitorSettings(row) {
			p.skip("monitor setting", sourceID, name, "monitor setting "+setting+" is not mapped")
		}
		monitor.ChannelIDs = unique(links[sourceID])
		p.Monitors = append(p.Monitors, monitor)
		monitorIDs[sourceID] = monitor.ID
	}

	publicGroups := make(map[string]bool)
	existingGroups := make(map[string]bool)
	groupPages := make(map[string]string)
	for _, row := range groups {
		groupID := text(row, "id")
		existingGroups[groupID] = true
		groupPages[groupID] = text(row, "status_page_id")
		if truth(row["public"]) {
			publicGroups[groupID] = true
		}
	}
	sourcePages := make(map[string]bool)
	for _, row := range statusPages {
		sourcePages[text(row, "id")] = true
	}
	if len(monitorGroups) > 0 {
		p.warn("Uptime Kuma monitor/group ordering is not retained; Pulsewatch's status-page membership relation has no order field")
	}
	pageMonitors := make(map[string][]string)
	for _, row := range monitorGroups {
		groupID := text(row, "group_id")
		monitorID := text(row, "monitor_id")
		if !existingGroups[groupID] {
			p.skip("status page monitor link", monitorID, "", "referenced source group does not exist")
			continue
		}
		if !publicGroups[groupID] {
			p.skip("private monitor group membership", monitorID, "", "private group membership is not made public on the imported status page")
			continue
		}
		pageID := groupPages[groupID]
		if pageID == "" || !sourcePages[pageID] {
			p.skip("status page monitor link", monitorID, "", "public group is not attached to an existing source status page")
			continue
		}
		if monitorIDs[monitorID] != "" {
			pageMonitors[pageID] = append(pageMonitors[pageID], monitorIDs[monitorID])
		} else {
			p.skip("status page monitor link", monitorID, "", "referenced monitor was unsupported or skipped")
		}
	}

	seenSlugs := make(map[string]bool)
	for _, row := range statusPages {
		sourceID, slug, title := text(row, "id"), text(row, "slug"), text(row, "title")
		if len(strings.TrimSpace(slug)) < 3 || len(strings.TrimSpace(title)) < 3 {
			p.skip("status page", sourceID, title, "slug and title must each be at least 3 characters")
			continue
		}
		if utf8.RuneCountInString(slug) > 255 || utf8.RuneCountInString(title) > 255 {
			p.skip("status page", sourceID, title, "slug or title exceeds Pulsewatch's 255-character limit")
			continue
		}
		if seenSlugs[strings.ToLower(slug)] {
			p.skip("status page", sourceID, title, "duplicate source slug")
			continue
		}
		seenSlugs[strings.ToLower(slug)] = true
		published := truth(row["published"]) && options.PublishPages
		if truth(row["published"]) && !options.PublishPages {
			p.skip("status page setting", sourceID, title, "source page was published; imported unpublished (use --publish-pages to publish)")
		}
		if strings.TrimSpace(text(row, "password")) != "" {
			published = false
			p.skip("status page setting", sourceID, title, "page password is not imported; page forced unpublished")
		}
		for _, setting := range unmappedStatusPageSettings(row) {
			p.skip("status page setting", sourceID, title, "status-page setting "+setting+" is not mapped")
		}
		p.StatusPages = append(p.StatusPages, StatusPage{
			ID: uuid.NewString(), SourceID: sourceID, Slug: slug, Title: title,
			Description: text(row, "description"), Icon: text(row, "icon"),
			Theme: defaultString(text(row, "theme"), "light"), Published: published,
			FooterText:  text(row, "footer_text"),
			AutoRefresh: defaultInt(integer(row["auto_refresh_interval"]), 300),
			MonitorIDs:  unique(pageMonitors[sourceID]),
		})
	}

	p.Summary.Monitors = len(p.Monitors)
	p.Summary.Channels = len(p.Channels)
	p.Summary.StatusPages = len(p.StatusPages)
	if !options.Activate && len(p.Monitors) > 0 {
		p.warn("Imported monitors will be inactive unless --activate is supplied")
	}
	if len(p.Channels) > 0 {
		p.warn("Imported notification channels are always inactive and must be reviewed and enabled in Pulsewatch")
	}
	return p, nil
}

// ValidateTarget checks the destination schema and status-page slug collisions
// before any writes are attempted.
func ValidateTarget(ctx context.Context, target *sql.DB, p *Plan) error {
	requiredColumns := map[string][]string{
		"monitors":              {"id", "type", "name", "interval", "timeout", "max_retries", "retry_interval", "resend_interval", "active", "status", "created_at", "updated_at", "config", "proxy_id", "push_token"},
		"notification_channels": {"id", "name", "type", "active", "is_default", "config", "created_at", "updated_at"},
		"monitor_notifications": {"id", "monitor_id", "notification_channel_id", "created_at", "updated_at"},
		"status_pages":          {"id", "slug", "title", "description", "icon", "theme", "published", "created_at", "updated_at", "footer_text", "auto_refresh_interval"},
		"monitor_status_pages":  {"id", "monitor_id", "status_page_id", "created_at", "updated_at"},
	}
	for table, required := range requiredColumns {
		columns, err := tableColumns(ctx, target, table)
		if err != nil {
			return fmt.Errorf("inspect target table %q: %w", table, err)
		}
		for _, column := range required {
			if !columns[column] {
				return fmt.Errorf("target table %q is missing required column %q; run Pulsewatch migrations first", table, column)
			}
		}
	}
	filteredPages := make([]StatusPage, 0, len(p.StatusPages))
	for _, page := range p.StatusPages {
		var count int
		if err := target.QueryRowContext(ctx, "SELECT count(*) FROM status_pages WHERE lower(slug)=lower(?)", page.Slug).Scan(&count); err != nil {
			return fmt.Errorf("check target status-page slug: %w", err)
		}
		if count != 0 {
			p.skip("status page", page.SourceID, page.Title, "slug already exists in target")
			continue
		}
		filteredPages = append(filteredPages, page)
	}
	p.StatusPages = filteredPages
	p.Summary.StatusPages = len(p.StatusPages)
	return nil
}

// Apply writes a previously parsed and validated plan in one SQLite transaction.
func Apply(ctx context.Context, target *sql.DB, p *Plan, options Options) error {
	tx, err := target.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	for _, channel := range p.Channels {
		if _, err := tx.ExecContext(ctx, `INSERT INTO notification_channels
			(id,name,type,active,is_default,config,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`,
			channel.ID, channel.Name, channel.Type, false, channel.IsDefault, channel.Config, now, now); err != nil {
			return fmt.Errorf("write notification channel %q: %w", channel.Name, err)
		}
	}
	for _, monitor := range p.Monitors {
		active := monitor.Active && options.Activate
		if _, err := tx.ExecContext(ctx, `INSERT INTO monitors
			(id,type,name,interval,timeout,max_retries,retry_interval,resend_interval,active,status,created_at,updated_at,config,proxy_id,push_token)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			monitor.ID, monitor.Type, monitor.Name, monitor.Interval, monitor.Timeout,
			monitor.MaxRetries, monitor.RetryInterval, monitor.ResendInterval, active,
			monitor.Status, now, now, monitor.Config, nil, ""); err != nil {
			return fmt.Errorf("write monitor %q: %w", monitor.Name, err)
		}
		for _, channelID := range monitor.ChannelIDs {
			if _, err := tx.ExecContext(ctx, `INSERT INTO monitor_notifications
				(id,monitor_id,notification_channel_id,created_at,updated_at) VALUES(?,?,?,?,?)`,
				uuid.NewString(), monitor.ID, channelID, now, now); err != nil {
				return fmt.Errorf("associate notification channel with monitor %q: %w", monitor.Name, err)
			}
		}
	}
	for _, page := range p.StatusPages {
		if _, err := tx.ExecContext(ctx, `INSERT INTO status_pages
			(id,slug,title,description,icon,theme,published,created_at,updated_at,footer_text,auto_refresh_interval)
			VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			page.ID, page.Slug, page.Title, page.Description, page.Icon, page.Theme,
			page.Published, now, now, page.FooterText, page.AutoRefresh); err != nil {
			return fmt.Errorf("write status page %q: %w", page.Title, err)
		}
		for i, monitorID := range page.MonitorIDs {
			if _, err := tx.ExecContext(ctx, `INSERT INTO monitor_status_pages
				(id,monitor_id,status_page_id,created_at,updated_at) VALUES(?,?,?,?,?)`,
				uuid.NewString(), monitorID, page.ID, now, now); err != nil {
				return fmt.Errorf("associate monitor with status page %q (position %d): %w", page.Title, i, err)
			}
		}
	}
	return tx.Commit()
}

func (p *Plan) skip(kind, id, name, reason string) {
	p.Summary.Skipped = append(p.Summary.Skipped, Record{Kind: kind, SourceID: id, Name: name, Reason: reason})
}

func (p *Plan) warn(message string) {
	for _, warning := range p.Summary.Warnings {
		if warning == message {
			return
		}
	}
	p.Summary.Warnings = append(p.Summary.Warnings, message)
}

func convertMonitor(row map[string]any) (Monitor, string) {
	name := strings.TrimSpace(text(row, "name"))
	if utf8.RuneCountInString(name) < 3 {
		return Monitor{}, "monitor name is missing or shorter than Pulsewatch's 3-character minimum"
	}
	if utf8.RuneCountInString(name) > 150 {
		return Monitor{}, "monitor name exceeds Pulsewatch's 150-character limit"
	}
	sourceType := strings.ToLower(text(row, "type"))
	targetType := sourceType
	config := map[string]any{}

	sensitiveFields := []string{
		"headers", "body", "basic_auth_user", "basic_auth_pass", "oauth_client_id",
		"oauth_client_secret", "bearer_token", "database_connection_string",
		"mqtt_username", "mqtt_password", "radius_password", "radius_secret",
		"tls_ca", "tls_cert", "tls_key", "ssh_username", "ssh_password", "ssh_private_key",
		"ssh_passphrase", "grpc_metadata", "grpc_body", "push_token", "kafka_producer_sasl_options",
	}
	hasSensitive := false
	for _, field := range sensitiveFields {
		if strings.TrimSpace(text(row, field)) != "" {
			hasSensitive = true
			break
		}
	}
	if hasSensitive {
		return Monitor{}, "contains credentials or request data with no supported safe field mapping"
	}
	if integer(row["proxy_id"]) > 0 {
		return Monitor{}, "uses a Uptime Kuma proxy; proxy settings are not imported"
	}
	if strings.TrimSpace(text(row, "auth_method")) != "" &&
		!strings.EqualFold(strings.TrimSpace(text(row, "auth_method")), "none") {
		return Monitor{}, "Uptime Kuma monitor authentication mode has no safe equivalent in this importer"
	}

	switch sourceType {
	case "http", "keyword":
		rawURL := text(row, "url")
		parsed, err := url.ParseRequestURI(rawURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return Monitor{}, "HTTP monitor has an invalid or unsupported URL"
		}
		if parsed.User != nil || parsed.RawQuery != "" {
			return Monitor{}, "URL contains user-info or query parameters that may contain credentials"
		}
		statusCodes := []string{"2XX"}
		rawCodes := text(row, "accepted_statuscodes_json")
		if rawCodes != "" {
			var sourceCodes []string
			if err := json.Unmarshal([]byte(rawCodes), &sourceCodes); err != nil {
				return Monitor{}, "accepted status-code list is malformed"
			}
			if len(sourceCodes) == 0 {
				return Monitor{}, "accepted status-code list is empty"
			}
			var converted []string
			for _, sourceCode := range sourceCodes {
				code := regexp.MustCompile(`^([1-5])00-([1-5])99$`).FindStringSubmatch(sourceCode)
				if len(code) != 3 || code[1] != code[2] {
					return Monitor{}, "accepted status-code range is not representable by Pulsewatch"
				}
				converted = append(converted, code[1]+"XX")
			}
			if len(converted) > 0 {
				statusCodes = unique(converted)
			}
		}
		method := strings.ToUpper(defaultString(text(row, "method"), "GET"))
		switch method {
		case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS":
		default:
			return Monitor{}, "HTTP method is unsupported by Pulsewatch"
		}
		config = map[string]any{
			"url": rawURL, "method": method, "headers": "", "encoding": "json",
			"accepted_statuscodes": statusCodes, "authMethod": "none",
			"ignore_tls_errors": truth(row["ignore_tls"]),
			"max_redirects":     max(integer(row["maxredirects"]), 0),
			"check_cert_expiry": truth(row["expiry_notification"]),
		}
		if sourceType == "keyword" {
			keyword := text(row, "keyword")
			if keyword == "" {
				return Monitor{}, "keyword monitor has no keyword to check"
			}
			config["keyword"] = keyword
			config["invert_keyword"] = truth(row["invert_keyword"])
		}
		targetType = "http"
	case "port":
		host, port := text(row, "hostname"), integer(row["port"])
		if host == "" || port < 1 || port > 65535 {
			return Monitor{}, "port monitor is missing a valid host or port"
		}
		targetType = "tcp"
		config = map[string]any{"host": host, "port": port}
	case "ping":
		host := text(row, "hostname")
		if host == "" {
			host = text(row, "url")
		}
		if host == "" {
			return Monitor{}, "ping monitor is missing a host"
		}
		targetType = "ping"
		packetSize := defaultInt(integer(row["packet_size"]), 32)
		if packetSize > 65507 {
			return Monitor{}, "ping packet size exceeds Pulsewatch's 65507-byte limit"
		}
		config = map[string]any{"host": host, "packet_size": packetSize}
	case "dns":
		host := text(row, "hostname")
		if host == "" {
			host = text(row, "url")
		}
		resolver := text(row, "dns_resolve_server")
		if resolver == "" {
			resolver = "1.1.1.1"
		}
		if host == "" || net.ParseIP(resolver) == nil {
			return Monitor{}, "DNS monitor needs a host and an IP address resolver"
		}
		resolveType := strings.ToUpper(defaultString(text(row, "dns_resolve_type"), "A"))
		if !strings.Contains("|A|AAAA|CAA|CNAME|MX|NS|PTR|SOA|SRV|TXT|", "|"+resolveType+"|") {
			return Monitor{}, "DNS record type is unsupported"
		}
		targetType = "dns"
		config = map[string]any{"host": host, "resolver_server": resolver, "port": 53, "resolve_type": resolveType}
	default:
		return Monitor{}, "monitor type is not supported"
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return Monitor{}, "could not encode converted monitor configuration"
	}
	interval := defaultInt(integer(row["interval"]), 60)
	timeout := defaultInt(integer(row["timeout"]), 16)
	retry := integer(row["retry_interval"])
	return Monitor{
		ID: uuid.NewString(), SourceID: text(row, "id"), Name: name, Type: targetType,
		Interval: max(interval, 20), Timeout: max(timeout, 16),
		MaxRetries:    max(integer(row["maxretries"]), 0),
		RetryInterval: max(retry, 20), ResendInterval: max(integer(row["resend_interval"]), 0),
		Active: truth(row["active"]), Status: 2, Config: string(configJSON),
	}, ""
}

func unmappedMonitorSettings(row map[string]any) []string {
	var settings []string
	if text(row, "description") != "" {
		settings = append(settings, "description")
	}
	if truth(row["domain_expiry_notification"]) {
		settings = append(settings, "domain_expiry_notification")
	}
	sourceType := strings.ToLower(text(row, "type"))
	if truth(row["expiry_notification"]) && sourceType != "http" && sourceType != "keyword" {
		settings = append(settings, "expiry_notification")
	}
	if integer(row["maxredirects"]) > 0 && sourceType != "http" && sourceType != "keyword" {
		settings = append(settings, "maxredirects")
	}
	sort.Strings(settings)
	return settings
}

func unmappedStatusPageSettings(row map[string]any) []string {
	var settings []string
	if text(row, "custom_css") != "" {
		settings = append(settings, "custom_css")
	}
	if text(row, "google_analytics_tag_id") != "" {
		settings = append(settings, "google_analytics_tag_id")
	}
	if !truth(row["search_engine_index"]) && row["search_engine_index"] != nil {
		settings = append(settings, "search_engine_index")
	}
	if truth(row["show_tags"]) {
		settings = append(settings, "show_tags")
	}
	if !truth(row["show_powered_by"]) && row["show_powered_by"] != nil {
		settings = append(settings, "show_powered_by")
	}
	sort.Strings(settings)
	return settings
}

func convertChannel(row map[string]any, options Options) (Channel, []string, string) {
	sourceID, name := text(row, "id"), text(row, "name")
	raw := text(row, "config")
	var source map[string]any
	if err := json.Unmarshal([]byte(raw), &source); err != nil {
		return Channel{}, nil, "notification configuration is missing or malformed"
	}
	sourceType := strings.ToLower(strings.TrimSpace(text(source, "type")))
	if sourceType == "" {
		return Channel{}, nil, "notification type is missing"
	}
	if utf8.RuneCountInString(name) > 255 {
		return Channel{}, nil, "notification channel name exceeds Pulsewatch's 255-character limit"
	}
	fields := make(map[string]any, len(source))
	for key, value := range source {
		fields[normalizeKey(key)] = value
	}
	out := map[string]any{}
	switch sourceType {
	case "discord":
		setFrom(out, fields, "webhook_url", "discordwebhookurl", "webhookurl")
		setFrom(out, fields, "bot_display_name", "discordusername")
		setFrom(out, fields, "custom_message_prefix", "discordprefixmessage")
		sourceMode := strings.ToLower(fmt.Sprint(valueFrom(fields, "discordchanneltype")))
		switch sourceMode {
		case "posttothread":
			out["message_type"] = "send_to_thread"
		case "createnewforumpost":
			out["message_type"] = "send_to_new_forum_post"
		case "posttochannel", "", "<nil>":
			out["message_type"] = "send_to_channel"
		default:
			return Channel{}, nil, "Discord webhook mode is not supported"
		}
		setFrom(out, fields, "thread_id", "threadid")
		setFrom(out, fields, "thread_name", "postname")
		if sourceMode == "posttothread" && !nonEmpty(out["thread_id"]) {
			return Channel{}, nil, "Discord thread mode is missing its thread ID"
		}
		if sourceMode == "createnewforumpost" && !nonEmpty(out["thread_name"]) {
			return Channel{}, nil, "Discord forum mode is missing its thread name"
		}
	case "slack":
		setFrom(out, fields, "slack_webhook_url", "slackwebhookurl", "webhookurl")
		setFrom(out, fields, "slack_username", "slackusername")
		setFrom(out, fields, "slack_channel", "slackchannel")
		setFrom(out, fields, "slack_icon_emoji", "slackiconemo", "slackiconemoji")
		setFrom(out, fields, "slack_rich_message", "slackrichmessage")
		setFrom(out, fields, "slack_channel_notify", "slackchannelnotify")
	case "telegram":
		setFrom(out, fields, "bot_token", "telegrambottoken")
		setFrom(out, fields, "chat_id", "telegramchatid")
		setFrom(out, fields, "message_thread_id", "telegrammessagethreadid")
		setFrom(out, fields, "server_url", "telegramserverurl")
		setFrom(out, fields, "send_silently", "telegramsendsilently")
		setFrom(out, fields, "protect_content", "telegramprotectcontent")
	case "gotify":
		setFrom(out, fields, "server_url", "gotifyserverurl", "gotifyurl")
		setFrom(out, fields, "application_token", "gotifyapplicationtoken", "gotifytoken")
		if priority := valueFrom(fields, "gotifypriority"); priority != nil {
			out["priority"] = integer(priority)
		}
	case "webhook":
		setFrom(out, fields, "webhook_url", "webhookurl")
		contentType := "json"
		if v := valueFrom(fields, "webhookcontenttype"); v != nil {
			contentType = strings.ToLower(fmt.Sprint(v))
		}
		if contentType == "form-data" {
			contentType = "form-data"
		} else if contentType == "custom" {
			contentType = "custom"
		} else {
			contentType = "json"
		}
		out["webhook_content_type"] = contentType
		setFrom(out, fields, "webhook_custom_body", "webhookcustombody")
		setFrom(out, fields, "webhook_additional_headers", "webhookadditionalheaders")
	default:
		return Channel{}, nil, "notification provider has no supported Pulsewatch mapping"
	}
	if !requiredChannelFields(sourceType, out) {
		return Channel{}, nil, "notification config is missing required provider fields or cannot be mapped"
	}
	if !options.IncludeSecrets {
		return Channel{}, nil, "channel credentials are not imported by default (use --include-secrets to opt in)"
	}
	configJSON, err := json.Marshal(out)
	if err != nil {
		return Channel{}, nil, "could not encode converted channel configuration"
	}
	recognized := map[string]bool{"type": true}
	for _, alias := range supportedSourceKeys(sourceType) {
		recognized[normalizeKey(alias)] = true
	}
	var unmapped []string
	for key, value := range fields {
		if !recognized[key] && nonEmpty(value) {
			unmapped = append(unmapped, key)
		}
	}
	sort.Strings(unmapped)
	return Channel{ID: uuid.NewString(), SourceID: sourceID, Name: defaultString(name, sourceType), Type: sourceType,
		Config: string(configJSON), IsDefault: false}, unmapped, ""
}

func supportedSourceKeys(kind string) []string {
	switch kind {
	case "discord":
		return []string{"discordWebhookUrl", "webhookUrl", "discordUsername", "discordPrefixMessage",
			"discordChannelType", "threadId", "postName"}
	case "slack":
		return []string{"slackWebhookUrl", "webhookUrl", "slackUsername", "slackChannel", "slackIconEmo",
			"slackIconEmoji", "slackRichMessage", "slackChannelNotify"}
	case "telegram":
		return []string{"telegramBotToken", "telegramChatID", "telegramMessageThreadID", "telegramServerUrl",
			"telegramSendSilently", "telegramProtectContent"}
	case "gotify":
		return []string{"gotifyServerUrl", "gotifyUrl", "gotifyApplicationToken", "gotifyToken", "gotifyPriority"}
	case "webhook":
		return []string{"webhookURL", "webhookContentType", "webhookCustomBody", "webhookAdditionalHeaders"}
	default:
		return nil
	}
}

func requiredChannelFields(kind string, fields map[string]any) bool {
	switch kind {
	case "discord":
		return validHTTPURL(fields["webhook_url"])
	case "slack":
		return validHTTPURL(fields["slack_webhook_url"])
	case "telegram":
		return nonEmpty(fields["bot_token"]) && nonEmpty(fields["chat_id"]) &&
			(fields["server_url"] == nil || validHTTPURL(fields["server_url"]))
	case "gotify":
		if !validHTTPURL(fields["server_url"]) || !nonEmpty(fields["application_token"]) {
			return false
		}
		if fields["priority"] != nil && (integer(fields["priority"]) < 0 || integer(fields["priority"]) > 10) {
			return false
		}
		return true
	case "webhook":
		if !validHTTPURL(fields["webhook_url"]) {
			return false
		}
		contentType := fmt.Sprint(fields["webhook_content_type"])
		if contentType != "json" && contentType != "form-data" && contentType != "custom" {
			return false
		}
		if contentType == "custom" && !nonEmpty(fields["webhook_custom_body"]) {
			return false
		}
		if nonEmpty(fields["webhook_additional_headers"]) {
			var headers map[string]any
			if json.Unmarshal([]byte(fmt.Sprint(fields["webhook_additional_headers"])), &headers) != nil {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func validHTTPURL(value any) bool {
	if !nonEmpty(value) {
		return false
	}
	parsed, err := url.ParseRequestURI(fmt.Sprint(value))
	return err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func setFrom(out, in map[string]any, target string, aliases ...string) {
	if value := valueFrom(in, aliases...); value != nil {
		out[target] = value
	}
}

func valueFrom(fields map[string]any, aliases ...string) any {
	for _, alias := range aliases {
		if v, ok := fields[normalizeKey(alias)]; ok {
			return v
		}
	}
	return nil
}

func normalizeKey(value string) string {
	value = strings.ToLower(value)
	return strings.NewReplacer("_", "", "-", "", " ", "").Replace(value)
}

func readTable(ctx context.Context, db *sql.DB, table string, required bool) ([]map[string]any, error) {
	if !hasTable(ctx, db, table) {
		if required {
			return nil, fmt.Errorf("source does not contain the required Uptime Kuma table %q", table)
		}
		return nil, nil
	}
	rows, err := db.QueryContext(ctx, `SELECT * FROM "`+table+`"`)
	if err != nil {
		return nil, fmt.Errorf("read source table %q: %w", table, err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("read source table %q row: %w", table, err)
		}
		row := make(map[string]any, len(columns))
		for i, column := range columns {
			if b, ok := values[i].([]byte); ok {
				row[strings.ToLower(column)] = string(b)
			} else {
				row[strings.ToLower(column)] = values[i]
			}
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func hasTable(ctx context.Context, db *sql.DB, name string) bool {
	var count int
	return db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&count) == nil && count > 0
}

func tableColumns(ctx context.Context, db *sql.DB, table string) (map[string]bool, error) {
	var exists int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, fmt.Errorf("missing table")
	}
	rows, err := db.QueryContext(ctx, `PRAGMA table_info("`+table+`")`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return nil, err
		}
		columns[strings.ToLower(name)] = true
	}
	return columns, rows.Err()
}

func text(row map[string]any, key string) string {
	value := row[strings.ToLower(key)]
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func integer(value any) int {
	if value == nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprint(value)))
	return n
}

func truth(value any) bool {
	switch strings.ToLower(strings.TrimSpace(fmt.Sprint(value))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func nonEmpty(value any) bool {
	return value != nil && strings.TrimSpace(fmt.Sprint(value)) != ""
}

func unique(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func defaultInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}
