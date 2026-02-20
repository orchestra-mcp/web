<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\User;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Hash;
use Illuminate\Validation\ValidationException;

class AuthController extends Controller
{
    public function login(Request $request): JsonResponse
    {
        $request->validate([
            'email' => ['required', 'email'],
            'password' => ['required'],
        ]);

        /** @var User|null $user */
        $user = User::where('email', $request->email)->first();

        if (! $user || ! Hash::check($request->password, $user->password)) {
            throw ValidationException::withMessages([
                'email' => ['The provided credentials are incorrect.'],
            ]);
        }

        if ($user->isBlocked()) {
            throw ValidationException::withMessages([
                'email' => ['Your account has been blocked.'],
            ]);
        }

        if ($user->needsPassword()) {
            throw ValidationException::withMessages([
                'email' => ['Please set a password on the web app first.'],
            ]);
        }

        if (! $user->hasActiveSubscription() && ! $user->isAdmin()) {
            throw ValidationException::withMessages([
                'email' => ['An active GitHub Sponsors subscription is required. Visit the web app to subscribe.'],
            ]);
        }

        $token = $user->createToken('ide-token', ['ide:access'])->plainTextToken;

        return response()->json([
            'token' => $token,
            'user' => $user->load('roles', 'subscription'),
        ]);
    }

    public function user(Request $request): JsonResponse
    {
        return response()->json([
            'user' => $request->user()?->load('roles', 'subscription'),
        ]);
    }

    public function logout(Request $request): JsonResponse
    {
        $request->user()?->currentAccessToken()->delete();

        return response()->json(['message' => 'Logged out']);
    }
}
