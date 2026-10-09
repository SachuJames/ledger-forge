import { IsIn, IsInt, IsNotEmpty, IsString, Length, Min } from 'class-validator';
import { ApiProperty } from '@nestjs/swagger';

export class CreateAccountDto {
  @ApiProperty({ example: '1001' })
  @IsString()
  @IsNotEmpty()
  code!: string;

  @ApiProperty({ example: 'Operating cash' })
  @IsString()
  @IsNotEmpty()
  name!: string;

  @ApiProperty({ enum: ['asset', 'liability', 'equity', 'revenue', 'expense'] })
  @IsIn(['asset', 'liability', 'equity', 'revenue', 'expense'])
  type!: string;
}

export class TransferDto {
  @ApiProperty({ description: 'Account debited (money in)', example: 'uuid' })
  @IsString()
  @IsNotEmpty()
  debitAccountId!: string;

  @ApiProperty({ description: 'Account credited (money out)', example: 'uuid' })
  @IsString()
  @IsNotEmpty()
  creditAccountId!: string;

  @ApiProperty({ example: 'INR' })
  @IsString()
  @Length(3, 3)
  currency!: string;

  @ApiProperty({
    description: 'Amount in minor units (paise/cents). 2500 INR = Rs 25.00',
    example: 2500,
  })
  @IsInt()
  @Min(1)
  amount!: number;

  @ApiProperty({ required: false, example: 'Invoice #4821 settlement' })
  @IsString()
  narration?: string;

  @ApiProperty({
    required: false,
    description: 'Client-supplied idempotency key. Retrying with the same key never double-posts.',
  })
  @IsString()
  idempotencyKey?: string;
}
