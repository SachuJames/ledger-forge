import { Injectable } from '@nestjs/common';
import { PassportStrategy } from '@nestjs/passport';
import { ExtractJwt, Strategy } from 'passport-jwt';
import { config } from '../config.js';

@Injectable()
export class JwtStrategy extends PassportStrategy(Strategy) {
  constructor() {
    super({
      jwtFromRequest: ExtractJwt.fromAuthHeaderAsBearerToken(),
      secretOrKey: config.jwtSecret,
    });
  }

  // The payload is tiny on purpose: sub is the operator name. Anything the
  // request needs beyond identity comes from the ledger core.
  async validate(payload: { sub: string }) {
    return { username: payload.sub };
  }
}
