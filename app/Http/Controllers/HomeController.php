<?php

namespace App\Http\Controllers;

use App\Models\Page;
use Inertia\Inertia;
use Inertia\Response;

class HomeController extends Controller
{
    public function index(): Response
    {
        $page = Page::where('slug', 'home')->published()->first();

        return Inertia::render('home', [
            'page' => $page,
        ]);
    }

    public function show(string $slug): Response
    {
        $page = Page::where('slug', $slug)->published()->firstOrFail();

        return Inertia::render('page', [
            'page' => $page,
        ]);
    }
}
