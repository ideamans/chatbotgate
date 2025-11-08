import { test, expect } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import { execSync } from 'child_process';

// Config file path (relative to test file)
const CONFIG_FILE = path.join(__dirname, '../../config/proxy.e2e.with-whitelist.yaml');

// Helper to read config file
function readConfig(): string {
  return fs.readFileSync(CONFIG_FILE, 'utf-8');
}

// Helper to write config file
function writeConfig(content: string): void {
  fs.writeFileSync(CONFIG_FILE, content, 'utf-8');
}

// Helper to get container logs (with optional timestamp filter)
function getContainerLogs(since?: Date): string {
  try {
    const sinceArg = since ? `--since ${Math.floor(since.getTime() / 1000)}` : '--tail 500';
    return execSync(`docker logs e2e-proxy-app-with-whitelist ${sinceArg} 2>&1`, {
      encoding: 'utf-8',
      cwd: path.join(__dirname, '../..'),
    });
  } catch (error: any) {
    return error.stdout || '';
  }
}

// Helper to wait for log message to appear
async function waitForLogMessage(
  expectedPattern: string | RegExp,
  timeoutMs: number = 5000,
  since?: Date
): Promise<boolean> {
  const startTime = Date.now();
  while (Date.now() - startTime < timeoutMs) {
    const logs = getContainerLogs(since);
    const pattern = typeof expectedPattern === 'string'
      ? new RegExp(expectedPattern.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
      : expectedPattern;
    if (pattern.test(logs)) {
      return true;
    }
    await new Promise(resolve => setTimeout(resolve, 500));
  }
  return false;
}

test.describe('Dynamic Configuration Reload', () => {
  let originalConfig: string;

  test.beforeAll(() => {
    // Save original configuration
    originalConfig = readConfig();
  });

  test.afterAll(() => {
    // Restore original configuration
    writeConfig(originalConfig);
  });

  test('should reload configuration when file changes', async () => {
    // Step 1: Record timestamp before making changes
    const changeTime = new Date();

    // Step 2: Modify configuration (change service description to trigger reload)
    const updatedConfig = originalConfig.replace(
      'E2E verification environment with whitelist',
      'E2E verification environment with whitelist - MODIFIED'
    );
    writeConfig(updatedConfig);

    // Step 3: Wait for reload message to appear in logs
    const hasReloadMessage = await waitForLogMessage(
      'Configuration reloaded successfully',
      5000,
      changeTime
    );

    // Verify that config reload was detected
    expect(hasReloadMessage).toBe(true);

    // Step 4: Restore original config
    const restoreTime = new Date();
    writeConfig(originalConfig);

    // Step 5: Verify another reload happened
    const hasSecondReload = await waitForLogMessage(
      'Configuration reloaded successfully',
      5000,
      restoreTime
    );
    expect(hasSecondReload).toBe(true);
  });

  test('should keep running with old config when invalid YAML is provided', async ({ page }) => {
    // Step 1: Verify server is running with valid config
    await page.goto('http://localhost:4181/_auth/login');
    await expect(page.locator('h1')).toContainText('chatbotgate');

    // Step 2: Record timestamp before making changes
    const changeTime = new Date();

    // Step 3: Write invalid YAML (malformed syntax)
    const invalidYaml = 'invalid: yaml: content:\n  this is: [not valid yaml';
    writeConfig(invalidYaml);

    // Step 4: Wait for error message to appear in logs
    const hasErrorMessage = await waitForLogMessage(
      /(Keeping current|Failed to reload)/,
      5000,
      changeTime
    );

    // Verify that reload was attempted but failed
    expect(hasErrorMessage).toBe(true);

    // Step 5: Verify server is still running with old config
    await page.goto('http://localhost:4181/_auth/login');
    await expect(page.locator('h1')).toContainText('chatbotgate');

    // Step 6: Restore original config
    writeConfig(originalConfig);
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Step 7: Verify server is still accessible after restoration
    await page.goto('http://localhost:4181/_auth/login');
    await expect(page.locator('h1')).toContainText('chatbotgate');
  });

  test('should keep running when config has validation errors', async ({ page }) => {
    // Step 1: Verify server is running
    await page.goto('http://localhost:4181/_auth/login');
    await expect(page.locator('h1')).toContainText('chatbotgate');

    // Step 2: Record timestamp before making changes
    const changeTime = new Date();

    // Step 3: Write config with validation error (empty cookie secret)
    // Match the secret field under session.cookie (it's indented with 4 spaces)
    const invalidConfig = originalConfig.replace(
      /^(\s{4}secret: ).+$/m,
      '$1""  # Empty secret should fail validation'
    );

    // Verify replacement worked
    if (!invalidConfig.includes('secret: ""')) {
      throw new Error('Failed to create invalid config - regex did not match');
    }

    writeConfig(invalidConfig);

    // Step 4: Wait for validation error to appear in logs
    const hasValidationError = await waitForLogMessage(
      /(validation failed|Failed to reload)/,
      5000,
      changeTime
    );

    expect(hasValidationError).toBe(true);

    // Step 5: Verify server still works with old config
    await page.goto('http://localhost:4181/_auth/login');
    await expect(page.locator('h1')).toContainText('chatbotgate');

    // Step 6: Restore original config
    writeConfig(originalConfig);
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Step 7: Verify server is accessible after restoration
    await page.goto('http://localhost:4181/_auth/login');
    await expect(page.locator('h1')).toContainText('chatbotgate');
  });
});
