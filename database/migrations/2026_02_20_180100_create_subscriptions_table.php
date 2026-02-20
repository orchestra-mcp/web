<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('subscriptions', function (Blueprint $table) {
            $table->id();
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->enum('plan', ['free', 'sponsor', 'team_sponsor'])->default('free');
            $table->enum('status', ['active', 'expired', 'cancelled', 'past_due'])->default('active');
            $table->date('start_date')->nullable();
            $table->date('end_date')->nullable();
            $table->string('github_sponsor_id')->nullable();
            $table->integer('amount_cents')->default(0);
            $table->timestamp('last_payment_at')->nullable();
            $table->timestamp('alert_sent_at')->nullable();
            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('subscriptions');
    }
};
