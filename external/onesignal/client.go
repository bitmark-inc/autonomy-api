package onesignal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const ErrMsgAllPlayersNotSubscribed = "All included players are not subscribed"

func IsErrAllPlayersNotSubscribed(err error) bool {
	return err.Error() == ErrMsgAllPlayersNotSubscribed
}

type OneSignalClient struct {
	httpClient *http.Client
	traceMode  bool
	endpoint   string
	apiKey     string
	appID      string
}

type notificationButton struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// OneSignalError represents errors from OneSigal response
type OneSignalError struct {
	Errors interface{} `json:"errors"`
}

func (e *OneSignalError) Error() string {
	return fmt.Sprintf("%+v", e.Errors)
}

// NotificationRequest represents a request to create a notification.
type NotificationRequest struct {
	AppID                 string                 `json:"app_id"`
	Contents              map[string]string      `json:"contents,omitempty"`
	Headings              map[string]string      `json:"headings,omitempty"`
	Subtitle              map[string]string      `json:"subtitle,omitempty"`
	IsIOS                 bool                   `json:"isIos,omitempty"`
	IsAndroid             bool                   `json:"isAndroid,omitempty"`
	IncludedSegments      []string               `json:"included_segments,omitempty"`
	ExcludedSegments      []string               `json:"excluded_segments,omitempty"`
	IncludePlayerIDs      []string               `json:"include_player_ids,omitempty"`
	IOSBadgeType          string                 `json:"ios_badgeType,omitempty"`
	IOSBadgeCount         int                    `json:"ios_badgeCount,omitempty"`
	IOSSound              string                 `json:"ios_sound,omitempty"`
	AndroidSound          string                 `json:"android_sound,omitempty"`
	Data                  map[string]interface{} `json:"data,omitempty"`
	Buttons               []notificationButton   `json:"buttons,omitempty"`
	URL                   string                 `json:"url,omitempty"`
	SendAfter             string                 `json:"send_after,omitempty"`
	DelayedOption         string                 `json:"delayed_option,omitempty"`
	DeliveryTimeOfDay     string                 `json:"delivery_time_of_day,omitempty"`
	ContentAvailable      bool                   `json:"content_available,omitempty"`
	AndroidBackgroundData bool                   `json:"android_background_data,omitempty"`
	AmazonBackgroundData  bool                   `json:"amazon_background_data,omitempty"`
	TemplateID            string                 `json:"template_id,omitempty"`
	LocalChannelID        string                 `json:"existing_android_channel_id"`
	AndroidGroup          string                 `json:"android_group,omitempty"`
	AndroidGroupMessage   interface{}            `json:"android_group_message,omitempty"`
	Filters               []map[string]string    `json:"filters,omitempty"`
}

func NewClient(httpClient *http.Client) *OneSignalClient {
	return &OneSignalClient{
		httpClient: httpClient,
		traceMode:  viper.GetBool("onesignal.trace_mode"),
		endpoint:   viper.GetString("onesignal.endpoint"),
		apiKey:     viper.GetString("onesignal.key"),
		appID:      viper.GetString("onesignal.appid"),
	}
}

func (os *OneSignalClient) createRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	if err := enc.Encode(body); err != nil {
		return nil, err
	}

	fullurl := os.endpoint + path

	req, err := http.NewRequestWithContext(ctx, method, fullurl, buf)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic "+os.apiKey)

	return req, nil
}

func (os *OneSignalClient) SendNotification(ctx context.Context, reqBody *NotificationRequest) error {
	req, err := os.createRequest(ctx, "POST", "/api/v1/notifications", reqBody)

	if err != nil {
		return err
	}

	if os.traceMode {
		dumpBytes, err := httputil.DumpRequest(req, true)
		if err != nil {
			log.WithField("prefix", "onesignal").Error(err)
		}

		log.WithField("prefix", "onesignal").WithField("req", string(dumpBytes)).Debug("request to onesignal")
	}

	resp, err := os.httpClient.Do(req)
	if err != nil {
		return err
	}

	if os.traceMode {
		dumpBytes, err := httputil.DumpResponse(resp, true)
		if err != nil {
			log.WithField("prefix", "onesignal").Error(err)
		}

		log.WithContext(ctx).WithField("prefix", "onesignal").WithField("resp", string(dumpBytes)).Debug("response from onesignal")
	}

	decoder := json.NewDecoder(resp.Body)
	var errBody OneSignalError
	if err := decoder.Decode(&errBody); err != nil {
		return err
	}

	if errBody.Errors != nil {
		switch errors := errBody.Errors.(type) {
		case map[string]interface{}:
		case []interface{}:
			if len(errors) == 1 {
				if errStr, ok := errors[0].(string); ok {
					return fmt.Errorf(errStr)
				}
			}
		}
		return &errBody
	}

	return nil
}
