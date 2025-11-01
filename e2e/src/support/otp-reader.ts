import fs from 'fs/promises';
import path from 'path';
import { setTimeout as delay } from 'timers/promises';

export type OTPRecord = {
  email: string;
  token: string;
  expires_at: string;
  login_url: string;
};

const DEFAULT_OTP_FILE = path.resolve(__dirname, '../../tmp/passwordless-otp.jsonl');

async function readOtpFile(filePath: string): Promise<OTPRecord[]> {
  try {
    const content = await fs.readFile(filePath, 'utf8');
    return content
      .split('\n')
      .map((line) => line.trim())
      .filter((line) => line.length > 0)
      .map((line) => JSON.parse(line) as OTPRecord);
  } catch (error: unknown) {
    if ((error as NodeJS.ErrnoException).code === 'ENOENT') {
      return [];
    }
    throw error;
  }
}

export async function waitForOtp(
  email: string,
  options: { timeoutMs?: number; pollIntervalMs?: number; otpFile?: string } = {}
): Promise<OTPRecord> {
  const timeoutMs = options.timeoutMs ?? 30_000;
  const pollIntervalMs = options.pollIntervalMs ?? 500;
  const otpFile = options.otpFile ?? process.env.OTP_FILE ?? DEFAULT_OTP_FILE;

  const deadline = Date.now() + timeoutMs;

  while (Date.now() < deadline) {
    const records = await readOtpFile(otpFile);
    const latest = [...records].reverse().find((record) => record.email === email);
    if (latest) {
      return latest;
    }
    await delay(pollIntervalMs);
  }

  throw new Error(`OTP for ${email} not found within ${timeoutMs}ms`);
}

export async function clearOtpFile(filePath?: string): Promise<void> {
  const target = filePath ?? process.env.OTP_FILE ?? DEFAULT_OTP_FILE;
  try {
    await fs.truncate(target, 0);
  } catch (error: unknown) {
    if ((error as NodeJS.ErrnoException).code === 'ENOENT') {
      return;
    }
    throw error;
  }
}
