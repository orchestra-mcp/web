<?php

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class EnsurePasswordSet
{
    public function handle(Request $request, Closure $next): Response
    {
        if ($request->user() && ! $request->user()->password_set) {
            if (! $request->routeIs('set-password', 'set-password.store', 'logout')) {
                return redirect()->route('set-password');
            }
        }

        return $next($request);
    }
}
