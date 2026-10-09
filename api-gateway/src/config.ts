// Central place for every environment knob, so nothing reads
// process.env scattered across the codebase.
export const config = {
  port: parseInt(process.env.PORT ?? '3000', 10),
  ledgerCoreUrl: process.env.LEDGER_CORE_URL ?? 'http://localhost:8081',
  jwtSecret: process.env.JWT_SECRET ?? 'dev-secret-change-me',
  jwtExpiresIn: process.env.JWT_EXPIRES_IN ?? '12h',
  adminUser: process.env.LEDGER_ADMIN_USER ?? 'admin',
  // bcrypt hash of "changeme". Override LEDGER_ADMIN_PASSWORD_HASH in prod.
  adminPasswordHash:
    process.env.LEDGER_ADMIN_PASSWORD_HASH ??
    '$2b$10$A2k.HIOCyFbACZA0WQ8P4OUFOdY.0r/Ry5sHWTm98u4O76ZUZ7ylq',
};
