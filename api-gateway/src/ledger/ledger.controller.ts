import {
  Body,
  Controller,
  Get,
  Param,
  Post,
  Query,
  UseGuards,
} from '@nestjs/common';
import { ApiBearerAuth, ApiTags } from '@nestjs/swagger';
import { JwtAuthGuard } from '../auth/jwt-auth.guard.js';
import { EventsGateway } from '../events/events.gateway.js';
import { CreateAccountDto, TransferDto } from './dto.js';
import { LedgerService } from './ledger.service.js';

@ApiTags('ledger')
@ApiBearerAuth()
@UseGuards(JwtAuthGuard)
@Controller('api')
export class LedgerController {
  constructor(
    private readonly ledger: LedgerService,
    private readonly events: EventsGateway,
  ) {}

  @Post('accounts')
  createAccount(@Body() dto: CreateAccountDto) {
    return this.ledger.createAccount(dto);
  }

  @Post('transfers')
  async transfer(@Body() dto: TransferDto) {
    const { data, replayed } = await this.ledger.postTransfer(dto);
    if (!replayed) {
      this.events.broadcast('entry.posted', data);
    }
    return { replayed, entry: data };
  }

  @Get('accounts/:id/balance')
  balance(@Param('id') id: string, @Query('currency') currency = 'INR') {
    return this.ledger.balance(id, currency);
  }

  @Get('journal')
  journal(@Query('limit') limit = '25', @Query('offset') offset = '0') {
    return this.ledger.journal(parseInt(limit, 10), parseInt(offset, 10));
  }
}
