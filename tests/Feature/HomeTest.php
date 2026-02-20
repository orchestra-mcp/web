<?php

use App\Models\Page;

test('home page can be rendered', function () {
    Page::create(['slug' => 'home', 'title' => 'Home', 'content' => '{}', 'is_published' => true]);

    $response = $this->get(route('home'));
    $response->assertOk();
});

test('home page renders even without page data', function () {
    $response = $this->get(route('home'));
    $response->assertOk();
});
