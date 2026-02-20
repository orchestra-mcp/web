<?php

use App\Http\Controllers\Admin;
use App\Http\Controllers\Auth\SetPasswordController;
use App\Http\Controllers\Auth\DesktopAuthController;
use App\Http\Controllers\Auth\SocialAuthController;
use App\Http\Controllers\HomeController;
use App\Http\Controllers\NotificationController;
use App\Http\Controllers\SubscriptionController;
use App\Http\Controllers\Webhook\GitHubSponsorsController;
use Illuminate\Support\Facades\Route;
use Inertia\Inertia;

// Public pages
Route::get('/', [HomeController::class, 'index'])->name('home');
Route::get('/page/{slug}', [HomeController::class, 'show'])->name('page.show');

// Desktop integration OAuth callbacks (forwards code to local desktop app, not user login)
Route::get('/auth/{provider}/callback', [DesktopAuthController::class, 'callback'])
    ->whereIn('provider', ['notion', 'google-calendar'])
    ->name('auth.desktop.callback');

// Dynamic social auth (user login via Socialite)
Route::get('/auth/{provider}', [SocialAuthController::class, 'redirect'])->name('auth.social');
Route::get('/auth/{provider}/callback', [SocialAuthController::class, 'callback'])->name('auth.social.callback');

// Password setup (social login users)
Route::middleware('auth')->group(function () {
    Route::get('/set-password', [SetPasswordController::class, 'show'])->name('set-password');
    Route::post('/set-password', [SetPasswordController::class, 'store'])->name('set-password.store');
});

// Authenticated (password required, user active)
Route::middleware(['auth', 'verified', 'password.set', 'user.active'])->group(function () {
    Route::get('/dashboard', fn () => Inertia::render('dashboard'))->name('dashboard');
    Route::get('/subscription', [SubscriptionController::class, 'show'])->name('subscription');
    Route::get('/notifications', [NotificationController::class, 'index'])->name('notifications.index');
    Route::post('/notifications/{id}/read', [NotificationController::class, 'markRead'])->name('notifications.read');
});

// Admin panel
Route::middleware(['auth', 'verified', 'password.set', 'user.active', 'admin'])
    ->prefix('admin')
    ->name('admin.')
    ->group(function () {
        Route::get('/', [Admin\DashboardController::class, 'index'])->name('dashboard');
        Route::resource('users', Admin\UserController::class);
        Route::post('users/{user}/toggle-status', [Admin\UserController::class, 'toggleStatus'])->name('users.toggle-status');
        Route::resource('roles', Admin\RoleController::class);
        Route::resource('subscriptions', Admin\SubscriptionController::class)->only(['index', 'edit', 'update']);
        Route::get('subscriptions-alerts', [Admin\SubscriptionController::class, 'alerts'])->name('subscriptions.alerts');
        Route::resource('pages', Admin\PageController::class)->only(['index', 'edit', 'update']);
    });

// Webhooks (no auth, signature-verified)
Route::post('/webhooks/github-sponsors', [GitHubSponsorsController::class, 'handle'])->name('webhooks.github-sponsors');

require __DIR__.'/settings.php';
