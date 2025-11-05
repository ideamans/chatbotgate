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

// Helper to get container logs
function getContainerLogs(): string {
  try {
    return execSync('docker logs e2e-proxy-app-with-whitelist --tail 200 2>&1', {
      encoding: 'utf-8',
      cwd: path.join(__dirname, '../..'),
    });
  } catch (error: any) {
    return error.stdout || '';
  }
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
    // Step 1: Get baseline logs
    const logsBefore = getContainerLogs();
    const reloadCountBefore = (logsBefore.match(/Configuration reloaded successfully/g) || []).length;

    // Step 2: Modify configuration (change service description to trigger reload)
    const updatedConfig = originalConfig.replace(
      'E2E verification environment with whitelist',
      'E2E verification environment with whitelist - MODIFIED'
    );
    writeConfig(updatedConfig);

    // Step 3: Wait for config watcher to detect and reload (fsnotify is fast, wait 2s to be safe)
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Step 4: Check logs for reload message
    const logsAfter = getContainerLogs();
    const reloadCountAfter = (logsAfter.match(/Configuration reloaded successfully/g) || []).length;
    const hasReloadMessage = logsAfter.includes('Config content change detected, starting reload');

    // Verify that config reload was detected
    expect(hasReloadMessage).toBe(true);
    expect(reloadCountAfter).toBeGreaterThan(reloadCountBefore);

    // Step 5: Restore original config
    writeConfig(originalConfig);
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Step 6: Verify another reload happened
    const logsFinal = getContainerLogs();
    const reloadCountFinal = (logsFinal.match(/Configuration reloaded successfully/g) || []).length;
    expect(reloadCountFinal).toBeGreaterThan(reloadCountAfter);
  });

  test('should keep running with old config when invalid YAML is provided', async ({ page }) => {
    // Step 1: Verify server is running with valid config
    await page.goto('http://localhost:4181/_auth/login');
    await expect(page.locator('h1')).toContainText('chatbotgate');

    // Step 2: Write invalid YAML (malformed syntax)
    const invalidYaml = 'invalid: yaml: content:\n  this is: [not valid yaml';
    writeConfig(invalidYaml);

    // Step 3: Wait for watcher to detect change
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Step 4: Get logs and check for error message (not reload success)
    const logsAfter = getContainerLogs();
    const hasErrorMessage = logsAfter.includes('Keeping current') || logsAfter.includes('Failed to reload');

    // Verify that reload was attempted but failed
    expect(hasErrorMessage).toBe(true);

    // Step 5: Verify server is still running with old config
    await page.goto('http://localhost:4181/_auth/login');
    await expect(page.locator('h1')).toContainText('chatbotgate');

    // Step 5: Restore original config
    writeConfig(originalConfig);
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Step 6: Verify server is still accessible after restoration
    await page.goto('http://localhost:4181/_auth/login');
    await expect(page.locator('h1')).toContainText('chatbotgate');
  });

  test('should keep running when config has validation errors', async ({ page }) => {
    // Step 1: Verify server is running
    await page.goto('http://localhost:4181/_auth/login');
    await expect(page.locator('h1')).toContainText('chatbotgate');

    // Step 2: Write config with validation error (missing required field)
    const invalidConfig = originalConfig.replace(
      /cookie_secret: .+/,
      'cookie_secret: ""  # Empty secret should fail validation'
    );
    writeConfig(invalidConfig);

    // Step 3: Wait for watcher to detect
    await new Promise(resolve => setTimeout(resolve, 2000));

    // Step 4: Check that reload failed
    const logsAfter = getContainerLogs();
    const hasValidationError = logsAfter.includes('validation failed') || logsAfter.includes('Failed to reload');

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
