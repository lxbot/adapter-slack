# adapter-slack

[![Build Status](https://cloud.drone.io/api/badges/lxbot/adapter-slack/status.svg)](https://cloud.drone.io/lxbot/adapter-slack)
![GitHub](https://img.shields.io/github/license/lxbot/adapter-slack)

![](https://i.imgur.com/YQyQyCS.png)

lxbot adapter for slack

# ENV

| key | description |
| - | - |
| LXBOT_SLACK_OAUTH_ACCESS_TOKEN | Bot User OAuth Access Token |
| LXBOT_SLACK_SIGNING_SECRET | App Credentials - Signing Secret |

# NOTE

- Create Slack app in https://api.slack.com/apps .  
- `OAuth & Permissions - Scopes` must set be `chat:write` and `users.profiles:read` .  
- `Subscribe to bot events` must set be `app_mention` and `message.channels` .
