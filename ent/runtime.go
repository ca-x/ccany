// Code generated by ent, DO NOT EDIT.

package ent

import (
	"ccany/ent/appconfig"
	"ccany/ent/channel"
	"ccany/ent/requestlog"
	"ccany/ent/schema"
	"ccany/ent/user"
	"time"
)

// The init function reads all schema descriptors with runtime code
// (default values, validators, hooks and policies) and stitches it
// to their package variables.
func init() {
	appconfigFields := schema.AppConfig{}.Fields()
	_ = appconfigFields
	// appconfigDescCategory is the schema descriptor for category field.
	appconfigDescCategory := appconfigFields[3].Descriptor()
	// appconfig.DefaultCategory holds the default value on creation for the category field.
	appconfig.DefaultCategory = appconfigDescCategory.Default.(string)
	// appconfigDescType is the schema descriptor for type field.
	appconfigDescType := appconfigFields[4].Descriptor()
	// appconfig.DefaultType holds the default value on creation for the type field.
	appconfig.DefaultType = appconfigDescType.Default.(string)
	// appconfigDescIsEncrypted is the schema descriptor for is_encrypted field.
	appconfigDescIsEncrypted := appconfigFields[6].Descriptor()
	// appconfig.DefaultIsEncrypted holds the default value on creation for the is_encrypted field.
	appconfig.DefaultIsEncrypted = appconfigDescIsEncrypted.Default.(bool)
	// appconfigDescIsRequired is the schema descriptor for is_required field.
	appconfigDescIsRequired := appconfigFields[7].Descriptor()
	// appconfig.DefaultIsRequired holds the default value on creation for the is_required field.
	appconfig.DefaultIsRequired = appconfigDescIsRequired.Default.(bool)
	// appconfigDescCreatedAt is the schema descriptor for created_at field.
	appconfigDescCreatedAt := appconfigFields[9].Descriptor()
	// appconfig.DefaultCreatedAt holds the default value on creation for the created_at field.
	appconfig.DefaultCreatedAt = appconfigDescCreatedAt.Default.(func() time.Time)
	// appconfigDescUpdatedAt is the schema descriptor for updated_at field.
	appconfigDescUpdatedAt := appconfigFields[10].Descriptor()
	// appconfig.DefaultUpdatedAt holds the default value on creation for the updated_at field.
	appconfig.DefaultUpdatedAt = appconfigDescUpdatedAt.Default.(func() time.Time)
	// appconfig.UpdateDefaultUpdatedAt holds the default value on update for the updated_at field.
	appconfig.UpdateDefaultUpdatedAt = appconfigDescUpdatedAt.UpdateDefault.(func() time.Time)
	channelFields := schema.Channel{}.Fields()
	_ = channelFields
	// channelDescTimeout is the schema descriptor for timeout field.
	channelDescTimeout := channelFields[6].Descriptor()
	// channel.DefaultTimeout holds the default value on creation for the timeout field.
	channel.DefaultTimeout = channelDescTimeout.Default.(int)
	// channelDescMaxRetries is the schema descriptor for max_retries field.
	channelDescMaxRetries := channelFields[7].Descriptor()
	// channel.DefaultMaxRetries holds the default value on creation for the max_retries field.
	channel.DefaultMaxRetries = channelDescMaxRetries.Default.(int)
	// channelDescEnabled is the schema descriptor for enabled field.
	channelDescEnabled := channelFields[8].Descriptor()
	// channel.DefaultEnabled holds the default value on creation for the enabled field.
	channel.DefaultEnabled = channelDescEnabled.Default.(bool)
	// channelDescWeight is the schema descriptor for weight field.
	channelDescWeight := channelFields[9].Descriptor()
	// channel.DefaultWeight holds the default value on creation for the weight field.
	channel.DefaultWeight = channelDescWeight.Default.(int)
	// channelDescPriority is the schema descriptor for priority field.
	channelDescPriority := channelFields[10].Descriptor()
	// channel.DefaultPriority holds the default value on creation for the priority field.
	channel.DefaultPriority = channelDescPriority.Default.(int)
	// channelDescCreatedAt is the schema descriptor for created_at field.
	channelDescCreatedAt := channelFields[13].Descriptor()
	// channel.DefaultCreatedAt holds the default value on creation for the created_at field.
	channel.DefaultCreatedAt = channelDescCreatedAt.Default.(func() time.Time)
	// channelDescUpdatedAt is the schema descriptor for updated_at field.
	channelDescUpdatedAt := channelFields[14].Descriptor()
	// channel.DefaultUpdatedAt holds the default value on creation for the updated_at field.
	channel.DefaultUpdatedAt = channelDescUpdatedAt.Default.(func() time.Time)
	// channel.UpdateDefaultUpdatedAt holds the default value on update for the updated_at field.
	channel.UpdateDefaultUpdatedAt = channelDescUpdatedAt.UpdateDefault.(func() time.Time)
	// channelDescRequestCount is the schema descriptor for request_count field.
	channelDescRequestCount := channelFields[16].Descriptor()
	// channel.DefaultRequestCount holds the default value on creation for the request_count field.
	channel.DefaultRequestCount = channelDescRequestCount.Default.(int64)
	// channelDescErrorCount is the schema descriptor for error_count field.
	channelDescErrorCount := channelFields[17].Descriptor()
	// channel.DefaultErrorCount holds the default value on creation for the error_count field.
	channel.DefaultErrorCount = channelDescErrorCount.Default.(int64)
	// channelDescSuccessRate is the schema descriptor for success_rate field.
	channelDescSuccessRate := channelFields[18].Descriptor()
	// channel.DefaultSuccessRate holds the default value on creation for the success_rate field.
	channel.DefaultSuccessRate = channelDescSuccessRate.Default.(float64)
	// channelDescTotalTokens is the schema descriptor for total_tokens field.
	channelDescTotalTokens := channelFields[19].Descriptor()
	// channel.DefaultTotalTokens holds the default value on creation for the total_tokens field.
	channel.DefaultTotalTokens = channelDescTotalTokens.Default.(int64)
	// channelDescAvgResponseTime is the schema descriptor for avg_response_time field.
	channelDescAvgResponseTime := channelFields[20].Descriptor()
	// channel.DefaultAvgResponseTime holds the default value on creation for the avg_response_time field.
	channel.DefaultAvgResponseTime = channelDescAvgResponseTime.Default.(float64)
	requestlogFields := schema.RequestLog{}.Fields()
	_ = requestlogFields
	// requestlogDescStatusCode is the schema descriptor for status_code field.
	requestlogDescStatusCode := requestlogFields[5].Descriptor()
	// requestlog.DefaultStatusCode holds the default value on creation for the status_code field.
	requestlog.DefaultStatusCode = requestlogDescStatusCode.Default.(int)
	// requestlogDescIsStreaming is the schema descriptor for is_streaming field.
	requestlogDescIsStreaming := requestlogFields[6].Descriptor()
	// requestlog.DefaultIsStreaming holds the default value on creation for the is_streaming field.
	requestlog.DefaultIsStreaming = requestlogDescIsStreaming.Default.(bool)
	// requestlogDescInputTokens is the schema descriptor for input_tokens field.
	requestlogDescInputTokens := requestlogFields[7].Descriptor()
	// requestlog.DefaultInputTokens holds the default value on creation for the input_tokens field.
	requestlog.DefaultInputTokens = requestlogDescInputTokens.Default.(int)
	// requestlogDescOutputTokens is the schema descriptor for output_tokens field.
	requestlogDescOutputTokens := requestlogFields[8].Descriptor()
	// requestlog.DefaultOutputTokens holds the default value on creation for the output_tokens field.
	requestlog.DefaultOutputTokens = requestlogDescOutputTokens.Default.(int)
	// requestlogDescDurationMs is the schema descriptor for duration_ms field.
	requestlogDescDurationMs := requestlogFields[9].Descriptor()
	// requestlog.DefaultDurationMs holds the default value on creation for the duration_ms field.
	requestlog.DefaultDurationMs = requestlogDescDurationMs.Default.(float64)
	// requestlogDescCreatedAt is the schema descriptor for created_at field.
	requestlogDescCreatedAt := requestlogFields[11].Descriptor()
	// requestlog.DefaultCreatedAt holds the default value on creation for the created_at field.
	requestlog.DefaultCreatedAt = requestlogDescCreatedAt.Default.(func() time.Time)
	// requestlogDescUpdatedAt is the schema descriptor for updated_at field.
	requestlogDescUpdatedAt := requestlogFields[12].Descriptor()
	// requestlog.DefaultUpdatedAt holds the default value on creation for the updated_at field.
	requestlog.DefaultUpdatedAt = requestlogDescUpdatedAt.Default.(func() time.Time)
	// requestlog.UpdateDefaultUpdatedAt holds the default value on update for the updated_at field.
	requestlog.UpdateDefaultUpdatedAt = requestlogDescUpdatedAt.UpdateDefault.(func() time.Time)
	userFields := schema.User{}.Fields()
	_ = userFields
	// userDescRole is the schema descriptor for role field.
	userDescRole := userFields[5].Descriptor()
	// user.DefaultRole holds the default value on creation for the role field.
	user.DefaultRole = userDescRole.Default.(string)
	// user.RoleValidator is a validator for the "role" field. It is called by the builders before save.
	user.RoleValidator = userDescRole.Validators[0].(func(string) error)
	// userDescIsActive is the schema descriptor for is_active field.
	userDescIsActive := userFields[6].Descriptor()
	// user.DefaultIsActive holds the default value on creation for the is_active field.
	user.DefaultIsActive = userDescIsActive.Default.(bool)
	// userDescCreatedAt is the schema descriptor for created_at field.
	userDescCreatedAt := userFields[10].Descriptor()
	// user.DefaultCreatedAt holds the default value on creation for the created_at field.
	user.DefaultCreatedAt = userDescCreatedAt.Default.(func() time.Time)
	// userDescUpdatedAt is the schema descriptor for updated_at field.
	userDescUpdatedAt := userFields[11].Descriptor()
	// user.DefaultUpdatedAt holds the default value on creation for the updated_at field.
	user.DefaultUpdatedAt = userDescUpdatedAt.Default.(func() time.Time)
	// user.UpdateDefaultUpdatedAt holds the default value on update for the updated_at field.
	user.UpdateDefaultUpdatedAt = userDescUpdatedAt.UpdateDefault.(func() time.Time)
}
