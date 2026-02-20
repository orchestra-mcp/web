<?php

namespace App\Http\Controllers;

use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Inertia\Inertia;
use Inertia\Response;

class SubscriptionController extends Controller
{
    public function show(Request $request): Response
    {
        return Inertia::render('subscription', [
            'subscription' => $request->user()?->subscription,
        ]);
    }

    public function subscribe(Request $request): Response|RedirectResponse
    {
        $user = $request->user();

        if ($user?->hasActiveSubscription() || $user?->isAdmin()) {
            return redirect()->route('dashboard');
        }

        return Inertia::render('subscribe');
    }
}
