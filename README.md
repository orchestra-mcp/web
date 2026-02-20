# Orchestra MCP — Web Portal

The SaaS web portal for Orchestra MCP. Handles user authentication, subscription management, admin panel, and desktop OAuth integration.

## Tech Stack

| Layer    | Technology                                                    |
| -------- | ------------------------------------------------------------- |
| Backend  | PHP 8.2+, Laravel 12, SQLite (default)                        |
| Frontend | React 19 (with Compiler), TypeScript 5.7, Inertia.js 2       |
| Styling  | Tailwind CSS 4, shadcn/ui (New York), Lucide Icons            |
| Build    | Vite 7, React Compiler (babel-plugin-react-compiler)          |
| Auth     | Fortify (headless), Sanctum (API tokens), Socialite (OAuth)   |
| Testing  | Pest 4, PHPStan, ESLint, Prettier                             |
| Routing  | Wayfinder (type-safe Laravel routes in TypeScript)             |
| Perms    | Spatie Permission (roles + permissions)                        |
| Media    | Spatie Media Library (avatar uploads, conversions)             |

## Features

### Authentication
- Email/password login and registration
- Social OAuth — GitHub, Google, Discord
- Email verification (required before full access)
- Password reset flow
- Two-factor authentication (TOTP with recovery codes)
- Set password (social-only users can add a password later)
- Account blocking (admin-controlled)

### Subscriptions
- Plans: `free`, `pro`, `team`
- GitHub Sponsors webhook integration
- Expiring-soon alerts (7-day warning)
- Admin subscription management

### Admin Panel (`/admin`)
- Dashboard with stats
- User management (CRUD, block/unblock)
- Role management (Spatie Permission)
- Subscription management (view, edit, alerts)
- Page CMS (static pages like Terms, Privacy)

### Desktop Integration
- OAuth proxy for Notion and Google Calendar
- Forwards OAuth codes to `localhost` for the desktop app to consume

### API (Sanctum)
- `POST /api/auth/login` — email/password login, returns token
- `GET /api/auth/user` — authenticated user data
- `POST /api/auth/logout` — revoke token

### User Settings
- Profile (name, email, avatar upload)
- Password change
- Two-factor authentication setup
- Appearance (light / dark / system theme)
- Account deletion (soft delete)

## Setup

```bash
# Install dependencies
composer install
npm install

# Environment
cp .env.example .env
php artisan key:generate

# Database
touch database/database.sqlite
php artisan migrate --seed

# Build frontend
npm run build
```

## Development

```bash
# Start all dev processes (server + queue + logs + Vite HMR)
composer run dev
```

This runs four parallel processes:
- `php artisan serve` — Laravel HTTP server
- `php artisan queue:listen` — Job queue worker
- `php artisan pail` — Real-time log tail
- `npm run dev` — Vite with hot module replacement

The app is served at `http://localhost:8000` (or `https://web.test` with Laravel Herd).

## Testing

```bash
# Run all tests
php artisan test --compact

# Run specific test
php artisan test --filter=AuthenticationTest

# Direct Pest
vendor/bin/pest

# Linting
vendor/bin/pint          # PHP code style (Laravel Pint)
vendor/bin/phpstan       # Static analysis
npm run lint             # ESLint
npm run format           # Prettier
npm run types            # TypeScript type check
```

### Test Coverage
- **Auth**: login, register, password reset, email verification, 2FA challenge, set password, blocked user, API auth
- **Settings**: profile update, password update, 2FA setup
- **Admin**: user management, role management
- **Models**: User, Subscription

## Project Structure

```
web/
├── app/
│   ├── Actions/              # Fortify auth actions
│   ├── Console/Commands/     # Artisan commands
│   ├── Http/
│   │   ├── Controllers/
│   │   │   ├── Admin/        # User, Role, Subscription, Page CRUD
│   │   │   ├── Auth/         # Social login, desktop OAuth, set password
│   │   │   ├── Settings/     # Profile, password, 2FA
│   │   │   ├── Api/          # Sanctum token auth
│   │   │   └── Webhook/      # GitHub Sponsors webhook
│   │   ├── Middleware/        # Admin gate, active check, password check
│   │   └── Requests/         # Form request validation
│   ├── Models/
│   │   ├── User.php          # Roles, permissions, avatar, subscription
│   │   ├── Subscription.php  # Plans, status, GitHub sponsor
│   │   ├── UserMeta.php      # Key-value preferences
│   │   └── Page.php          # CMS pages
│   └── Notifications/
├── config/                   # Auth, Fortify, Sanctum, services, permissions
├── database/
│   └── migrations/           # Users, subscriptions, pages, roles, media
├── resources/
│   ├── js/
│   │   ├── components/       # App header, sidebar, user menu, 2FA modal
│   │   │   └── ui/           # 27 shadcn/ui components
│   │   ├── hooks/            # Appearance, 2FA, clipboard, mobile
│   │   ├── layouts/          # App, auth, settings layouts
│   │   ├── pages/
│   │   │   ├── auth/         # Login, register, 2FA, password reset
│   │   │   ├── settings/     # Profile, password, 2FA, appearance
│   │   │   └── admin/        # Dashboard, users, roles, subscriptions, pages
│   │   └── lib/utils.ts      # cn() Tailwind merge helper
│   ├── css/app.css           # Tailwind v4 imports + CSS variables
│   └── views/app.blade.php   # Root HTML template
├── routes/
│   ├── web.php               # Public + auth + admin routes
│   ├── api.php               # Sanctum API routes
│   └── settings.php          # Settings page routes
├── tests/
│   ├── Unit/                 # User, Subscription model tests
│   └── Feature/              # Auth, settings, admin, API tests
├── composer.json
├── package.json
├── vite.config.ts
└── .env.example
```

## Database Schema

| Table               | Purpose                                    |
| ------------------- | ------------------------------------------ |
| `users`             | Accounts with soft delete, 2FA, status     |
| `user_metas`        | Key-value user preferences                 |
| `subscriptions`     | Plan, status, GitHub sponsor, dates        |
| `pages`             | CMS content (slug, title, markdown, meta)  |
| `roles`             | Spatie: super_admin, admin, subscriber, user |
| `permissions`       | Spatie: view-admin-panel, manage-users, etc |
| `media`             | Spatie Media Library (avatars, page images) |
| `personal_access_tokens` | Sanctum API tokens                    |
| `sessions`          | Database session driver                    |
| `notifications`     | Laravel database notifications             |

## Roles & Permissions

| Role          | Permissions                                                       |
| ------------- | ----------------------------------------------------------------- |
| `super_admin` | All permissions                                                   |
| `admin`       | view-admin-panel, manage-users, manage-roles, manage-subscriptions, manage-pages |
| `subscriber`  | access-ide-features                                               |
| `user`        | Default (no special permissions)                                  |

## Environment Variables

Key variables in `.env` (see `.env.example` for full list):

```env
APP_URL=http://web.test

# OAuth Providers
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
DISCORD_CLIENT_ID=
DISCORD_CLIENT_SECRET=

# Desktop OAuth Proxy
NOTION_CLIENT_ID=
NOTION_CLIENT_SECRET=
GOOGLE_CALENDAR_CLIENT_ID=
GOOGLE_CALENDAR_CLIENT_SECRET=

# GitHub Sponsors Webhook
GITHUB_WEBHOOK_SECRET=
```

## Production Build

```bash
npm run build
composer install --optimize-autoloader --no-dev
php artisan optimize   # Cache routes, config, views
```

Requires PHP 8.2+, Composer, and Node.js (for frontend builds).
