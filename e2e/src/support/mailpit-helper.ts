import { setTimeout as delay } from 'timers/promises';

// Mailpit API types
export type MailpitMessage = {
  ID: string;
  MessageID: string;
  From: { Name: string; Address: string };
  To: Array<{ Name: string; Address: string }>;
  Subject: string;
  Created: string;
  Size: number;
  Snippet: string;
};

export type MailpitMessageDetail = {
  ID: string;
  MessageID: string;
  From: { Name: string; Address: string };
  To: Array<{ Name: string; Address: string }>;
  Subject: string;
  Date: string;
  Text: string;
  HTML: string;
  Size: number;
};

export type MailpitMessagesResponse = {
  total: number;
  unread: number;
  count: number;
  messages: MailpitMessage[];
};

// Use localhost for local development, mailpit for Docker Compose
const DEFAULT_MAILPIT_URL = process.env.CI || process.env.DOCKER ? 'http://mailpit:8025' : 'http://localhost:8025';

/**
 * Get all messages from Mailpit
 */
export async function getMessages(mailpitUrl?: string): Promise<MailpitMessage[]> {
  const baseUrl = mailpitUrl ?? process.env.MAILPIT_URL ?? DEFAULT_MAILPIT_URL;
  const response = await fetch(`${baseUrl}/api/v1/messages`);
  if (!response.ok) {
    throw new Error(`Failed to get messages: ${response.status} ${response.statusText}`);
  }
  const data = (await response.json()) as MailpitMessagesResponse;
  return data.messages || [];
}

/**
 * Get message detail by ID
 */
export async function getMessage(id: string, mailpitUrl?: string): Promise<MailpitMessageDetail> {
  const baseUrl = mailpitUrl ?? process.env.MAILPIT_URL ?? DEFAULT_MAILPIT_URL;
  const response = await fetch(`${baseUrl}/api/v1/message/${id}`);
  if (!response.ok) {
    throw new Error(`Failed to get message ${id}: ${response.status} ${response.statusText}`);
  }
  return (await response.json()) as MailpitMessageDetail;
}

/**
 * Wait for a message to a specific email address
 */
export async function waitForMessage(
  toEmail: string,
  options: { timeoutMs?: number; pollIntervalMs?: number; mailpitUrl?: string } = {}
): Promise<MailpitMessage> {
  const timeoutMs = options.timeoutMs ?? 30_000;
  const pollIntervalMs = options.pollIntervalMs ?? 500;
  const mailpitUrl = options.mailpitUrl ?? process.env.MAILPIT_URL ?? DEFAULT_MAILPIT_URL;

  const deadline = Date.now() + timeoutMs;

  while (Date.now() < deadline) {
    const messages = await getMessages(mailpitUrl);
    // Find the latest message to the target email
    const found = messages.find((msg) => msg.To.some((to) => to.Address === toEmail));
    if (found) {
      return found;
    }
    await delay(pollIntervalMs);
  }

  throw new Error(`Message to ${toEmail} not found within ${timeoutMs}ms`);
}

/**
 * Extract login URL from message content
 * Looks for URLs in the format: http(s)://.../_auth/email/verify?token=...
 */
export function extractLoginUrl(messageText: string): string | null {
  // Look for URLs with the email verification path
  const urlPattern = /(https?:\/\/[^\s]+\/_auth\/email\/verify\?token=[^\s&]+)/i;
  const match = messageText.match(urlPattern);
  return match ? match[1] : null;
}

/**
 * Wait for an email and extract the login URL
 */
export async function waitForLoginEmail(
  toEmail: string,
  options: { timeoutMs?: number; pollIntervalMs?: number; mailpitUrl?: string } = {}
): Promise<string> {
  const message = await waitForMessage(toEmail, options);
  const detail = await getMessage(message.ID, options.mailpitUrl);

  // Try to extract URL from text content first, then HTML
  let loginUrl = extractLoginUrl(detail.Text);
  if (!loginUrl && detail.HTML) {
    loginUrl = extractLoginUrl(detail.HTML);
  }

  if (!loginUrl) {
    throw new Error(`Login URL not found in email to ${toEmail}`);
  }

  // Rewrite localhost URLs to proxy-app when running in Docker
  // This is needed because emails contain localhost URLs but Playwright in Docker
  // needs to access the proxy via the Docker service name
  if (process.env.BASE_URL) {
    loginUrl = loginUrl.replace(/http:\/\/localhost:4180/g, process.env.BASE_URL);
  }

  return loginUrl;
}

/**
 * Delete all messages in Mailpit
 */
export async function clearAllMessages(mailpitUrl?: string): Promise<void> {
  const baseUrl = mailpitUrl ?? process.env.MAILPIT_URL ?? DEFAULT_MAILPIT_URL;
  const response = await fetch(`${baseUrl}/api/v1/messages`, { method: 'DELETE' });
  if (!response.ok) {
    throw new Error(`Failed to clear messages: ${response.status} ${response.statusText}`);
  }
}
