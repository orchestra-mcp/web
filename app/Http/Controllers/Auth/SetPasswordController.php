<?php

namespace App\Http\Controllers\Auth;

use App\Http\Controllers\Controller;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rules\Password;
use Inertia\Inertia;
use Inertia\Response;

class SetPasswordController extends Controller
{
    public function show(Request $request): RedirectResponse|Response
    {
        if ($request->user()?->password_set) {
            return redirect()->route('dashboard');
        }

        return Inertia::render('auth/set-password');
    }

    public function store(Request $request): RedirectResponse
    {
        $request->validate([
            'password' => ['required', 'confirmed', Password::defaults()],
        ]);

        $user = $request->user();

        if (! $user) {
            return redirect()->route('login');
        }

        $user->update([
            'password' => $request->password,
            'password_set' => true,
        ]);

        return redirect()->route('dashboard')->with('success', 'Password set successfully.');
    }
}
