<?php

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class EnsureActiveSubscription
{
    public function handle(Request $request, Closure $next): Response
    {
        $user = $request->user();

        if ($user && ! $user->hasActiveSubscription() && ! $user->isAdmin()) {
            if (! $request->routeIs('subscribe', 'subscription', 'logout', 'notifications.*')) {
                return redirect()->route('subscribe');
            }
        }

        return $next($request);
    }
}
