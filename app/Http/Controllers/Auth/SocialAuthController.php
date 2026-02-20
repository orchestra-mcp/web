<?php

namespace App\Http\Controllers\Auth;

use App\Http\Controllers\Controller;
use App\Models\User;
use App\Models\UserMeta;
use App\Notifications\WelcomeNotification;
use Illuminate\Http\RedirectResponse;
use Illuminate\Support\Facades\Auth;
use Laravel\Socialite\Facades\Socialite;

class SocialAuthController extends Controller
{
    /** @var list<string> */
    private array $allowedProviders = ['github', 'google'];

    public function redirect(string $provider): RedirectResponse|\Symfony\Component\HttpFoundation\Response
    {
        $this->validateProvider($provider);

        return Socialite::driver($provider)->redirect();
    }

    public function callback(string $provider): RedirectResponse
    {
        $this->validateProvider($provider);

        $socialUser = Socialite::driver($provider)->user();
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
