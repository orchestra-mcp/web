<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Models\Page;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Inertia\Inertia;
use Inertia\Response;

class PageController extends Controller
{
    public function index(): Response
    {
        return Inertia::render('admin/pages/index', [
            'pages' => Page::latest()->paginate(20),
        ]);
    }

    public function edit(Page $page): Response
    {
        return Inertia::render('admin/pages/edit', [
            'page' => $page,
        ]);
    }

    public function update(Request $request, Page $page): RedirectResponse
    {
        $validated = $request->validate([
            'title' => ['required', 'string', 'max:255'],
            'content' => ['required', 'string'],
            'meta' => ['nullable', 'array'],
            'is_published' => ['boolean'],
        ]);

        $page->update([
            ...$validated,
            'updated_by' => $request->user()?->id,
        ]);

        if ($request->hasFile('image')) {
            $page->addMediaFromRequest('image')->toMediaCollection('images');
        }

        return redirect()->route('admin.pages.index')->with('success', 'Page updated.');
    }
}
