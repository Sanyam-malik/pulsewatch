package uptimekuma

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func TestReadAndApplyImportsPublicMonitorsAndStatusPage(t *testing.T) {
	ctx := context.Background()
	source := testDB(t)
	target := testDB(t)
	createSourceSchema(t, source)
	createTargetSchema(t, target)
	_, err := source.Exec(`INSERT INTO monitor(id,name,type,active,interval,timeout,maxretries,retry_interval,resend_interval,url,method,accepted_statuscodes_json)
		VALUES(1,'Website','http',1,60,10,2,10,3,'https://example.com/health','GET','["200-299"]')`)
	require.NoError(t, err)
	_, err = source.Exec(`INSERT INTO status_page(id,slug,title,description,theme,published,footer_text)
		VALUES(7,'public','Public status','All good','dark',1,'Contact us');
		INSERT INTO "group"(id,name,public,status_page_id) VALUES(9,'Services',1,7);
		INSERT INTO monitor_group(monitor_id,group_id,weight) VALUES(1,9,1);
		INSERT INTO heartbeat(id,monitor_id,status) VALUES(1,1,1);
		INSERT INTO incident(id,title) VALUES(1,'Incident')`)
	require.NoError(t, err)

	plan, err := Read(ctx, source, Options{})
	require.NoError(t, err)
	require.Len(t, plan.Monitors, 1)
	require.Equal(t, "http", plan.Monitors[0].Type)
	require.True(t, plan.Monitors[0].Active)
	require.Equal(t, 16, plan.Monitors[0].Timeout)
	require.Equal(t, 20, plan.Monitors[0].RetryInterval)
	require.Len(t, plan.StatusPages, 1)
	require.False(t, plan.StatusPages[0].Published)
	require.Equal(t, []string{plan.Monitors[0].ID}, plan.StatusPages[0].MonitorIDs)
	require.Equal(t, 1, plan.Summary.Monitors)
	require.Equal(t, 1, plan.Summary.StatusPages)
	require.NotEmpty(t, plan.Summary.Warnings)
	require.NoError(t, ValidateTarget(ctx, target, plan))
	require.NoError(t, Apply(ctx, target, plan, Options{}))

	var monitorCount, pageCount, membershipCount int
	require.NoError(t, target.QueryRow("SELECT count(*) FROM monitors WHERE active=0").Scan(&monitorCount))
	require.NoError(t, target.QueryRow("SELECT count(*) FROM status_pages WHERE published=0").Scan(&pageCount))
	require.NoError(t, target.QueryRow("SELECT count(*) FROM monitor_status_pages").Scan(&membershipCount))
	require.Equal(t, 1, monitorCount)
	require.Equal(t, 1, pageCount)
	require.Equal(t, 1, membershipCount)
}

func TestReadSkipsCredentialsAndChannelsByDefault(t *testing.T) {
	ctx := context.Background()
	source := testDB(t)
	createSourceSchema(t, source)
	_, err := source.Exec(`INSERT INTO monitor(id,name,type,url,headers) VALUES(1,'Secure site','http','https://example.com','{"Authorization":"secret"}');
		INSERT INTO notification(id,name,config) VALUES(2,'Telegram','{"type":"telegram","telegramBotToken":"secret","telegramChatID":"42"}')`)
	require.NoError(t, err)
	plan, err := Read(ctx, source, Options{})
	require.NoError(t, err)
	require.Empty(t, plan.Monitors)
	require.Empty(t, plan.Channels)
	require.Len(t, plan.Summary.Skipped, 2)
	for _, item := range plan.Summary.Skipped {
		require.NotContains(t, item.Reason, "bot-secret")
		require.NotContains(t, item.Name, "bot-secret")
	}
	encoded, err := json.Marshal(plan.Summary)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "bot-secret")
	require.NotContains(t, string(encoded), "Authorization")
}

