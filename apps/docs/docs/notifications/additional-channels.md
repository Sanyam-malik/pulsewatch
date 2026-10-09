# Additional notification channels

The notification-channel selector includes the integrations below. Each one uses a webhook or provider API directly; enter credentials only for an account/service you control. Use the channel's **Test** action before attaching it to monitors.

## Microsoft Teams

Create an incoming webhook for the target Teams channel and paste its URL in **Incoming webhook URL**. The optional title is rendered using Liquid. The sender posts a MessageCard with the monitor alert in its text.

## WhatsApp (Whapi)

This is separate from the existing [WAHA integration](./whatsapp.md): WAHA remains the `whatsapp` channel, while Whapi.Cloud is `whatsapp_whapi`.

- **Whapi.Cloud API URL**: base URL, normally `https://gate.whapi.cloud`
- **API token**: bearer token for the Whapi.Cloud channel
- **Recipient**: international phone number or supported group chat ID

The sender calls `POST /messages/text` with the destination and notification text.

## CallMeBot

Select the activated CallMeBot service (WhatsApp, Telegram, or Facebook Messenger), then provide the API key CallMeBot issued when activating that service. WhatsApp also requires the destination phone number; Telegram requires the destination username/user ID. Facebook Messenger activation keys are account-bound and do not require a recipient field. The Telegram option uses CallMeBot's Telegram call endpoint; the optional language setting is forwarded as its `lang` value. The endpoint override is optional and intended for compatible endpoints/testing.

CallMeBot must already be activated for the destination; activation steps and supported features are controlled by CallMeBot.

## Aliyun SMS (阿里云短信服务)

Provide an Aliyun AccessKey ID/secret, SMS sign name, template code and phone number(s). The sender uses the Aliyun SMS `SendSms` RPC API, signing requests with HMAC-SHA1. Region defaults to `cn-hangzhou`; the standard endpoint is `https://dysmsapi.aliyuncs.com/`.

**Template parameters** must be a JSON object with keys matching the variables in your Aliyun SMS template. Values support Liquid expressions, for example:

```json
{"monitor":"{{ monitor.name }}","status":"{{ status }}","message":"{{ msg }}"}
```

Leave parameters as `{}` only when the selected Aliyun template has no variables. The endpoint override is optional and should normally remain empty.

## DingDing (钉钉)

Create a custom robot in the target DingTalk group and paste its webhook URL. If signature verification is enabled for the robot, provide its signing secret; requests include DingTalk's timestamp and HMAC-SHA256 signature. Alerts are sent as Markdown messages.

## ClickSend SMS

Enter your ClickSend account username, API key and recipient's international phone number. Sender ID is optional and depends on destination-country restrictions. The optional custom message uses Liquid syntax; leave it empty to send the standard monitor alert. SMS length and segmentation are subject to ClickSend's limits and billing.

## Rocket.Chat

Create an incoming webhook integration in Rocket.Chat and provide its webhook URL. Optional display name, channel override, and emoji are sent with the alert text. The webhook's configured channel is used when the channel field is empty.

## Template values

Where a form accepts a Liquid template, the available values include `{{ msg }}`, `{{ status }}`, `{{ monitor.name }}`, and `{{ heartbeat.* }}`. Heartbeat values are not populated for certificate-expiry notifications.
