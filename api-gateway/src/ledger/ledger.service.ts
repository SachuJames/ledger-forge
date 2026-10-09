import { HttpException, Injectable } from '@nestjs/common';
import { randomUUID } from 'node:crypto';
import { config } from '../config.js';
import { CreateAccountDto, TransferDto } from './dto.js';

// Thin client over the Go ledger core. Every accounting rule lives in Go;
// this layer only translates, forwards core errors untouched, and never
// invents ledger semantics of its own.
@Injectable()
export class LedgerService {
  private readonly base = config.ledgerCoreUrl.replace(/\/$/, '');

  private async call<T>(path: string, init?: RequestInit): Promise<{ data: T; replayed: boolean }> {
    const headers = new Headers(init?.headers);
    headers.set('content-type', 'application/json');
    let res: Response;
    try {
      res = await fetch(this.base + path, { ...init, headers });
    } catch {
      throw new HttpException('ledger core unreachable', 502);
    }
    const data = (await res.json().catch(() => ({}))) as T;
    if (!res.ok) {
      const msg = (data as { error?: string }).error ?? 'ledger core error';
      throw new HttpException(msg, res.status === 422 ? 422 : 502);
    }
    return { data, replayed: res.headers.get('idempotent-replayed') === 'true' };
  }

  createAccount(dto: CreateAccountDto) {
    return this.call('/v1/accounts', {
      method: 'POST',
      body: JSON.stringify(dto),
    }).then((r) => r.data);
  }

  async postTransfer(dto: TransferDto) {
    const entry = {
      idempotency_key: dto.idempotencyKey ?? randomUUID(),
      narration: dto.narration ?? '',
      lines: [
        { account_id: dto.debitAccountId, currency: dto.currency, debit: dto.amount },
        { account_id: dto.creditAccountId, currency: dto.currency, credit: dto.amount },
      ],
    };
    return this.call('/v1/entries', { method: 'POST', body: JSON.stringify(entry) });
  }

  balance(accountId: string, currency: string) {
    return this.call(
      `/v1/accounts/${encodeURIComponent(accountId)}/balance?currency=${encodeURIComponent(currency)}`,
    ).then((r) => r.data);
  }

  journal(limit = 25, offset = 0) {
    return this.call(`/v1/entries?limit=${limit}&offset=${offset}`).then((r) => r.data);
  }
}
