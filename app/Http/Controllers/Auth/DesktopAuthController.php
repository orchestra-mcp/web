<?php

namespace App\Http\Controllers\Auth;

use App\Http\Controllers\Controller;
use Illuminate\Http\Request;

/**
 * Handles OAuth callbacks for desktop app integrations.
 *
 * Unlike SocialAuthController (which handles user login via Socialite),
 * this controller receives OAuth authorization codes from third-party
 * providers (Notion, Google Calendar, etc.) and forwards them to the
 * local desktop app's settings server for token exchange.
 *
 * Flow:
 *  1. Desktop opens browser → provider OAuth page
 *  2. Provider redirects to orchestra-mcp.dev/auth/{provider}/callback?code=xxx
 *  3. This controller renders a page that forwards the code to localhost:19191
 *  4. Desktop app exchanges the code for a token and stores it
 */
class DesktopAuthController extends Controller
{
    /** @var list<string> Providers handled by this controller (desktop integrations) */
    private array $providers = ['notion', 'google-calendar'];

    public function callback(Request $request, string $provider)
    {
        if (! in_array($provider, $this->providers)) {
            abort(404, "Provider [{$provider}] is not supported for desktop auth.");
        }

        $code = $request->query('code');
        $error = $request->query('error');

        if ($error) {
            return response()->view('auth.desktop-callback', [
                'provider' => $provider,
                'success' => false,
                'error' => $request->query('error_description', $error),
            ]);
        }

        if (! $code) {
            return response()->view('auth.desktop-callback', [
                'provider' => $provider,
                'success' => false,
                'error' => 'No authorization code received.',
            ]);
        }

        // Map provider to local API callback path
        $apiPaths = [
            'notion' => '/api/notion/auth/callback',
            'google-calendar' => '/api/google-calendar/auth/callback',
        ];

        return response()->view('auth.desktop-callback', [
            'provider' => $provider,
            'providerLabel' => $this->providerLabel($provider),
            'success' => true,
            'code' => $code,
            'localCallbackUrl' => 'http://127.0.0.1:19191' . ($apiPaths[$provider] ?? ''),
        ]);
    }

    private function providerLabel(string $provider): string
    {
        return match ($provider) {
            'notion' => 'Notion',
            'google-calendar' => 'Google Calendar',
            default => ucfirst($provider),
        };
    }
}
