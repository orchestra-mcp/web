<?php

namespace App\Http\Controllers\Auth;

use App\Http\Controllers\Controller;
use App\Models\User;
use App\Models\UserMeta;
use App\Notifications\WelcomeNotification;
use GuzzleHttp\Exception\ClientException;
use Illuminate\Http\RedirectResponse;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\Log;
use Laravel\Socialite\Facades\Socialite;

class SocialAuthController extends Controller
{
    /** @var list<string> */
    private array $allowedProviders = ['github', 'google', 'discord'];

    public function redirect(string $provider): RedirectResponse|\Symfony\Component\HttpFoundation\Response
    {
        $this->validateProvider($provider);

        return Socialite::driver($provider)->redirect();
    }

    public function callback(string $provider): RedirectResponse
    {
        $this->validateProvider($provider);

        try {
            $socialUser = Socialite::driver($provider)->user();
        } catch (ClientException $e) {
            $status = $e->getResponse()->getStatusCode();
            $providerName = ucfirst($provider);

            if ($status === 403) {
                Log::warning("Social auth rate limited by {$provider}", [
                    'status' => $status,
                    'provider' => $provider,
                ]);

                return redirect()->route('login')->withErrors([
                    'email' => "{$providerName} API rate limit exceeded. Please wait a few minutes and try again.",
                ]);
            }

            Log::error("Social auth failed for {$provider}", [
                'status' => $status,
                'provider' => $provider,
                'message' => $e->getMessage(),
            ]);

            return redirect()->route('login')->withErrors([
                'email' => "Unable to authenticate with {$providerName}. Please try again later.",
            ]);
        } catch (\Exception $e) {
            Log::error("Social auth unexpected error for {$provider}", [
                'provider' => $provider,
                'message' => $e->getMessage(),
            ]);

            return redirect()->route('login')->withErrors([
                'email' => 'Authentication failed. Please try again or use a different login method.',
            ]);
        }

        $metaKey = $provider.'_id';

        // Find by provider ID in user_metas
        $meta = UserMeta::where('key', $metaKey)
            ->where('value', (string) $socialUser->getId())
            ->first();

        if ($meta) {
            $user = $meta->user;
        } else {
            // Find by email or create new user
            $user = User::where('email', $socialUser->getEmail())->first();

            if (! $user) {
                $user = User::create([
                    'name' => $socialUser->getName() ?? $socialUser->getNickname() ?? 'User',
                    'email' => $socialUser->getEmail(),
                    'password' => null,
                    'password_set' => false,
                    'email_verified_at' => now(),
                ]);

                $user->assignRole('user');
                $user->notify(new WelcomeNotification);
            }

            // Store provider ID in metas
            $user->setMeta($metaKey, (string) $socialUser->getId());
        }

        // Store provider avatar URL
        if ($socialUser->getAvatar()) {
            $user->setMeta($provider.'_avatar_url', $socialUser->getAvatar());
        }

        if ($user->isBlocked()) {
            return redirect()->route('login')->withErrors([
                'email' => 'Your account has been blocked.',
            ]);
        }

        Auth::login($user, remember: true);

        if ($user->needsPassword()) {
            return redirect()->route('set-password');
        }

        return redirect()->intended(route('dashboard'));
    }

    private function validateProvider(string $provider): void
    {
        if (! in_array($provider, $this->allowedProviders)) {
            abort(404, "Provider [{$provider}] is not supported.");
        }
    }
}
