<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Subscription;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;
use Inertia\Inertia;
use Inertia\Response;

class SubscriptionController extends Controller
{
    public function index(Request $request): Response
    {
        $query = Subscription::with('user');

        if ($status = $request->input('status')) {
            $query->where('status', $status);
        }

        return Inertia::render('admin/subscriptions/index', [
            'subscriptions' => $query->latest()->paginate(20)->withQueryString(),
            'filters' => $request->only(['status']),
        ]);
    }

    public function edit(Subscription $subscription): Response
    {
        return Inertia::render('admin/subscriptions/edit', [
            'subscription' => $subscription->load('user'),
        ]);
    }

    public function update(Request $request, Subscription $subscription): RedirectResponse
    {
        $validated = $request->validate([
            'plan' => ['required', Rule::in(['free', 'pro', 'team'])],
            'status' => ['required', Rule::in(['active', 'expired', 'cancelled', 'past_due'])],
            'start_date' => ['nullable', 'date'],
            'end_date' => ['nullable', 'date', 'after_or_equal:start_date'],
        ]);

        $subscription->update($validated);

        return redirect()->route('admin.subscriptions.index')->with('success', 'Subscription updated.');
    }

    public function alerts(): Response
    {
        return Inertia::render('admin/subscriptions/alerts', [
            'expiring' => Subscription::with('user')->expiringSoon()->get(),
            'unpaid' => Subscription::with('user')->unpaid()->get(),
        ]);
    }
}
