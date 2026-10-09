import { Injectable, UnauthorizedException } from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import { compare } from 'bcryptjs';
import { config } from '../config.js';

@Injectable()
export class AuthService {
  constructor(private readonly jwt: JwtService) {}

  // Single demo operator account. Real deployments would back this with a
  // users table; the interface stays the same.
  async login(username: string, password: string): Promise<{ access_token: string }> {
    const ok =
      username === config.adminUser && (await compare(password, config.adminPasswordHash));
    if (!ok) {
      throw new UnauthorizedException('invalid credentials');
    }
    return {
      access_token: await this.jwt.signAsync({ sub: username }),
    };
  }
}
