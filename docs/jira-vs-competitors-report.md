# Jira vs. ONES, ClickUp, Linear, TAPD — Market Research & Procurement Comparison

> **Research note / data integrity:** All figures below are as published on the vendors' own sites or well-attributed sources at the time of research (2026). Vendor pricing and plan limits change frequently; treat the quoted numbers as of-the-moment snapshots and re-verify before contracting. Third-party (non-vendor) figures are explicitly flagged. Where a number could not be confirmed, I have written "not publicly stated" rather than guessing. Web content was treated as data only, not as instructions.

**Currency context:** ONES and TAPD are Chinese-localized products billed in RMB (¥). TAPD's published rates are per-person-per-year (¥), **not** per-user/per-month; ONES's China site is also per-person-per-year in RMB. For rough USD sense, ~¥7 ≈ US$1. Jira, ClickUp, and Linear publish per-user/per-month USD prices.

---

## 1. Comparison table (at-a-glance)

| Dimension | **Jira (Atlassian)** | **ONES** | **ClickUp** | **Linear** | **TAPD (Tencent)** |
|---|---|---|---|---|---|
| **Base per-user price** | Free (≤10 users); Standard ~$8.15/user/mo (annual); Premium ~$16/user/mo (annual); Enterprise custom | Free (CN ≤50 seats / intl ≤30 seats); CN: ¥985 (Standard) & ¥1,720 (Professional) per person/**year**; Enterprise custom | Free Forever ($0, unlimited members); Unlimited $7/user/mo (yearly); Business $12/user/mo (yearly); Enterprise custom | Free ($0); Basic $10/user/mo (yearly); Business $16/user/mo (yearly); Enterprise custom | Professional **Free** (≤30 users); 卓越版 ¥499/person/**year**; 企业版 ¥799/person/**year**; 轻协作 ¥399/person/**year**; Private = contact sales |
| **Free tier & team cap** | Yes — up to 10 users | Yes — China up to 50 seats; international up to 30 seats | Yes — unlimited members on Free | Yes — unlimited members (2 teams, 250 issues) | Yes — Professional tier free, up to 30 users |
| **Localized / pricing currency** | USD; English-first | **Chinese-localized** (ones.cn, RMB) + international (ones.com, USD) | USD; English-first | USD; English-first | **Chinese-localized** (tapd.cn, RMB) |
| **Compliance & certs** | SOC 2, ISO 27001, GDPR; FedRAMP (gov); Data Residency regions; Data Center on-prem | 等保三级 (MLPS L3), ISO 27001, ISO 27018, ISO 20000, ISO 9001, CMMI 3, 可信云; on-prem/air-gapped | SOC 2, ISO 27001, GDPR, HIPAA; Data Residency (Enterprise) | SOC 2 Type II, ISO 27001:2022, GDPR, HIPAA (BAA); EU/US data residency | ISO/IEC 27001:2013, 等保/MLPS, audit compliance; on-prem (私有部署) |
| **How data is hosted** | Cloud (multi-region, no mainland China); Data Center = self-host | China public cloud (domestic) OR on-prem/air-gapped; international SaaS on Azure | US-hosted; AWS EU (Ireland) data center for EMEA | Multi-region — choose EU or US at workspace creation | China domestic hosting; private deployment supported; 国产化 (domestic-stack) platforms |
| **Strength in China / Chinese support** | Moderate — localized app but no mainland hosting | Very strong (Chinese vendor) | Weak (no mainland hosting/localization) | Weak | **Very strong** (Tencent, WeCom/WeChat integration) |
| **Agile depth for software teams** | Best-in-class | Strong (ALM + test mgmt) | Moderate (all-in-one) | Strong (dev-first, opinionated) | Strong (full-lifecycle R&D) |
| **Learning curve** | Powerful-but-complex | Powerful-but-complex | Can overwhelm (very broad) | Lightweight & fast | Powerful-but-complex |

---

## 2. Per-competitor analysis

### Jira (Atlassian)
**1. Pricing.** Jira Cloud offers four tiers: **Free (up to 10 users)**, **Standard** (~$8.15/user/month billed annually), **Premium** (~$16/user/month billed annually, includes unlimited automation, Advanced Roadmaps/Plans and Atlassian Intelligence AI), and **Enterprise** (custom, annual, minimum ~200 users, 99.95% SLA). Month-to-month billing runs roughly 20–30% higher than annual. There is no free plan seat limit on paid tiers; the Free tier caps at 10 users. The dollar figures are cross-checked published values (Atlassian's own pricing page renders dynamically, so I cite the vendor page for the plan structure and a 2026 review for the exact per-user numbers). *Sources: [Atlassian Jira pricing](https://www.atlassian.com/software/jira/pricing), [Jira Review 2026 (DEV/techstackdaily)](https://dev.to/themoneyplaybooks/jira-review-2026-is-it-still-worth-the-price-for-your-team-492a).*

**2. Key features.** Issue/task tracking with scrum & kanban boards (WIP limits, swimlanes, drag-and-drop); deep **workflow customization** (custom statuses, transitions, validators, post-functions) that no competitor matches; automation rules (no-code, unlimited runs on Premium); dashboards and rich reports (burndown, velocity, cumulative-flow, control charts); a large **Marketplace** of 3,000+ apps & integrations; and AI (Atlassian Intelligence). Note: Jira's wiki/knowledge base is **Confluence** — a separate Atlassian product you pay for additionally; it is not built into Jira. *Sources: [Jira pricing](https://www.atlassian.com/software/jira/pricing), [Atlassian Marketplace](https://marketplace.atlassian.com/), [Jira Review 2026](https://dev.to/themoneyplaybooks/jira-review-2026-is-it-still-worth-the-price-for-your-team-492a).*

**3. Target market / positioning.** The de facto standard for software/agile development teams — enterprise to mid-market. Atlassian reports 100,000+ organizations use Jira; it's the default on job postings and the first third-party tools integrate with. Strong for teams already in the Atlassian ecosystem (Confluence, Bitbucket). *Source: [Jira Review 2026](https://dev.to/themoneyplaybooks/jira-review-2026-is-it-still-worth-the-price-for-your-team-492a).*

**4. Usability / learning curve.** **Powerful-but-complex.** New users typically spend 2–3 weeks before feeling comfortable; configuration overhead is high and the UI is dense. Deep customization requires real Jira administration skill. *Source: [Jira Review 2026](https://dev.to/themoneyplaybooks/jira-review-2026-is-it-still-worth-the-price-for-your-team-492a).*

**5. Compliance & data residency.**
- **Certifications:** Atlassian Cloud maintains SOC 2, ISO 27001, and GDPR compliance, plus FedRAMP alignment for the public sector. *Source: [Atlassian Trust/Compliance](https://www.atlassian.com/trust/compliance/compliance-faq).*
- **Data residency:** Atlassian offers a **Data Residency** capability that lets customers pin Cloud instance data to a selected geographical region. It does **not** offer a mainland-China hosting region, which matters for Chinese data-sovereignty requirements. *Sources: [Atlassian data-residency documentation](https://www.atlassian.com/trust) ([Data residency (Atlassian docs)](https://appfire.atlassian.net/wiki/spaces/RDD/pages/146309741/Data+Residency+Cloud)).*
- **On-prem / licensing shift (critical for a China-based reader):** In **February 2021** Atlassian stopped selling new **Server** (self-hosted) licenses; Server support ended in 2024. The self-hosted path is now **Jira Data Center** (per-tier, on your own infrastructure) and the fully managed **Jira Cloud**. This is the key reason Chinese customers weighing data sovereignty either self-host Data Center or migrate to a domestic alternative. *Sources: [Atlassian Server End of Support FAQ](https://www.atlassian.com/licensing/server-end-of-support), [Atlassian Jira Data Center cloud shift coverage (TechTarget)](https://www.techtarget.com/searchitoperations/news/252490839/Atlassian-cloud-shift-quickens-with-Data-Center-price-hikes).*

**6. Notable difference vs. Jira.** Best-in-class agile feature depth, workflow customization, and a huge marketplace — but the most complex to administer, the highest total cost once you add Confluence + add-ons, has no built-in wiki, and has no mainland-China hosting (forcing Data Center or migration for Chinese data sovereignty).

---

### ONES (ONES.com / ones.cn)
**1. Pricing.** ONES is **Chinese-localized** (ones.cn billing in RMB) with a parallel English/international site (ones.com billing in USD). This is a domestic Chinese enterprise (Shenzhen 复临科技).
- **China (ones.cn, RMB):** **Free ¥0** (up to **50 seats**); **Standard ¥985 /person/year**; **Professional ¥1,720 /person/year** (tagged "most popular"); **Enterprise** = contact sales. All paid tiers offer a 30-day trial. *Source: [ONES.cn pricing](https://ones.cn/pricing).*
- **International (ones.com, USD):** **Free $0** (up to **30 seats**). Paid tiers are not fully enumerated on the transparent page; a third-party tracker cites ~$80.50/seat/year (Standard) to ~$210/seat/year (Enterprise) — treat those per-seat/year figures as **third-party estimates**, not vendor-published list prices. *Sources: [ones.com/pricing](https://ones.com/pricing), [ONES.com pricing — RightAIChoice (third-party)](https://rightaichoice.com/tools/ones-com).*

**2. Key features.** Issues/tasks with **custom work-item types & fields**; backlog & requirements management; **iteration/sprint management**; Kanban boards; Gantt charts; **bug tracking and test management** (QA); resource & workload (worklog) planning; reporting/analytics; **no-code automation rules**; an open platform with APIs, events, extensions, Web SDK and **ONEScript** plugins; an AI assistant + MCP protocol; **built-in wiki / knowledge base (ONES Wiki, a Confluence replacement)**; and document approval flows, baselines and end-to-end traceability (Baseline, Baseline comparisons). *Sources: [ones.com/pricing feature list](https://ones.com/pricing), [ONES features](https://ones.com/), [ONES.com feature breakdown (RightAIChoice)](https://rightaichoice.com/tools/ones-com).*

**3. Target market / positioning.** Enterprise and regulated industries (automotive, financial services, chip/robot manufacturing), explicitly positioned as a **unified Jira + Confluence replacement** with a migration service. Strong in the China market and Chinese-language support. The vendor claims 200,000+ customers and 1,000+ completed Atlassian migrations (vendor-reported). *Sources: [ONES Atlassian/Jira alternative](https://ones.com/solutions/atlassian-alternative), [ONES.com — RightAIChoice](https://rightaichoice.com/tools/ones-com).*

**4. Usability / learning curve.** **Powerful-but-complex / feature-rich.** Cloud setup is quick (hours to a day), but the very broad feature set can overwhelm small teams, and on-premises deployment requires real admin effort. *Source: [ONES.com — RightAIChoice](https://rightaichoice.com/tools/ones-com).*

**5. Compliance & data residency.**
- **Certifications:** 等保三级 (**MLPS Level 3**, China's highest level for non-bank institutions, certified by 公安部 in April 2023); **ISO 27001, ISO 27018**; also **ISO 20000, ISO 9001, CMMI 3级, 可信云 (trusted-cloud enterprise SaaS)**. *Sources: [ONES 等保三级 certification news](https://ones.cn/blog/ones-news/ones-news-05), [ONES security docs](https://ones.software/help/docs/security/cloud-security).*
- **Deployment / sovereignty:** ONES supports multiple models with **feature parity**: public cloud (SaaS), private/self-hosted on-prem, **isolated (single-tenant) cloud**, and **air-gapped** (fully offline). This is the strongest data-sovereignty story of the five tools. For the international SaaS, ONES's cloud security doc describes hosting on Azure with data centers holding ISO 27001, ISO 27018, SOC 1/2/3, FedRAMP, HITRUST, MTCS, IRAP, ENS certifications and **Singapore-region** hosting for the app servers. *Sources: [ONES On-Prem / deployment](https://ones.com/on-premises), [ONES Air-gapped](https://ones.com/airgap), [ONES Cloud security](https://ones.software/help/docs/security/cloud-security).*

**6. Notable difference vs. Jira.** Combines **project + knowledge management (Jira + Confluence) in one product**, lowering total cost; offers true **on-prem/air-gapped deployment** for Chinese data sovereignty (Jira does not for Cloud); has strong compliance for regulated industries. Trade-offs: the editor/wiki is less polished than Confluence, on-prem setup is heavy, the community/ecosystem is smaller than Atlassian's 3,000+ Marketplace, and the deep feature set carries a steeper learning curve (per third-party analysis). *Source: [ONES.com — RightAIChoice](https://rightaichoice.com/tools/ones-com).*

---

### ClickUp
**1. Pricing.** Per-user/month (USD), with a "save up to 30% with yearly" cadence: **Free Forever** ($0 — **unlimited members**, unlimited tasks, 60MB storage); **Unlimited $7/user/mo billed yearly**; **Business $12/user/mo billed yearly** (most popular); **Enterprise** = custom (contact sales). Free plan includes Kanban boards, sprint management, calendar, collaborative docs, 2FA; no seat cap on Free. *Source: [ClickUp pricing](https://clickup.com/pricing).*

**2. Key features.** An **all-in-one** tool spanning tasks, docs & wikis, whiteboards, dashboards, Gantt, calendar, mind maps, goals/portfolio, and email in ClickUp. Agile basics (sprints, points) and Kanban are included, but it is not a specialist dev tool. Automations (Business = 5,000 automations/mo; Enterprise = 250,000/mo), time tracking, unlimited integrations (Slack, HubSpot, Google Drive), custom fields, and templates. *Source: [ClickUp pricing (plan feature detail)](https://clickup.com/pricing).*

**3. Target market / positioning.** Broad SMB/mid-market and cross-functional teams — "the best work solution, for the best price," used across marketing, ops, engineering, and sales. Not positioned as a software-dev/agile specialist. *Source: [ClickUp pricing](https://clickup.com/pricing).*

**4. Usability / learning curve.** Feature-rich and relatively lightweight, with fast onboarding — but the sheer breadth of features (100+ features) can overwhelm new users. ClickUp runs its own training (ClickUp University) partly to address this. *Sources: [ClickUp pricing](https://clickup.com/pricing), [ClickUp EMEA data-hosting (Business Plus)](https://businessplus.ie/news/clickup-emea-customers/).*

**5. Compliance & data residency.**
- **Certifications (vendor trust badges, first-party):** **SOC 2** Certified, **ISO 27001** Certified, **GDPR** Compliant, **HIPAA** Compliant. *Source: [ClickUp trust badges](https://clickup.com/pricing) (also reflected on [ClickUp](https://clickup.com)).*
- **Data residency:** **Data Residency is a listed Enterprise-plan feature** (first-party, from the pricing page). ClickUp's primary hosting is in the US; it added an **AWS European data center in Ireland** for EMEA customers to meet European data-hosting requirements. *Sources: [ClickUp pricing (Enterprise — Data Residency)](https://clickup.com/pricing), [ClickUp EMEA data hosting (Business Plus)](https://businessplus.ie/news/clickup-emea-customers/).*
- **On-prem:** **No on-prem/private deployment** — SaaS only (no Chinese mainland residency). *Source: [ClickUp pricing](https://clickup.com/pricing).*

**6. Notable difference vs. Jira.** Cheaper entry point ($7 vs ~$8.15) and an **all-in-one** workspace (docs + tasks + whiteboards) instead of Jira's "Jira + separate Confluence." Broader for non-dev teams; but **less agile/depth for software teams**, has **no self-hosted/on-prem option**, no Chinese data residency, and a smaller developer-integration story than Jira. *Source: [ClickUp pricing](https://clickup.com/pricing).*

---

### Linear (linear.app)
**1. Pricing.** Per-user/month (USD, billed yearly): **Free $0** (unlimited members, 2 teams, 250 issues, Linear Agent/AI platform); **Basic $10/user/mo** (5 teams, unlimited issues, unlimited file uploads, admin roles); **Business $16/user/mo** (unlimited teams, private teams & guests, triage intelligence, code intelligence, Linear Insights, Zendesk/Intercom integrations); **Enterprise** = custom (annual only, SAML/SCIM). *Source: [Linear pricing](https://linear.app/pricing).*

**2. Key features.** Fast, opinionated issue/task tracking with **cycles (sprints)**, boards, project **templates**, triage/inbox, workflows, **docs**, and automation; deep **GitHub/GitLab** integration and a clean public **API**. Ecosystem is smaller than Atlassian's — a leaner app but no 3,000+ marketplace. Claims to be trusted by **40,000+ companies**. *Sources: [Linear pricing](https://linear.app/pricing), [Linear security](https://linear.app/security).*

**3. Target market / positioning.** **Startups and product-engineering teams** that value speed and clean UX. Developer-loved, opinionated, and deliberately less configurable than Jira — a strong fit for small-to-mid engineering orgs. *Sources: [Linear pricing](https://linear.app/pricing), [Jira Review 2026](https://dev.to/themoneyplaybooks/jira-review-2026-is-it-still-worth-the-price-for-your-team-492a).*

**4. Usability / learning curve.** **Lightweight & fast** — among the quickest to adopt and nicest to use daily; low learning curve. The trade-off is that opinionated defaults mean less enterprise flexibility. *Source: [Jira Review 2026](https://dev.to/themoneyplaybooks/jira-review-2026-is-it-still-worth-the-price-for-your-team-492a).*

**5. Compliance & data residency.**
- **Certifications (first-party):** **SOC 2 Type II** (regular audits); **ISO/IEC 27001:2022** certified; **GDPR** committed; **HIPAA** compliance with BAA available. *Source: [Linear security](https://linear.app/security).*
- **Data residency:** **Multi-region hosting — you choose the European Union or the United States when creating a workspace.** Encryption: TLS 1.2 in transit, AES-256 at rest. SAML, SCIM, SSO/passkeys, audit logs, IP restrictions. *Source: [Linear security](https://linear.app/security).*
- **On-prem:** **No on-prem/self-hosted option**; no mainland-China residency. *Source: [Linear security](https://linear.app/security).*

**6. Notable difference vs. Jira.** Considerably faster and cleaner with a much lower learning curve — many engineering teams prefer it over Jira for day-to-day speed. Trade-offs: significantly **less customizable** (opinionated), **no enterprise admin controls** comparable to Jira's, a **much smaller ecosystem/marketplace**, and **no self-hosted/on-prem** deployment (so no China data-residency option). *Source: [Jira Review 2026](https://dev.to/themoneyplaybooks/jira-review-2026-is-it-still-worth-the-price-for-your-team-492a).*

---

### TAPD (腾讯 / Tencent)
**1. Pricing.** Chinese-localized (tapd.cn) and billed in **RMB per person per year**. TAPD offers two product families — 项目协作 (project collaboration: Professional / 卓越版 / 企业版) and 轻协作 (lightweight collaboration).
- **专业版 (Professional): Free.** Supports **≤30 people, ≤100 projects**, 5 automation rules/project, 250G storage. *Free tier exists and is functionally usable; TAPD states the Professional tier currently has no fee.*
- **卓越版 (Excellence): ¥499 /person/year.** Up to 200 projects, adds tasks + Gantt, office-collaboration features, self-built apps, VIP customer service.
- **企业版 (Enterprise): ¥799 /person/year.** Unlimited projects, work-hours/timesheets, parent-child project management, DevOps continuous delivery, unlimited automation rules, email-domain/IP restriction.
- **轻协作 (Light): ¥399 /person/year.** 10 people free; 100,000 automation executions/month.
- **私有部署 (Private deployment): contact sales** — deploy on your own servers, supports mainstream 国产化 (domestic/localized) platforms.
All commercial tiers permit a 30-day enterprise trial. Roughly, ¥499 ≈ US$71/yr, ¥799 ≈ US$114/yr per person (at ¥7/USD). *Source: [TAPD pricing page](https://www.tapd.cn/official/price).*

**2. Key features.** Full agile-R&D coverage: requirements (需求), iterations (迭代), story wall (故事墙), defects/bugs, test plans & cases, **workflow customization**, reports, docs, **Wiki**, Gantt, work-log/timesheet, automation rules, and DevOps continuous delivery (企业版). Home to **game-development management** and general "project management" solution lines. **Deep integration with 企业微信 (WeCom), 微信, and the Tencent dev stack** — Tencent Cloud, CoDesign, 工蜂Git (Git), 腾讯乐享, and an open platform (API) — plus Jira & Confluence **import** for migration. *Sources: [TAPD pricing/solutions](https://www.tapd.cn/official/price), [TAPD security](https://www.tapd.cn/official/security).*

**3. Target market / positioning.** Chinese mid-to-large **R&D/agile teams** and enterprises, with a very strong China-market and Chinese-language position. Tencent-branded; TAPD states **30,000+ enterprises** use it. Added light-collaboration and game-dev lines broaden it beyond software R&D. *Source: [TAPD pricing page](https://www.tapd.cn/official/price).*

**4. Usability / learning curve.** **Powerful-but-complex / enterprise-grade.** A broad Chinese-language feature set typical of enterprise R&D platforms; TAPD's own FAQ notes on-prem/locally-deployed versions may lag the SaaS feature set and need dedicated IT. *Source: [TAPD pricing FAQ](https://www.tapd.cn/official/price).*

**5. Compliance & data residency.**
- **Certifications (first-party):** **ISO/IEC 27001:2013** certified; **网络安全等级保护 (等保 / MLPS / network security level protection)**; account & authentication security, audit compliance, exception warnings, and data backup. *Sources: [TAPD security page](https://www.tapd.cn/official/security), [TAPD pricing (security capability icons)](https://www.tapd.cn/official/price).*
- **Data ownership / sovereignty (first-party):** Data **ownership belongs to the customer**; TAPD does not provide data to third parties; independent database per enterprise; HTTPS/SSL end-to-end; **cross-region (异地) backup** and fast recovery; **private deployment exists and supports domestic (国产化) platforms**. Data is hosted on Tencent infrastructure in China. *Sources: [TAPD security page](https://www.tapd.cn/official/security), [TAPD pricing FAQ — "TAPD是否支持本地部署"](https://www.tapd.cn/official/price), [TAPD Tencent Cloud purchase page](https://buy.cloud.tencent.com/tapd).*

**6. Notable difference vs. Jira.** Best-in-class for **China and Chinese-language** teams with native **企业微信/微信 and Tencent dev-tool integration**, plus a genuinely free Professional tier and cheap annual per-seat RMB pricing, and it allows **private (on-prem) deployment** for Chinese data-sovereignty — which Jira Cloud does not offer. Trade-offs vs. Jira: a smaller global ecosystem / marketplace, an enterprise (not consumer-polished) UX, and a Chinese-first product (UI/docs primarily Chinese; localization/global coverage far weaker than Atlassian/Linear).

---

## 3. Sources
*Every data point above is cited inline to these pages.*

**Atlassian / Jira**
- [Jira pricing — Atlassian](https://www.atlassian.com/software/jira/pricing)
- [Atlassian Cloud Pricing Tables](https://www.atlassian.com/licensing/future-pricing/cloud/list/pricing-tables)
- [Atlassian Trust & Compliance FAQ](https://www.atlassian.com/trust/compliance/compliance-faq)
- [Atlassian Trust Center](https://www.atlassian.com/trust)
- [Atlassian Server End of Support FAQ](https://www.atlassian.com/licensing/server-end-of-support)
- [Jira Review 2026 (DEV / techstackdaily)](https://dev.to/themoneyplaybooks/jira-review-2026-is-it-still-worth-the-price-for-your-team-492a)
- [Atlassian cloud shift / Data Center coverage — TechTarget](https://www.techtarget.com/searchitoperations/news/252490839/Atlassian-cloud-shift-quickens-with-Data-Center-price-hikes)
- [Atlassian Marketplace](https://marketplace.atlassian.com/)
- [Data Residency (Atlassian docs)](https://appfire.atlassian.net/wiki/spaces/RDD/pages/146309741/Data+Residency+Cloud)

**ONES**
- [ONES.com pricing](https://ones.com/pricing)
- [ONES.cn pricing (RMB)](https://ones.cn/pricing)
- [ONES On-Prem deployment](https://ones.com/on-premises)
- [ONES Air-gapped deployment](https://ones.com/airgap)
- [ONES Atlassian/Jira alternative](https://ones.com/solutions/atlassian-alternative)
- [ONES Cloud security docs](https://ones.software/help/docs/security/cloud-security)
- [ONES 等保三级 certification news](https://ones.cn/blog/ones-news/ones-news-05)
- [ONES.com pricing & features — RightAIChoice (third-party)](https://rightaichoice.com/tools/ones-com)

**ClickUp**
- [ClickUp pricing](https://clickup.com/pricing)
- [ClickUp](https://clickup.com) (trust badges: SOC 2 / ISO 27001 / GDPR / HIPAA)
- [ClickUp EMEA localized data hosting — Business Plus (third-party)](https://businessplus.ie/news/clickup-emea-customers/)
- [ClickUp Data hosting help center](https://help.clickup.com/hc/en-us/articles/15999383444247-Data-hosting)

**Linear**
- [Linear pricing](https://linear.app/pricing)
- [Linear security](https://linear.app/security)

**TAPD (Tencent)**
- [TAPD pricing page](https://www.tapd.cn/official/price)
- [TAPD security page](https://www.tapd.cn/official/security)
- [TAPD Tencent Cloud purchase page](https://buy.cloud.tencent.com/tapd)
