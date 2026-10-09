import { HttpException } from '@nestjs/common';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { TransferDto } from './dto.js';
import { LedgerService } from './ledger.service.js';

function mockFetchOnce(json: unknown, status = 200, headers: Record<string, string> = {}) {
  const res = {
    ok: status >= 200 && status < 300,
    status,
    headers: new Headers(headers),
    json: async () => json,
  } as Response;
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(res));
  return vi.mocked(fetch);
}

describe('LedgerService', () => {
  let svc: LedgerService;

  beforeEach(() => {
    vi.unstubAllGlobals();
    svc = new LedgerService();
  });

  it('builds a balanced two-line entry from a transfer', async () => {
    const fetchMock = mockFetchOnce({ id: 'e1' }, 201);
    const dto = Object.assign(new TransferDto(), {
      debitAccountId: 'a1',
      creditAccountId: 'a2',
      currency: 'INR',
      amount: 2500,
      narration: 'test',
    });

    const { data, replayed } = await svc.postTransfer(dto);

    expect(replayed).toBe(false);
    expect(data).toEqual({ id: 'e1' });
    const [, init] = fetchMock.mock.calls[0];
    const body = JSON.parse((init as RequestInit).body as string);
    expect(body.lines).toHaveLength(2);
    expect(body.lines[0]).toMatchObject({ account_id: 'a1', debit: 2500 });
    expect(body.lines[1]).toMatchObject({ account_id: 'a2', credit: 2500 });
    expect(body.idempotency_key).toBeTruthy(); // auto-generated when absent
  });

  it('keeps a client-supplied idempotency key', async () => {
    const fetchMock = mockFetchOnce({ id: 'e1' }, 201);
    const dto = Object.assign(new TransferDto(), {
      debitAccountId: 'a1',
      creditAccountId: 'a2',
      currency: 'USD',
      amount: 100,
      idempotencyKey: 'order-99',
    });

    await svc.postTransfer(dto);

    const [, init] = fetchMock.mock.calls[0];
    const body = JSON.parse((init as RequestInit).body as string);
    expect(body.idempotency_key).toBe('order-99');
  });

  it('surfaces the replayed flag from the core', async () => {
    mockFetchOnce({ id: 'e1' }, 201, { 'idempotent-replayed': 'true' });
    const dto = Object.assign(new TransferDto(), {
      debitAccountId: 'a1',
      creditAccountId: 'a2',
      currency: 'INR',
      amount: 100,
    });

    const { replayed } = await svc.postTransfer(dto);
    expect(replayed).toBe(true);
  });

  it('passes core validation errors through with their message', async () => {
    mockFetchOnce({ error: 'ledger: unbalanced in INR' }, 422);

    const dto = Object.assign(new TransferDto(), {
      debitAccountId: 'a1',
      creditAccountId: 'a2',
      currency: 'INR',
      amount: 100,
    });

    await expect(svc.postTransfer(dto)).rejects.toThrowError(
      expect.objectContaining({ message: 'ledger: unbalanced in INR' }),
    );
  });

  it('maps an unreachable core to 502', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockRejectedValue(new Error('connect refused')),
    );
    await expect(svc.balance('a1', 'INR')).rejects.toThrowError(
      expect.objectContaining({ status: 502 }),
    );
  });

  it('creates accounts through the core', async () => {
    const fetchMock = mockFetchOnce({ id: 'a1', code: '1001' }, 201);
    const data = await svc.createAccount({ code: '1001', name: 'Cash', type: 'asset' });
    expect(data).toEqual({ id: 'a1', code: '1001' });
    expect(fetchMock.mock.calls[0][0]).toContain('/v1/accounts');
  });

  it('throws HttpException (not a raw error) on core failure', async () => {
    mockFetchOnce({ error: 'boom' }, 500);
    const err = await svc.journal().catch((e) => e);
    expect(err).toBeInstanceOf(HttpException);
  });
});
