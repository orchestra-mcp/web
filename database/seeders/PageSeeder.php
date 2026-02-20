<?php

namespace Database\Seeders;

use App\Models\Page;
use Illuminate\Database\Seeder;

class PageSeeder extends Seeder
{
    public function run(): void
    {
        Page::firstOrCreate(
            ['slug' => 'home'],
            [
                'title' => 'Orchestra MCP — AI-Agentic IDE',
                'content' => json_encode([
                    'hero' => [
                        'title' => 'Orchestra MCP',
                        'subtitle' => 'The AI-Agentic IDE for Modern Development',
                        'description' => 'Build faster with AI-powered code generation, intelligent autocomplete, and agentic workflows across Desktop, Chrome, Mobile, and Web.',
                        'cta_primary' => ['label' => 'Get Started', 'url' => '/register'],
                        'cta_secondary' => ['label' => 'Learn More', 'url' => '#features'],
                    ],
                    'features' => [
                        ['title' => 'AI-Powered IDE', 'description' => 'Built-in AI chat, code generation, and intelligent suggestions powered by Claude and GPT.', 'icon' => 'brain'],
                        ['title' => 'Multi-Platform', 'description' => 'Available on Desktop (macOS, Windows, Linux), Chrome Extension, Mobile (iOS, Android), and Web.', 'icon' => 'monitor'],
                        ['title' => 'Extension Ecosystem', 'description' => 'Native extensions, Raycast compatibility (~95%), and VS Code compatibility (~85%).', 'icon' => 'puzzle'],
                        ['title' => 'Real-time Sync', 'description' => 'Settings, projects, and sessions sync across all your devices in real-time.', 'icon' => 'refresh'],
                    ],
                    'plans' => [
                        ['name' => 'Free', 'price' => '$0', 'features' => ['Basic IDE features', 'Community support', '1 device']],
                        ['name' => 'Pro', 'price' => '$5/mo', 'features' => ['All IDE features', 'AI-powered tools', 'Unlimited devices', 'Priority support']],
                        ['name' => 'Team', 'price' => '$25/mo', 'features' => ['Everything in Pro', 'Team collaboration', 'Admin panel', 'Custom extensions']],
                    ],
                ], JSON_THROW_ON_ERROR),
                'meta' => [
                    'og_title' => 'Orchestra MCP — AI-Agentic IDE',
                    'og_description' => 'Build faster with AI across Desktop, Chrome, Mobile, and Web.',
                ],
                'is_published' => true,
            ],
        );
    }
}
