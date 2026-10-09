# Pulsewatch - self-hosted uptime monitoring

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/go-%23007d9c.svg?style=flat&logo=go&logoColor=white)
![React](https://img.shields.io/badge/react-%2320232a.svg?style=flat&logo=react&logoColor=%2361dafb)
![TypeScript](https://img.shields.io/badge/typescript-%23007acc.svg?style=flat&logo=typescript&logoColor=white)
![MongoDB](https://img.shields.io/badge/mongodb-4ea94b.svg?style=flat&logo=mongodb&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/postgresql-%23336791.svg?style=flat&logo=postgresql&logoColor=white)
![SQLite](https://img.shields.io/badge/sqlite-%2307405e.svg?style=flat&logo=sqlite&logoColor=white)
![Legacy Docker image pulls](https://img.shields.io/docker/pulls/0xfurai/peekaping-web)

**A modern, self-hosted uptime monitoring solution**

Pulsewatch is a community hard fork of Peekaping, a Go and React uptime monitoring system. It monitors websites, APIs, and services with status pages and alert notifications.

🔗 **[Pulsewatch project site](https://sanyam-malik.github.io/pulsewatch)**

🔗 Upstream demo **[demo.peekaping.com](https://demo.peekaping.com)**

🔗 **[Pulsewatch documentation](https://sanyam-malik.github.io/pulsewatch/docs)**

🔗 Upstream Terraform provider **[registry.terraform.io/providers/tafaust/peekaping](https://registry.terraform.io/providers/tafaust/peekaping/latest)**

## Why Pulsewatch Is an Alternative to Uptime Kuma

Pulsewatch continues the upstream project with an API-first architecture and a focus on extensible monitoring for DevOps teams.

**Key Advantages:**
- **API-first architecture** — all system functions are accessible through a RESTful API, ensuring complete automation and seamless integration with CI/CD processes and Infrastructure as Code tools
- **Easily extensible server architecture** — the modular structure allows adding new monitor types and notification channels without modifying the system core
- **Server built with Golang** — using one of the most performant compiled languages ensures high speed with minimal consumption of RAM and CPU resources
- **Unmatched stability** — thanks to a typed client and compiled Golang language, the system demonstrates high reliability and predictable operation
- **Modern interface** — clean user interface design built on contemporary UI/UX principles
- **Flexible storage options** — support for three popular databases (SQLite / PostgreSQL / MongoDB) allows adapting the solution to any infrastructure
- **API key management and access control** — built-in security system with access rights management and API keys provides enterprise-level protection


## ⚠️ Beta Status

**Pulsewatch is a community-maintained hard fork.**
Please note:

- The fork is under active development
- Some features could be changed
- I recommend testing in non-production environments first
- Please report any issues you encounter - your feedback helps us improve!

Please try Pulsewatch and provide feedback. Contributions help improve the fork.

## Quick start (docker + SQLite)

```bash
docker build -f Dockerfile.bundle.sqlite -t pulsewatch-bundle-sqlite:local .
docker run -d --restart=always \
  -p 8383:8383 \
  -e DB_NAME=/app/data/peekaping.db \
  -v $(pwd)/.data/sqlite:/app/data \
  --name pulsewatch \
  pulsewatch-bundle-sqlite:local
```

[Docker + SQLite Setup](https://sanyam-malik.github.io/pulsewatch/docs/self-hosting/docker-with-sqlite)

The existing published image coordinates and SQLite filename (`peekaping.db`) retain their upstream names for compatibility with existing deployments. Those published images are maintained upstream and may not include Pulsewatch features; build from this repository to run the fork. The setup guides also describe [PostgreSQL](https://sanyam-malik.github.io/pulsewatch/docs/self-hosting/docker-with-postgres) and [MongoDB](https://sanyam-malik.github.io/pulsewatch/docs/self-hosting/docker-with-mongo) setup.

## ⚡ Features

### Available Monitors

- HTTP/HTTPS
- TCP
- Ping (ICMP)
- DNS
- Push (incoming webhook)
- Docker container
- gRPC
- SNMP
- PostgreSQL
- Microsoft SQL Server
- MongoDB
- Redis
- MySQL/MariaDB
- MQTT Broker
- RabbitMQ
- Kafka Producer
- Steam (Source A2S_INFO)
- GameDig-compatible A2S servers
- Playwright browser checks (remote Chromium DevTools)

### 🔔 Alert Channels

- Email (SMTP)
- Webhook
- Telegram
- Slack
- Google Chat
- Signal
- Mattermost
- Matrix
- Discord
- WeCom
- WhatsApp (WAHA)
- PagerDuty
- Opsgenie
- Grafana OnCall
- NTFY
- Gotify
- Pushover
- SendGrid
- Twilio
- LINE Messenger
- PagerTree
- Pushbullet
- Microsoft Teams
- WhatsApp (Whapi.Cloud)
- CallMeBot (WhatsApp, Telegram, Facebook Messenger)
- Aliyun SMS
- DingDing
- ClickSend SMS
- Rocket.Chat

### ✨ Other

- Beautiful Status Pages
- SVG Status Badges
- Multi-Factor Authentication (MFA)
- Brute-Force Login Protection
- SSL Certificate Expiration Checks
- Multi-user workspaces with owner, admin, member, and viewer roles
- Monitor groups
- Status-page incident timelines
- HTTP status, response-time, and JSON response conditions

## 💡 Motivation and upstream

Pulsewatch is a hard fork of Peekaping, which was inspired by Uptime Kuma — a popular open-source monitoring solution. This fork retains the upstream project's MIT license and acknowledges its contributors.

This fork continues the upstream project's API-first, strongly typed architecture while adding community-requested capabilities.

**Our Approach:**
**API as the foundation.** The upstream project was designed as an API-first solution, where system functions are accessible programmatically.

**Performance through the right technology choices.** The server side is implemented in Golang — a fast and efficient language that delivers high performance with minimal RAM consumption. This is especially critical when monitoring a large number of services.

**Extensibility by design.** The system architecture allows easy addition of new notification channels, monitor types, and integrations without needing to modify the core codebase.

**Reliable client side.** The frontend is built with React and TypeScript, ensuring not only high performance but also reliability thanks to static typing. The client side was also designed with ease of extension in mind.

Pulsewatch aims to provide a reliable and customizable uptime monitoring solution capable of growing alongside your infrastructure.


![Pulsewatch Dashboard](./pictures/monitor.png)

## 📡 Upstream updates

The original maintainer shares updates on X.

[![Follow the upstream maintainer on X](https://img.shields.io/twitter/follow/0xfurai?label=Follow&style=social)](https://x.com/0xfurai)

## 🚧 Development roadmap

### General

- [x] Incidents
- [x] Migration tool (from Uptime Kuma)
- [x] Multi user, groups, access levels
- [x] Group monitors
- [x] Add support for Homepage widget
- [x] Gatus-like HTTP response conditions (status, response time, and JSON path)

### Monitors

- [x] HTTPS keyword and JSON query checks
- [x] Steam (Source A2S_INFO)
- [x] GameDig (Source A2S-compatible servers)
- [x] Playwright browser checks (remote Chromium DevTools endpoint)

### Notification channels

- [x] Microsoft Teams
- [x] WhatsApp (Whapi)
- [x] CallMeBot (WhatsApp, Telegram Call, Facebook Messenger)
- [x] AliyunSMS (阿里云短信服务)
- [x] DingDing (钉钉)
- [x] ClickSend SMS
- [x] Rocket.Chat

![Alt](https://repobeats.axiom.co/api/embed/747c845fe0118082b51a1ab2fc6f8a4edd73c016.svg "Repobeats analytics image")

## 🤝 Contributing

We welcome contributions! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Hard forked from [Peekaping](https://github.com/0xfurai/peekaping); thanks to the original authors and contributors.
- Inspired by [Uptime Kuma](https://github.com/louislam/uptime-kuma)
- Built with amazing open-source technologies
- Thanks to all Pulsewatch contributors and users

## 📞 Support

- **Issues**: Report bugs and request features via GitHub Issues

---

**Maintained by the Pulsewatch community; built on the Peekaping project**

## Criteria to Consider When Looking for Uptime Kuma Alternatives
When choosing an Uptime Kuma alternative, it's important to consider the technology stack and architecture. Solutions built with compiled languages (Go, Rust) deliver better performance and consume fewer resources. An API-first approach is critical for automation and integration with CI/CD processes.

Ensure the system supports the necessary monitoring protocols (HTTP/HTTPS, TCP, databases, Docker, gRPC) and integrates with your communication channels. Also important are flexible storage options (SQLite/PostgreSQL/MongoDB), security features (MFA, RBAC, API keys), modular architecture for extensibility, and an active community with quality documentation.

## How to Choose the Right Uptime Kuma Alternative
Determine the deployment format based on your capabilities and requirements. Self-hosted solutions provide complete control over data and deep customization options, but require technical resources for deployment and maintenance. Cloud solutions offer quick setup and automatic updates, but come with customization limitations and monthly subscriptions.

Conduct practical testing: deploy the system, create real monitors of different types, simulate failures, and evaluate detection speed. Check resource consumption under load, ease of API integrations, and interface quality. Pulsewatch is intended for teams that need a high-performance, API-first monitor with freedom from vendor lock-in.
