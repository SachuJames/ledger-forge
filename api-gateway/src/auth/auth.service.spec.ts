import { JwtService } from '@nestjs/jwt';
import { beforeEach, describe, expect, it } from 'vitest';
import { AuthService } from './auth.service.js';

describe('AuthService', () => {
  let svc: AuthService;

  beforeEach(() => {
    svc = new AuthService(
      new JwtService({ secret: 'test-secret' } as never),
    );
  });

  it('issues a token for the demo operator', async () => {
    const { access_token } = await svc.login('admin', 'changeme');
    expect(typeof access_token).toBe('string');
    expect(access_token.split('.')).toHaveLength(3); // header.payload.signature
  });

  it('rejects a wrong password', async () => {
    await expect(svc.login('admin', 'wrong')).rejects.toThrowError(/invalid credentials/);
  });

  it('rejects an unknown user', async () => {
    await expect(svc.login('nobody', 'changeme')).rejects.toThrowError(/invalid credentials/);
  });
});
