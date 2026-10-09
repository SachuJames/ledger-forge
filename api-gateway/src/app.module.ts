import { Module } from '@nestjs/common';
import { ThrottlerModule } from '@nestjs/throttler';
import { AuthModule } from './auth/auth.module.js';
import { EventsModule } from './events/events.module.js';
import { LedgerModule } from './ledger/ledger.module.js';

@Module({
  imports: [
    // 100 requests/minute per IP across the API. Transfers are additionally
    // protected by idempotency keys, so a retried request is always safe.
    ThrottlerModule.forRoot({ throttlers: [{ limit: 100, ttl: 60_000 }] }),
    AuthModule,
    EventsModule,
    LedgerModule,
  ],
})
export class AppModule {}