func TestExplicitChannelImportIsInactiveAndLinked(t *testing.T) {
	ctx := context.Background()
	source := testDB(t)
	target := testDB(t)
	createSourceSchema(t, source)
	createTargetSchema(t, target)
	_, err := source.Exec(`INSERT INTO monitor(id,name,type,active,interval,timeout,url,method)
		VALUES(1,'Website','http',1,60,16,'https://example.com','GET');
		INSERT INTO notification(id,name,active,is_default,config)
		VALUES(2,'Telegram',1,1,'{"type":"telegram","telegramBotToken":"bot-secret","telegramChatID":"42","telegramTemplate":"template-secret"}');
		INSERT INTO monitor_notification(monitor_id,notification_id) VALUES(1,2)`)
	require.NoError(t, err)
	plan, err := Read(ctx, source, Options{IncludeSecrets: true, Activate: true})
	require.NoError(t, err)
	require.Len(t, plan.Channels, 1)
	require.Len(t, plan.Monitors, 1)
	require.Len(t, plan.Monitors[0].ChannelIDs, 1)
	summaryJSON, err := json.Marshal(plan.Summary)
	require.NoError(t, err)
	require.NotContains(t, string(summaryJSON), "bot-secret")
	require.NotContains(t, string(summaryJSON), "template-secret")
	require.Contains(t, string(summaryJSON), "not mapped")
	require.NoError(t, ValidateTarget(ctx, target, plan))
	require.NoError(t, Apply(ctx, target, plan, Options{IncludeSecrets: true, Activate: true}))

	var channelActive, monitorActive bool
	var config string
	require.NoError(t, target.QueryRow("SELECT active,config FROM notification_channels").Scan(&channelActive, &config))
	require.False(t, channelActive)
	require.Contains(t, config, "bot-secret")
	require.NoError(t, target.QueryRow("SELECT active FROM monitors").Scan(&monitorActive))
	require.True(t, monitorActive)
	var links int
	require.NoError(t, target.QueryRow("SELECT count(*) FROM monitor_notifications").Scan(&links))
	require.Equal(t, 1, links)
}

func TestValidateTargetSkipsExistingStatusPageSlug(t *testing.T) {
	ctx := context.Background()
	source := testDB(t)
	target := testDB(t)
	createSourceSchema(t, source)
	createTargetSchema(t, target)
	_, err := source.Exec(`INSERT INTO status_page(id,slug,title) VALUES(1,'same','Source page')`)
	require.NoError(t, err)
	_, err = target.Exec(`INSERT INTO status_pages(id,slug,title) VALUES('existing','same','Existing page')`)
	require.NoError(t, err)
	plan, err := Read(ctx, source, Options{})
	require.NoError(t, err)
	require.NoError(t, ValidateTarget(ctx, target, plan))
	require.Empty(t, plan.StatusPages)
	require.Contains(t, plan.Summary.Skipped[0].Reason, "already exists")
}

func TestReadSupportsMappedMonitorTypes(t *testing.T) {
	ctx := context.Background()
	source := testDB(t)
	createSourceSchema(t, source)
	_, err := source.Exec(`INSERT INTO monitor(id,name,type,hostname,port,url,keyword,invert_keyword,dns_resolve_server,dns_resolve_type)
		VALUES(1,'Port check','port','example.com',443,'','',0,'',''),
		      (2,'Ping check','ping','example.com',0,'','',0,'',''),
		      (3,'DNS check','dns','example.com',0,'','',0,'1.1.1.1','A'),
		      (4,'Keyword check','keyword','',0,'https://example.com','ready',1,'',''),
		      (5,'Unsupported','docker','',0,'','',0,'','')`)
	require.NoError(t, err)
	plan, err := Read(ctx, source, Options{})
	require.NoError(t, err)
	require.Len(t, plan.Monitors, 4)
	types := make(map[string]bool)
	for _, monitor := range plan.Monitors {
		types[monitor.Type] = true
		var config map[string]any
		require.NoError(t, json.Unmarshal([]byte(monitor.Config), &config))
		switch monitor.Type {
		case "tcp":
			require.EqualValues(t, 443, config["port"])
		case "dns":
			require.Equal(t, "1.1.1.1", config["resolver_server"])
		case "http":
			if monitor.Name == "Keyword check" {
				require.Equal(t, "ready", config["keyword"])
				require.Equal(t, true, config["invert_keyword"])
			}
		}
	}
	require.True(t, types["http"])
	require.True(t, types["tcp"])
	require.True(t, types["ping"])
	require.True(t, types["dns"])
	require.Len(t, plan.Summary.Skipped, 1)
	require.Contains(t, plan.Summary.Skipped[0].Reason, "not supported")
}

