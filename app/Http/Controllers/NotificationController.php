<?php

namespace App\Http\Controllers;

use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Inertia\Inertia;
use Inertia\Response;

class NotificationController extends Controller
{
    public function index(Request $request): Response
    {
        return Inertia::render('notifications', [
            'notifications' => $request->user()?->notifications()->paginate(20),
        ]);
    }

    public function markRead(Request $request, string $id): RedirectResponse
    {
        $request->user()?->notifications()->where('id', $id)->first()?->markAsRead();

        return back();
    }
}
