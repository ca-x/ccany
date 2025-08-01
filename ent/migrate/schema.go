// Code generated by ent, DO NOT EDIT.

package migrate

import (
	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/schema/field"
)

var (
	// AppConfigsColumns holds the columns for the "app_configs" table.
	AppConfigsColumns = []*schema.Column{
		{Name: "id", Type: field.TypeString, Unique: true},
		{Name: "key", Type: field.TypeString, Unique: true},
		{Name: "value", Type: field.TypeString, Size: 2147483647},
		{Name: "category", Type: field.TypeString, Default: "general"},
		{Name: "type", Type: field.TypeString, Default: "string"},
		{Name: "description", Type: field.TypeString, Nullable: true},
		{Name: "is_encrypted", Type: field.TypeBool, Default: false},
		{Name: "is_required", Type: field.TypeBool, Default: false},
		{Name: "default_value", Type: field.TypeString, Nullable: true},
		{Name: "created_at", Type: field.TypeTime},
		{Name: "updated_at", Type: field.TypeTime},
	}
	// AppConfigsTable holds the schema information for the "app_configs" table.
	AppConfigsTable = &schema.Table{
		Name:       "app_configs",
		Columns:    AppConfigsColumns,
		PrimaryKey: []*schema.Column{AppConfigsColumns[0]},
		Indexes: []*schema.Index{
			{
				Name:    "appconfig_key",
				Unique:  true,
				Columns: []*schema.Column{AppConfigsColumns[1]},
			},
			{
				Name:    "appconfig_category",
				Unique:  false,
				Columns: []*schema.Column{AppConfigsColumns[3]},
			},
			{
				Name:    "appconfig_is_required",
				Unique:  false,
				Columns: []*schema.Column{AppConfigsColumns[7]},
			},
		},
	}
	// ChannelsColumns holds the columns for the "channels" table.
	ChannelsColumns = []*schema.Column{
		{Name: "id", Type: field.TypeString, Unique: true},
		{Name: "name", Type: field.TypeString},
		{Name: "provider", Type: field.TypeString},
		{Name: "base_url", Type: field.TypeString},
		{Name: "api_key", Type: field.TypeString},
		{Name: "custom_key", Type: field.TypeString, Unique: true},
		{Name: "timeout", Type: field.TypeInt, Default: 30},
		{Name: "max_retries", Type: field.TypeInt, Default: 3},
		{Name: "enabled", Type: field.TypeBool, Default: true},
		{Name: "weight", Type: field.TypeInt, Default: 1},
		{Name: "priority", Type: field.TypeInt, Default: 1},
		{Name: "models_mapping", Type: field.TypeJSON, Nullable: true},
		{Name: "capabilities", Type: field.TypeJSON, Nullable: true},
		{Name: "created_at", Type: field.TypeTime},
		{Name: "updated_at", Type: field.TypeTime},
		{Name: "last_used_at", Type: field.TypeTime, Nullable: true},
		{Name: "request_count", Type: field.TypeInt64, Default: 0},
		{Name: "error_count", Type: field.TypeInt64, Default: 0},
		{Name: "success_rate", Type: field.TypeFloat64, Default: 1},
		{Name: "total_tokens", Type: field.TypeInt64, Default: 0},
		{Name: "avg_response_time", Type: field.TypeFloat64, Default: 0},
	}
	// ChannelsTable holds the schema information for the "channels" table.
	ChannelsTable = &schema.Table{
		Name:       "channels",
		Columns:    ChannelsColumns,
		PrimaryKey: []*schema.Column{ChannelsColumns[0]},
		Indexes: []*schema.Index{
			{
				Name:    "channel_custom_key",
				Unique:  true,
				Columns: []*schema.Column{ChannelsColumns[5]},
			},
			{
				Name:    "channel_provider",
				Unique:  false,
				Columns: []*schema.Column{ChannelsColumns[2]},
			},
			{
				Name:    "channel_enabled",
				Unique:  false,
				Columns: []*schema.Column{ChannelsColumns[8]},
			},
			{
				Name:    "channel_priority",
				Unique:  false,
				Columns: []*schema.Column{ChannelsColumns[10]},
			},
			{
				Name:    "channel_weight",
				Unique:  false,
				Columns: []*schema.Column{ChannelsColumns[9]},
			},
			{
				Name:    "channel_success_rate",
				Unique:  false,
				Columns: []*schema.Column{ChannelsColumns[18]},
			},
			{
				Name:    "channel_last_used_at",
				Unique:  false,
				Columns: []*schema.Column{ChannelsColumns[15]},
			},
		},
	}
	// RequestLogsColumns holds the columns for the "request_logs" table.
	RequestLogsColumns = []*schema.Column{
		{Name: "id", Type: field.TypeString, Unique: true},
		{Name: "claude_model", Type: field.TypeString},
		{Name: "openai_model", Type: field.TypeString},
		{Name: "request_body", Type: field.TypeString, Size: 2147483647},
		{Name: "response_body", Type: field.TypeString, Nullable: true, Size: 2147483647},
		{Name: "status_code", Type: field.TypeInt, Default: 0},
		{Name: "is_streaming", Type: field.TypeBool, Default: false},
		{Name: "input_tokens", Type: field.TypeInt, Default: 0},
		{Name: "output_tokens", Type: field.TypeInt, Default: 0},
		{Name: "duration_ms", Type: field.TypeFloat64, Default: 0},
		{Name: "error_message", Type: field.TypeString, Nullable: true},
		{Name: "created_at", Type: field.TypeTime},
		{Name: "updated_at", Type: field.TypeTime},
	}
	// RequestLogsTable holds the schema information for the "request_logs" table.
	RequestLogsTable = &schema.Table{
		Name:       "request_logs",
		Columns:    RequestLogsColumns,
		PrimaryKey: []*schema.Column{RequestLogsColumns[0]},
		Indexes: []*schema.Index{
			{
				Name:    "requestlog_created_at",
				Unique:  false,
				Columns: []*schema.Column{RequestLogsColumns[11]},
			},
			{
				Name:    "requestlog_claude_model",
				Unique:  false,
				Columns: []*schema.Column{RequestLogsColumns[1]},
			},
			{
				Name:    "requestlog_status_code",
				Unique:  false,
				Columns: []*schema.Column{RequestLogsColumns[5]},
			},
		},
	}
	// UsersColumns holds the columns for the "users" table.
	UsersColumns = []*schema.Column{
		{Name: "id", Type: field.TypeString, Unique: true},
		{Name: "username", Type: field.TypeString, Unique: true},
		{Name: "email", Type: field.TypeString, Nullable: true},
		{Name: "password_hash", Type: field.TypeString},
		{Name: "salt", Type: field.TypeString},
		{Name: "role", Type: field.TypeString, Default: "user"},
		{Name: "is_active", Type: field.TypeBool, Default: true},
		{Name: "last_login", Type: field.TypeTime, Nullable: true},
		{Name: "session_token", Type: field.TypeString, Nullable: true},
		{Name: "session_expires", Type: field.TypeTime, Nullable: true},
		{Name: "created_at", Type: field.TypeTime},
		{Name: "updated_at", Type: field.TypeTime},
	}
	// UsersTable holds the schema information for the "users" table.
	UsersTable = &schema.Table{
		Name:       "users",
		Columns:    UsersColumns,
		PrimaryKey: []*schema.Column{UsersColumns[0]},
		Indexes: []*schema.Index{
			{
				Name:    "user_username",
				Unique:  true,
				Columns: []*schema.Column{UsersColumns[1]},
			},
			{
				Name:    "user_email",
				Unique:  true,
				Columns: []*schema.Column{UsersColumns[2]},
			},
			{
				Name:    "user_session_token",
				Unique:  false,
				Columns: []*schema.Column{UsersColumns[8]},
			},
			{
				Name:    "user_created_at",
				Unique:  false,
				Columns: []*schema.Column{UsersColumns[10]},
			},
		},
	}
	// Tables holds all the tables in the schema.
	Tables = []*schema.Table{
		AppConfigsTable,
		ChannelsTable,
		RequestLogsTable,
		UsersTable,
	}
)

func init() {
}
