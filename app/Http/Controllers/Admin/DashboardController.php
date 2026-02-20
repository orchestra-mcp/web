<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Subscription;
use App\Models\User;
use Inertia\Inertia;
use Inertia\Response;

class DashboardController extends Controller
{
    public function index(): Response
    {
        return Inertia::render('admin/dashboard', [
            'stats' => [
                'total_users' => User::count(),
                'active_users' => User::active()->count(),
                'blocked_users' => User::blocked()->count(),
                'active_subscriptions' => Subscription::where('status', 'active')->count(),
                'expiring_soon' => Subscription::expiringSoon()->count(),
                'unpaid' => Subscription::unpaid()->count(),
            ],
        ]);
    }
}