func TestApplyRollsBackAllImportedRowsOnFailure(t *testing.T) {
	ctx := context.Background()
	source := testDB(t)
	target := testDB(t)
	createSourceSchema(t, source)
	createTargetSchema(t, target)
	_, err := source.Exec(`INSERT INTO monitor(id,name,type,url,method) VALUES(1,'Website','http','https://example.com','GET');
		INSERT INTO notification(id,name,config) VALUES(2,'Telegram','{"type":"telegram","telegramBotToken":"token","telegramChatID":"42"}')`)
	require.NoError(t, err)
	plan, err := Read(ctx, source, Options{IncludeSecrets: true})
	require.NoError(t, err)
	require.NoError(t, ValidateTarget(ctx, target, plan))
	_, err = target.Exec(`CREATE TRIGGER reject_monitor BEFORE INSERT ON monitors BEGIN SELECT RAISE(ABORT,'test failure'); END`)
	require.NoError(t, err)
	err = Apply(ctx, target, plan, Options{IncludeSecrets: true})
	require.Error(t, err)
	var channels int
	require.NoError(t, target.QueryRow("SELECT count(*) FROM notification_channels").Scan(&channels))
	require.Zero(t, channels)
}

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open(sqliteshim.ShimName, ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func createSourceSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TABLE monitor(id INTEGER, name TEXT, type TEXT, active BOOLEAN, interval INTEGER, timeout INTEGER, maxretries INTEGER, retry_interval INTEGER, resend_interval INTEGER, url TEXT, method TEXT, accepted_statuscodes_json TEXT, headers TEXT, hostname TEXT, port INTEGER, packet_size INTEGER, keyword TEXT, ignore_tls BOOLEAN, invert_keyword BOOLEAN, auth_method TEXT, dns_resolve_server TEXT, dns_resolve_type TEXT);
		CREATE TABLE notification(id INTEGER, name TEXT, active BOOLEAN, is_default BOOLEAN, config TEXT);
		CREATE TABLE monitor_notification(monitor_id INTEGER, notification_id INTEGER);
		CREATE TABLE status_page(id INTEGER, slug TEXT, title TEXT, description TEXT, icon TEXT, theme TEXT, published BOOLEAN, password TEXT, footer_text TEXT);
		CREATE TABLE "group"(id INTEGER, name TEXT, public BOOLEAN, status_page_id INTEGER);
		CREATE TABLE monitor_group(monitor_id INTEGER, group_id INTEGER, weight INTEGER);
		CREATE TABLE heartbeat(id INTEGER, monitor_id INTEGER, status INTEGER);
		CREATE TABLE incident(id INTEGER, title TEXT);
	`)
	require.NoError(t, err)
}

func createTargetSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TABLE monitors(id TEXT PRIMARY KEY,type TEXT,name TEXT,interval INTEGER,timeout INTEGER,max_retries INTEGER,retry_interval INTEGER,resend_interval INTEGER,active BOOLEAN,status INTEGER,created_at TIMESTAMP,updated_at TIMESTAMP,config TEXT,proxy_id TEXT,push_token TEXT);
		CREATE TABLE notification_channels(id TEXT PRIMARY KEY,name TEXT,type TEXT,active BOOLEAN,is_default BOOLEAN,config TEXT,created_at TIMESTAMP,updated_at TIMESTAMP);
		CREATE TABLE monitor_notifications(id TEXT PRIMARY KEY,monitor_id TEXT,notification_channel_id TEXT,created_at TIMESTAMP,updated_at TIMESTAMP);
		CREATE TABLE status_pages(id TEXT PRIMARY KEY,slug TEXT UNIQUE,title TEXT,description TEXT,icon TEXT,theme TEXT,published BOOLEAN,created_at TIMESTAMP,updated_at TIMESTAMP,footer_text TEXT,auto_refresh_interval INTEGER);
		CREATE TABLE monitor_status_pages(id TEXT PRIMARY KEY,monitor_id TEXT,status_page_id TEXT,created_at TIMESTAMP,updated_at TIMESTAMP);
	`)
	require.NoError(t, err)
}
