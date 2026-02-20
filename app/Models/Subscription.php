<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * @property \Illuminate\Support\Carbon|null $start_date
 * @property \Illuminate\Support\Carbon|null $end_date
 * @property \Illuminate\Support\Carbon|null $last_payment_at
 * @property \Illuminate\Support\Carbon|null $alert_sent_at
 */
class Subscription extends Model
{
    protected $fillable = [
        'user_id',
        'plan',
        'status',
        'start_date',
        'end_date',
        'github_sponsor_id',
        'amount_cents',
        'last_payment_at',
        'alert_sent_at',
    ];

    /** @return array<string, string> */
    protected function casts(): array
    {
        return [
            'start_date' => 'date',
            'end_date' => 'date',
            'last_payment_at' => 'datetime',
            'alert_sent_at' => 'datetime',
        ];
    }

    /** @return BelongsTo<User, $this> */
    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }

    public function isActive(): bool
    {
        return $this->status === 'active' && ($this->end_date === null || $this->end_date->isFuture());
    }

    public function isExpired(): bool
    {
        return $this->status === 'expired' || ($this->end_date !== null && $this->end_date->isPast());
    }

    public function isPastDue(): bool
    {
        return $this->status === 'past_due';
    }

    public function isSponsor(): bool
    {
        return in_array($this->plan, ['sponsor', 'team_sponsor']) && $this->isActive();
    }

    /** @param Builder<Subscription> $query */
    public function scopeExpiringSoon(Builder $query): void
    {
        $query->where('status', 'active')
            ->whereNotNull('end_date')
            ->where('end_date', '<=', now()->addDays(7));
    }

    /** @param Builder<Subscription> $query */
    public function scopeUnpaid(Builder $query): void
    {
        $query->where('status', 'past_due');
    }
}
